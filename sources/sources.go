package sources

import "time"

type Source interface {
	FetchData() error
	GetAndFlushData() ([]DataEntry, error)
}

type DataEntry struct {
	Time time.Time
	Data map[string]string
}

