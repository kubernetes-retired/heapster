package fluent

import (
	"context"
	"time"
)

// Ping is a helper method that allows you to call client.Ping in a periodic
// manner.
//
// By default a ping message will be sent every 5 minutes. You may change this
// using the WithPingInterval option.
//
// If you need to capture ping failures, pass it a channel using WithPingResultChan
func Ping(ctx context.Context, client Client, tag string, record interface{}, options ...Option) {
	var interval = 5 * time.Minute
	var replyCh chan error

	for _, option := range options {
		switch option.Name() {
		case optkeyPingInterval:
			interval = option.Value().(time.Duration)
		case optkeyPingResultChan:
			replyCh = option.Value().(chan error)
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := client.Ping(tag, record, options...)
			if err != nil && replyCh != nil {
				replyCh <- err
			}
		}
	}
}
