package kafkatest

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/optiopay/kafka"
	"github.com/optiopay/kafka/proto"
)

var (
	ErrTimeout = errors.New("timeout")

	ErrNotImplemented = errors.New("not implemented")

	// test implementation should implement the interface
	_ kafka.Client   = &Broker{}
	_ kafka.Producer = &Producer{}
	_ kafka.Consumer = &Consumer{}
)

// Broker is mock version of kafka's broker. It's implementing Broker interface
// and provides easy way of mocking server actions.
type Broker struct {
	produced chan *ProducedMessages

	mu        sync.Mutex
	consumers map[string]map[int32]*Consumer

	// OffsetEarliestHandler is callback function called whenever
	// OffsetEarliest method of the broker is called. Overwrite to change
	// default behaviour -- always returning ErrUnknownTopicOrPartition
	OffsetEarliestHandler func(string, int32) (int64, error)

	// OffsetLatestHandler is callback function called whenever OffsetLatest
	// method of the broker is called. Overwrite to change default behaviour --
	// always returning ErrUnknownTopicOrPartition
	OffsetLatestHandler func(string, int32) (int64, error)
}

func NewBroker() *Broker {
	return &Broker{
		consumers: make(map[string]map[int32]*Consumer),
		produced:  make(chan *ProducedMessages),
	}
}

// Close is no operation method, required by Broker interface.
func (b *Broker) Close() {
}

// OffsetEarliest return result of OffsetEarliestHandler callback set on the
// broker. If not set, always return ErrUnknownTopicOrPartition
func (b *Broker) OffsetEarliest(topic string, partition int32) (int64, error) {
	if b.OffsetEarliestHandler != nil {
		return b.OffsetEarliestHandler(topic, partition)
	}
	return 0, proto.ErrUnknownTopicOrPartition
}

// OffsetLatest return result of OffsetLatestHandler callback set on the
// broker. If not set, always return ErrUnknownTopicOrPartition
func (b *Broker) OffsetLatest(topic string, partition int32) (int64, error) {
	if b.OffsetLatestHandler != nil {
		return b.OffsetLatestHandler(topic, partition)
	}
	return 0, proto.ErrUnknownTopicOrPartition
}

// Consumer returns consumer mock and never error.
//
// At most one consumer for every topic-partition pair can be created --
// calling this for the same topic-partition will always return the same
// consumer instance.
func (b *Broker) Consumer(conf kafka.ConsumerConf) (kafka.Consumer, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if t, ok := b.consumers[conf.Topic]; ok {
		if c, ok := t[conf.Partition]; ok {
			return c, nil
		}
	} else {
		b.consumers[conf.Topic] = make(map[int32]*Consumer)
	}

	c := &Consumer{
		conf:     conf,
		Broker:   b,
		Messages: make(chan *proto.Message),
		Errors:   make(chan error),
	}
	b.consumers[conf.Topic][conf.Partition] = c
	return c, nil
}

// Producer returns producer mock instance.
func (b *Broker) Producer(kafka.ProducerConf) kafka.Producer {
	return &Producer{
		Broker:         b,
		ResponseOffset: 1,
	}
}

// OffsetCoordinator returns offset coordinator mock instance. It's always
// successful, so you can always ignore returned error.
func (b *Broker) OffsetCoordinator(conf kafka.OffsetCoordinatorConf) (kafka.OffsetCoordinator, error) {
	c := &OffsetCoordinator{
		Broker: b,
		conf:   conf,
	}
	return c, nil
}

// ReadProducers return ProduceMessages representing produce call of one of
// created by broker producers or ErrTimeout.
func (b *Broker) ReadProducers(timeout time.Duration) (*ProducedMessages, error) {
	select {
	case p := <-b.produced:
		return p, nil
	case <-time.After(timeout):
		return nil, ErrTimeout
	}
}

// Consumer mocks kafka's consumer. Use Messages and Errors channels to mock
// Consume method results.
type Consumer struct {
	conf kafka.ConsumerConf

	Broker *Broker

	// Messages is channel consumed by fetch method call. Pushing message into
	// this channel will result in Consume method call returning message data.
	Messages chan *proto.Message

	// Errors is channel consumed by fetch method call. Pushing error into this
	// channel will result in Consume method call returning error.
	Errors chan error
}

// Consume returns message or error pushed through consumers Messages and Errors
// channel. Function call will block until data on at least one of those
// channels is available.
func (c *Consumer) Consume() (*proto.Message, error) {
	select {
	case msg := <-c.Messages:
		msg.Topic = c.conf.Topic
		msg.Partition = c.conf.Partition
		return msg, nil
	case err := <-c.Errors:
		return nil, err
	}
}

// Producer mocks kafka's producer.
type Producer struct {
	Broker *Broker

	// ResponseOffset is offset counter returned and incremented by every
	// Produce method call. By default set to 1.
	ResponseOffset int64

	// ResponseError if set, force Produce method call to instantly return
	// error, without publishing messages. By default nil.
	ResponseError error
}

// ProducedMessages represents all arguments used for single Produce method
// call.
type ProducedMessages struct {
	Topic     string
	Partition int32
	Messages  []*proto.Message
}

// Produce is settings messages Crc and Offset attributes and pushing all
// passed arguments to broker. Produce call is blocking until pushed message
// will be read with broker's ReadProduces.
func (p *Producer) Produce(topic string, partition int32, messages ...*proto.Message) (int64, error) {
	if p.ResponseError != nil {
		return 0, p.ResponseError
	}
	off := p.ResponseOffset

	for i, msg := range messages {
		msg.Offset = off + int64(i)
		msg.Crc = proto.ComputeCrc(msg, proto.CompressionNone)
	}

	p.Broker.produced <- &ProducedMessages{
		Topic:     topic,
		Partition: partition,
		Messages:  messages,
	}
	p.ResponseOffset += int64(len(messages))
	return off, nil
}

type OffsetCoordinator struct {
	conf   kafka.OffsetCoordinatorConf
	Broker *Broker

	// Offsets is used to store all offset commits when using mocked
	// coordinator's default behaviour.
	Offsets map[string]int64

	// CommitHandler is callback function called whenever Commit method of the
	// OffsetCoordinator is called. If CommitHandler is nil, Commit method will
	// return data using Offset attribute as store.
	CommitHandler func(consumerGroup string, topic string, partition int32, offset int64) error

	// OffsetHandler is callback function called whenever Offset method of the
	// OffsetCoordinator is called. If OffsetHandler is nil, Commit method will
	// use Offset attribute to retrieve the offset.
	OffsetHandler func(consumerGroup string, topic string, partition int32) (offset int64, metadata string, err error)
}

// Commit return result of CommitHandler callback set on coordinator. If
// handler is nil, this method will use Offsets attribute to store data for
// further use.
func (c *OffsetCoordinator) Commit(topic string, partition int32, offset int64) error {
	if c.CommitHandler != nil {
		return c.CommitHandler(c.conf.ConsumerGroup, topic, partition, offset)
	}
	c.Offsets[fmt.Sprintf("%s:%d", topic, partition)] = offset
	return nil
}

// Offset return result of OffsetHandler callback set on coordinator. If
// handler is nil, this method will use Offsets attribute to retrieve committed
// offset. If no offset for given topic and partition pair was saved,
// proto.ErrUnknownTopicOrPartition is returned.
func (c *OffsetCoordinator) Offset(topic string, partition int32) (offset int64, metadata string, err error) {
	if c.OffsetHandler != nil {
		return c.OffsetHandler(c.conf.ConsumerGroup, topic, partition)
	}
	off, ok := c.Offsets[fmt.Sprintf("%s:%d", topic, partition)]
	if !ok {
		return 0, "", proto.ErrUnknownTopicOrPartition
	}
	return off, "", nil
}
