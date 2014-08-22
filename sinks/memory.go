package sinks

import (
	"container/list"
	"flag"
	"fmt"
	"time"

	"github.com/vishh/caggregator/sources"
)

var argMaxStorageDuration = flag.Duration("sink_memory_ttl", 1*time.Hour, "Time duration for which stats should be cached if the memory sink is used")

type MemorySink struct {
	containersData     *list.List
	oldestData         time.Time
	maxStorageDuration time.Duration
}

type entry struct {
	timestamp time.Time
	data      interface{}
}

func (self *MemorySink) reapOldData() {
	fmt.Printf("%+v\n", self.containersData.Front().Value)
	if self.containersData.Len() == 0 || time.Since(self.oldestData) < self.maxStorageDuration {
		return
	}
}

func (self *MemorySink) StoreData(data Data) error {
	if data, ok := data.([]sources.Pod); ok {
		for _, value := range data {
			self.containersData.PushFront(entry{time.Now(), value})
			if self.containersData.Len() == 1 {
				self.oldestData = time.Now()
			}
		}
	}
	self.reapOldData()
	return nil
}

func NewMemorySink() Sink {
	return &MemorySink{
		containersData:     list.New(),
		oldestData:         time.Now(),
		maxStorageDuration: *argMaxStorageDuration,
	}
}
