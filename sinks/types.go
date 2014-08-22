package sinks

import (
	"flag"
	"fmt"
	"time"
)

var argSink = flag.String("sink", "memory", "Backend storage. Options are [memory | influxdb]")

type Data interface{}

type Sink interface {
	StoreData(data Data) error
	RetrieveData(from, to time.Time) (Data, error)
}

func NewSink() (Sink, error) {
	switch *argSink {
	case "memory":
		return nil, nil
	case "influxdb":
		return nil, nil
	default:
		return nil, fmt.Errorf("Invalid sink specified - %s", *argSink)
	}
}
