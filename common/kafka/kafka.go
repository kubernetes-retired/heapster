// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"
	"time"

	kafka "github.com/Shopify/sarama"
	"github.com/golang/glog"
)

const (
	brokerClientID         = "kafka-sink"
	brokerDialTimeout      = 10 * time.Second
	brokerDialRetryLimit   = 1
	brokerDialRetryWait    = 0
	brokerLeaderRetryLimit = 1
	brokerLeaderRetryWait  = 0
	metricsTopic           = "heapster-metrics"
	eventsTopic            = "heapster-events"
)

const (
	TimeSeriesTopic = "timeseriestopic"
	EventsTopic     = "eventstopic"
)

type KafkaClient interface {
	Name() string
	Stop()
	ProduceKafkaMessage(msgData interface{}) error
}

type kafkaSink struct {
	producer  kafka.SyncProducer
	dataTopic string
}

func (sink *kafkaSink) ProduceKafkaMessage(msgData interface{}) error {
	start := time.Now()
	msgJson, err := json.Marshal(msgData)
	if err != nil {
		return fmt.Errorf("failed to transform the items to json : %s", err)
	}

	_, _, err = sink.producer.SendMessage(&kafka.ProducerMessage{
		Topic: sink.dataTopic,
		Key:   nil,
		Value: kafka.ByteEncoder(msgJson),
	})
	if err != nil {
		return fmt.Errorf("failed to produce message to %s: %s", sink.dataTopic, err)
	}
	end := time.Now()
	glog.V(4).Infof("Exported %d data to kafka in %s", len(msgJson), end.Sub(start))
	return nil
}

func (sink *kafkaSink) Name() string {
	return "Apache Kafka Sink"
}

func (sink *kafkaSink) Stop() {
	sink.producer.Close()
}

func getTopic(opts map[string][]string, topicType string) (string, error) {
	var topic string
	switch topicType {
	case TimeSeriesTopic:
		topic = metricsTopic
	case EventsTopic:
		topic = eventsTopic
	default:
		return "", fmt.Errorf("Topic type '%s' is illegal.", topicType)
	}

	if len(opts[topicType]) > 0 {
		topic = opts[topicType][0]
	}

	return topic, nil
}

func getCompression(opts url.Values) (kafka.CompressionCodec, error) {
	if len(opts["compression"]) == 0 {
		return kafka.CompressionNone, nil
	}
	comp := opts["compression"][0]
	switch comp {
	case "none":
		return kafka.CompressionNone, nil
	case "gzip":
		return kafka.CompressionGZIP, nil
	case "snappy":
		return kafka.CompressionSnappy, nil
	case "lz4":
		return kafka.CompressionLZ4, nil
	default:
		return kafka.CompressionNone, fmt.Errorf("Compression '%s' is illegal. Use none, snappy, lz4 or gzip", comp)
	}
}

func getTlsConfiguration(opts url.Values) (*tls.Config, bool, error) {
	if len(opts["cacert"]) == 0 &&
		(len(opts["cert"]) == 0 || len(opts["key"]) == 0) {
		return nil, false, nil
	}
	t := &tls.Config{}
	if len(opts["cacert"]) != 0 {
		caFile := opts["cacert"][0]
		caCert, err := ioutil.ReadFile(caFile)
		if err != nil {
			return nil, false, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		t.RootCAs = caCertPool
	}

	if len(opts["cert"]) != 0 && len(opts["key"]) != 0 {
		certFile := opts["cert"][0]
		keyFile := opts["key"][0]
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, false, err
		}
		t.Certificates = []tls.Certificate{cert}
	}
	if len(opts["insecuressl"]) != 0 {
		insecuressl := opts["insecuressl"][0]
		insecure, err := strconv.ParseBool(insecuressl)
		if err != nil {
			return nil, false, err
		}
		t.InsecureSkipVerify = insecure
	}

	return t, true, nil
}

func getSASLConfiguration(opts url.Values) (string, string, bool, error) {
	if len(opts["user"]) == 0 {
		return "", "", false, nil
	}
	user := opts["user"][0]
	if len(opts["password"]) == 0 {
		return "", "", false, nil
	}
	password := opts["password"][0]
	return user, password, true, nil
}

func getOptionsWithoutSecrets(values url.Values) string {
	var password []string
	if len(values["password"]) != 0 {
		password = values["password"]
		values["password"] = []string{"***"}
		defer func() { values["password"] = password }()
	}
	options := fmt.Sprintf("kafka sink option: %v", values)
	return options
}

func NewKafkaClient(uri *url.URL, topicType string) (KafkaClient, error) {
	opts, err := url.ParseQuery(uri.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url's query string: %s", err)
	}
	glog.V(3).Info(getOptionsWithoutSecrets(opts))

	topic, err := getTopic(opts, topicType)
	if err != nil {
		return nil, err
	}

	compression, err := getCompression(opts)
	if err != nil {
		return nil, err
	}

	var kafkaBrokers []string
	if len(opts["brokers"]) < 1 {
		return nil, fmt.Errorf("There is no broker assigned for connecting kafka")
	}
	kafkaBrokers = append(kafkaBrokers, opts["brokers"]...)
	glog.V(2).Infof("initializing kafka sink with brokers - %v", kafkaBrokers)

	kafka.Logger = GologAdapterLogger{}

	//structure the config of broker
	config := kafka.NewConfig()
	config.ClientID = brokerClientID
	config.Net.DialTimeout = brokerDialTimeout
	config.Metadata.Retry.Max = brokerDialRetryLimit
	config.Metadata.Retry.Backoff = brokerDialRetryWait
	config.Producer.Retry.Max = brokerLeaderRetryLimit
	config.Producer.Retry.Backoff = brokerLeaderRetryWait
	config.Producer.Compression = compression
	config.Producer.Partitioner = kafka.NewRoundRobinPartitioner
	config.Producer.RequiredAcks = kafka.WaitForLocal
	config.Producer.Return.Errors = true
	config.Producer.Return.Successes = true

	config.Net.TLS.Config, config.Net.TLS.Enable, err = getTlsConfiguration(opts)
	if err != nil {
		return nil, err
	}

	config.Net.SASL.User, config.Net.SASL.Password, config.Net.SASL.Enable, err = getSASLConfiguration(opts)
	if err != nil {
		return nil, err
	}

	// set up producer of kafka server.
	glog.V(3).Infof("attempting to setup kafka sink")
	sinkProducer, err := kafka.NewSyncProducer(kafkaBrokers, config)
	if err != nil {
		return nil, fmt.Errorf("Failed to setup Producer: - %v", err)
	}

	glog.V(3).Infof("kafka sink setup successfully")
	return &kafkaSink{
		producer:  sinkProducer,
		dataTopic: topic,
	}, nil
}
