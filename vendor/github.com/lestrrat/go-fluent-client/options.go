package fluent

import (
	"context"
	"time"
)

type option struct {
	name  string
	value interface{}
}

func (o *option) Name() string {
	return o.name
}

func (o *option) Value() interface{} {
	return o.value
}

// WithBuffered specifies if we should create a buffered
// or unbuffered client
func WithBuffered(b bool) Option {
	return &option{
		name:  optkeyBuffered,
		value: b,
	}
}

// WithNetwork specifies the network type, i.e. "tcp" or "unix"
// for `fluent.New`
func WithNetwork(s string) Option {
	return &option{
		name:  optkeyNetwork,
		value: s,
	}
}

// WithAddress specifies the address to connect to for `fluent.New`
// A unix domain socket path, or a hostname/IP address.
func WithAddress(s string) Option {
	return &option{
		name:  optkeyAddress,
		value: s,
	}
}

// WithTimestamp specifies the timestamp to be used for `Client.Post`
func WithTimestamp(t time.Time) Option {
	return &option{
		name:  optkeyTimestamp,
		value: t,
	}
}

// WithJSONMarshaler specifies JSON marshaling to be used when
// sending messages to fluentd. Used for `fluent.New`
func WithJSONMarshaler() Option {
	return &option{
		name:  optkeyMarshaler,
		value: marshalFunc(jsonMarshal),
	}
}

// WithMsgpackMarshaler specifies msgpack marshaling to be used when
// sending messages to fluentd. Used in `fluent.New`
func WithMsgpackMarshaler() Option {
	return &option{
		name:  optkeyMarshaler,
		value: marshalFunc(msgpackMarshal),
	}
}

// WithTagPrefix specifies the prefix to be appended to tag names
// when sending messages to fluend. Used in `fluent.New`
func WithTagPrefix(s string) Option {
	return &option{
		name:  optkeyTagPrefix,
		value: s,
	}
}

// WithSyncAppend specifies if we should synchronously check for
// success when appending to the underlying pending buffer.
// Used in `Client.Post`. If not specified, errors appending
// are not reported.
func WithSyncAppend(b bool) Option {
	return &option{
		name:  optkeySyncAppend,
		value: b,
	}
}

// WithBufferLimit specifies the buffer limit to be used for
// the underlying pending buffer. If a `Client.Post` operation
// would exceed this size, an error is returned (note: you must
// use `WithSyncAppend` in `Client.Post` if you want this error
// to be reported). The defalut value is 8MB
func WithBufferLimit(v interface{}) Option {
	return &option{
		name:  optkeyBufferLimit,
		value: v,
	}
}

// WithWriteThreshold specifies the minimum number of bytes that we
// should have pending before starting to attempt to write to the
// server. The default value is 8KB
func WithWriteThreshold(i int) Option {
	return &option{
		name:  optkeyWriteThreshold,
		value: i,
	}
}

// WithSubsecond specifies if we should use EventTime for timestamps
// on fluentd messages. May be used on a per-client basis or per-call
// to Post(). By default this feature is turned OFF.
//
// Note that this option will only work for fluentd v0.14 or above.
func WithSubsecond(b bool) Option {
	return &option{
		name:  optkeySubSecond,
		value: b,
	}
}

// WithContext specifies the context.Context object to be used by Post().
// Possible blocking operations are (1) writing to the background buffer,
// and (2) waiting for a reply from when WithSyncAppend(true) is in use.
func WithContext(ctx context.Context) Option {
	return &option{
		name:  optkeyContext,
		value: ctx,
	}
}

// WithMaxConnAttempts specifies the maximum number of attempts made by
// the client to connect to the fluentd server during final data flushing
// for buffered clients. For unbuffered clients, this controls the number
// of attempts made when calling `Post`.
//
// For buffered clients: During normal operation, the client will indefinitely
// attempt to connect to the server (whilst being backed-off properly), as
// it should try as hard as possible to send the stored data.
//
// This option controls the behavior when the client still has more data to
// send AFTER it has been told to Close() or Shutdown(). In this case we know
// the client wants to stop at some point, so we try to connect up to a
// finite number of attempts.
//
// The default value is 64 for both buffered and unbuffered clients.
func WithMaxConnAttempts(n uint64) Option {
	return &option{
		name:  optkeyMaxConnAttempts,
		value: n,
	}
}

// WithDialTimeout specifies the amount of time allowed for the client to
// establish connection with the server. If we are forced to wait for a
// duration that exceeds the specified timeout, we deem the connection to
// have failed. The default value is 3 seconds
func WithDialTimeout(d time.Duration) Option {
	return &option{
		name:  optkeyDialTimeout,
		value: d,
	}
}

// WithWriteQueueSize specifies the channel buffer size for the queue
// used to pass messages from the Client to the background writer
// goroutines. The default value is 64.
func WithWriteQueueSize(n int) Option {
	return &option{
		name:  optkeyWriteQueueSize,
		value: n,
	}
}

// WithConnectOnStart is specified when you would like a buffered client
// to make sure that it can connect to the specified fluentd server on
// startup.
func WithConnectOnStart(b bool) Option {
	return &option{
		name:  optkeyConnectOnStart,
		value: b,
	}
}

// WithPingInterval is used in the fluent.Ping method to specify the time
// between pings. The default value is 5 minutes
func WithPingInterval(t time.Duration) Option {
	return &option{
		name:  optkeyPingInterval,
		value: t,
	}
}

// WithPingResultChan specifies the channel where you will receive ping failures
func WithPingResultChan(ch chan error) Option {
	return &option{
		name:  optkeyPingResultChan,
		value: ch,
	}
}
