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
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/golang/glog"
	"github.com/optiopay/kafka"
	"github.com/optiopay/kafka/proto"
	"k8s.io/heapster/extpoints"
	sink_api "k8s.io/heapster/sinks/api"
	kube_api "k8s.io/kubernetes/pkg/api"
)

const (
	partition                = 0
	brokerClientID           = "kafka-sink"
	brokerDialTimeout        = 10 * time.Second
	brokerDialRetryLimit     = 10
	brokerDialRetryWait      = 500 * time.Millisecond
	brokerAllowTopicCreation = true
	brokerLeaderRetryLimit   = 10
	brokerLeaderRetryWait    = 500 * time.Millisecond
)

type kafkaSink struct {
	producer        kafka.Producer
	timeSeriesTopic string
	eventsTopic     string
	sinkBrokerHosts []string
}

// START: ExternalSink interface implementations

func (self *kafkaSink) Register(mds []sink_api.MetricDescriptor) error {
	return nil
}

func (self *kafkaSink) Unregister(mds []sink_api.MetricDescriptor) error {
	return nil
}

func (self *kafkaSink) StoreTimeseries(timeseries []sink_api.Timeseries) error {
	if timeseries == nil || len(timeseries) <= 0 {
		return nil
	}
	for _, t := range timeseries {
		err := self.produceKafkaMessage(t, self.timeSeriesTopic)
		if err != nil {
			return fmt.Errorf("failed to produce Kafka messages: %s", err)
		}
	}
	return nil
}

func (self *kafkaSink) StoreEvents(events []kube_api.Event) error {
	if events == nil || len(events) <= 0 {
		return nil
	}
	for _, event := range events {
		err := self.produceKafkaMessage(event, self.eventsTopic)
		if err != nil {
			return fmt.Errorf("failed to produce Kafka messages: %s", err)
		}
	}
	return nil
}

// produceKafkaMessage produces messages to kafka
func (self *kafkaSink) produceKafkaMessage(v interface{}, topic string) error {
	if v == nil {
		return nil
	}
	jsonItems, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to transform the items to json : %s", err)
	}
	message := &proto.Message{Value: []byte(string(jsonItems))}
	_, err = self.producer.Produce(topic, partition, message)
	if err != nil {
		return fmt.Errorf("failed to produce message to %s:%d: %s", topic, partition, err)
	}
	return nil
}

func (self *kafkaSink) DebugInfo() string {
	info := fmt.Sprintf("%s\n", self.Name())
	info += fmt.Sprintf("There are two kafka's topics: %s,%s:\n", self.eventsTopic, self.timeSeriesTopic)
	info += fmt.Sprintf("The kafka's broker list is: %s", self.sinkBrokerHosts)
	return info
}

func (self *kafkaSink) Name() string {
	return "Apache-Kafka Sink"
}

func init() {
	extpoints.SinkFactories.Register(NewKafkaSink, "kafka")
}

func NewKafkaSink(uri *url.URL, _ extpoints.HeapsterConf) ([]sink_api.ExternalSink, error) {
	var kafkaSink kafkaSink
	opts, err := url.ParseQuery(uri.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to parser url's query string: %s", err)
	}

	if len(opts["timeseriestopic"]) < 1 {
		return nil, fmt.Errorf("There is no timeseriestopic assign for config kafka-sink")
	}
	kafkaSink.timeSeriesTopic = opts["eventstopic"][0]

	if len(opts["eventstopic"]) < 1 {
		return nil, fmt.Errorf("There is no eventstopic assign for config kafka-sink")
	}
	kafkaSink.eventsTopic = opts["eventstopic"][0]

	if len(opts["brokers"]) < 1 {
		return nil, fmt.Errorf("There is no broker assign for connecting kafka broker")
	}
	kafkaSink.sinkBrokerHosts = append(kafkaSink.sinkBrokerHosts, opts["brokers"]...)

	//connect to kafka cluster
	brokerConf := kafka.NewBrokerConf(brokerClientID)
	brokerConf.DialTimeout = brokerDialTimeout
	brokerConf.DialRetryLimit = brokerDialRetryLimit
	brokerConf.DialRetryWait = brokerDialRetryWait
	brokerConf.LeaderRetryLimit = brokerLeaderRetryLimit
	brokerConf.LeaderRetryWait = brokerLeaderRetryWait
	brokerConf.AllowTopicCreation = true

	broker, err := kafka.Dial(kafkaSink.sinkBrokerHosts, brokerConf)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to kafka cluster: %s", err)
	}
	defer broker.Close()

	//create kafka producer
	conf := kafka.NewProducerConf()
	conf.RequiredAcks = proto.RequiredAcksLocal
	sinkProducer := broker.Producer(conf)
	kafkaSink.producer = sinkProducer
	glog.Infof("created kafka sink successfully with brokers: %v", kafkaSink.sinkBrokerHosts)
	return []sink_api.ExternalSink{&kafkaSink}, nil
}
