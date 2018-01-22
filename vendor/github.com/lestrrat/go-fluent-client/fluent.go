// Package fluent implements a client for the fluentd data logging daemon.
package fluent

// New creates a new client. By default a buffered client is created.
// The `WithBufered` option switches which type of client is created.
// `WithBuffered(true)` (default) creates a buffered client, and
// `WithBuffered(false)` creates a unbuffered client.
// All options are delegates to `NewBuffered` and `NewUnbuffered`
// respectively.
func New(options ...Option) (Client, error) {
	var buffered = true
	for _, opt := range options {
		switch opt.Name() {
		case optkeyBuffered:
			buffered = opt.Value().(bool)
		}
	}

	if buffered {
		return NewBuffered(options...)
	}
	return NewUnbuffered(options...)
}
