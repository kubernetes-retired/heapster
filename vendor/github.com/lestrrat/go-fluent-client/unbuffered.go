package fluent

import (
	"context"
	"io"
	"net"
	"time"

	pdebug "github.com/lestrrat/go-pdebug"
	"github.com/pkg/errors"
)

// NewUnbuffered creates an unbuffered client. Unlike the normal
// buffered client, an unbuffered client handles the Post() method
// synchronously, and does not attempt to buffer the payload.
//
//    * fluent.WithAddress
//    * fluent.WithDialTimeout
//    * fluent.WithMarshaler
//    * fluent.WithMaxConnAttempts
//    * fluent.WithNetwork
//    * fluent.WithSubSecond
//    * fluent.WithTagPrefix
//
// Please see their respective documentation for details.
func NewUnbuffered(options ...Option) (client *Unbuffered, err error) {
	if pdebug.Enabled {
		g := pdebug.Marker("fluent.NewUnbuffered").BindError(&err)
		defer g.End()
	}

	var c = &Unbuffered{
		address:         "127.0.0.1:24224",
		dialTimeout:     3 * time.Second,
		maxConnAttempts: 64,
		marshaler:       marshalFunc(msgpackMarshal),
		network:         "tcp",
		writeTimeout:    3 * time.Second,
	}

	var connectOnStart bool
	for _, opt := range options {
		switch opt.Name() {
		case optkeyAddress:
			c.address = opt.Value().(string)
		case optkeyDialTimeout:
			c.dialTimeout = opt.Value().(time.Duration)
		case optkeyMarshaler:
			c.marshaler = opt.Value().(marshaler)
		case optkeyMaxConnAttempts:
			c.maxConnAttempts = opt.Value().(uint64)
		case optkeyNetwork:
			v := opt.Value().(string)
			switch v {
			case "tcp", "unix":
			default:
				return nil, errors.Errorf(`invalid network type: %s`, v)
			}
			c.network = v
		case optkeySubSecond:
			c.subsecond = opt.Value().(bool)
		case optkeyTagPrefix:
			c.tagPrefix = opt.Value().(string)
		case optkeyConnectOnStart:
			connectOnStart = opt.Value().(bool)
		}
	}

	if connectOnStart {
		if _, err := c.connect(true); err != nil {
			return nil, errors.Wrap(err, `failed to connect on start`)
		}
	}

	return c, nil
}

// Close cloes the currenct cached connection, if any
func (c *Unbuffered) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return nil
	}
	c.conn.Close()
	c.conn = nil
	return nil
}

// Shutdown is an alias to Close(). Since an unbuffered
// Client does not have any pending buffers at any given moment,
// we do not have to do anything other than close
func (c *Unbuffered) Shutdown(_ context.Context) error {
	return c.Close()
}

func (c *Unbuffered) connect(force bool) (net.Conn, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		if !force {
			return c.conn, nil
		}
		c.conn.Close()
	}

	conn, err := dial(context.Background(), c.network, c.address, c.dialTimeout)
	if err != nil {
		return nil, err
	}

	c.conn = conn
	return conn, nil
}

// Post posts the given structure after encoding it along with the given tag.
//
// If you would like to specify options to `Post()`, you may pass them at the
// end of the method. Currently you can use the following:
//
//   fluent.WithTimestamp: allows you to set arbitrary timestamp values
//
func (c *Unbuffered) Post(tag string, v interface{}, options ...Option) (err error) {
	if pdebug.Enabled {
		g := pdebug.Marker("fluent.Unbuffered.Post").BindError(&err)
		defer g.End()
	}

	var t time.Time
	for _, opt := range options {
		switch opt.Name() {
		case optkeyTimestamp:
			t = opt.Value().(time.Time)
		}
	}

	if t.IsZero() {
		t = time.Now()
	}

	msg := makeMessage(tag, v, t, c.subsecond, false)
	defer releaseMessage(msg)

	serialized, err := c.marshaler.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, `failed to serialize payload`)
	}

	var attempt uint64
WRITE:
	attempt++
	if pdebug.Enabled {
		pdebug.Printf("Attempt %d/%d", attempt, c.maxConnAttempts)
	}
	payload := serialized
	if attempt > c.maxConnAttempts {
		return errors.New(`exceeded max connection attempts`)
	}

	conn, err := c.connect(attempt > 1)
	if err != nil {
		goto WRITE
	}
	if pdebug.Enabled {
		pdebug.Printf("Successfully connected to server")
	}

	if pdebug.Enabled {
		pdebug.Printf("Going to write %d bytes", len(payload))
	}

	for len(payload) > 0 {
		n, err := conn.Write(payload)
		if err != nil {
			if err == io.EOF {
				goto WRITE // Try again
			}

			return errors.Wrap(err, `failed to write serialized payload`)
		}
		if pdebug.Enabled {
			pdebug.Printf("Wrote %d bytes", n)
		}
		payload = payload[n:]
	}

	// All done!
	return nil
}

// Ping sends a ping message. A ping for an unbuffered client is completely
// analogous to sending a message with Post
func (c *Unbuffered) Ping(tag string, v interface{}, options ...Option) (err error) {
	if pdebug.Enabled {
		g := pdebug.Marker("fluent.Unbuffered.Ping").BindError(&err)
		defer g.End()
	}

	return c.Post(tag, v, options...)
}
