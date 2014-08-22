package sinks

import (
	"container/list"
	"flag"
	"time"

	"github.com/vishh/caggregator/sources"
)

var argMaxStorageDuration = flag.Duration("sink_memory_ttl", 12*time.Hour, "Time duration for which stats should be cached if the memory sink is used")

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
	if self.containersData.Len() == 0 || time.Since(self.oldestData) < self.maxStorageDuration {
		return
	}
	element := self.containersData.Back()
	for element != nil {
		// TODO(vishh): Implement reaping of old data.
		return
	}
}

func (self *MemorySink) StoreData(data Data) error {
	if data, ok := data.([]sources.Pod); ok {
		for _, value := range data {
			self.containersData.PushFront(entry{timestamp: time.Now(), data: value})
			if self.containersData.Len() == 1 {
				self.oldestData = time.Now()
			}
		}
	}
	self.reapOldData()
	return nil
}

func (self *MemorySink) RetrieveData(from, to time.Time, data Data) error {
	// TODO(vishh): Implement this.
	return nil
}

func NewMemorySink() Sink {
	return &MemorySink{
		containersData:     list.New(),
		oldestData:         time.Now(),
		maxStorageDuration: *argMaxStorageDuration,
	}
}
