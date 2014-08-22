package sinks

import (
	"flag"
	"fmt"
)

var argSink = flag.String("sink", "memory", "Backend storage. Options are [memory | influxdb]")

type Data interface{}

type Sink interface {
	StoreData(data Data) error
}

func NewSink() (Sink, error) {
	switch *argSink {
	case "memory":
		return NewMemorySink(), nil
	case "influxdb":
		return NewInfluxdbSink()
	default:
		return nil, fmt.Errorf("Invalid sink specified - %s", *argSink)
	}
}
