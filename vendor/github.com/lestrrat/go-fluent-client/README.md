# go-fluent-client

A fluentd client

[![Build Status](https://travis-ci.org/lestrrat/go-fluent-client.png?branch=master)](https://travis-ci.org/lestrrat/go-fluent-client)

[![GoDoc](https://godoc.org/github.com/lestrrat/go-fluent-client?status.svg)](https://godoc.org/github.com/lestrrat/go-fluent-client)

# SYNOPSIS

```go
package fluent_test

import (
  "context"
  "log"
  "time"

  fluent "github.com/lestrrat/go-fluent-client"
)

func Example() {
  // Connects to fluentd at 127.0.0.1:24224. If you want to connect to
  // a different host, use the following:
  //
  //   client, err := fluent.New(fluent.WithAddress("fluent.example.com"))
  //
  client, err := fluent.New()
  if err != nil {
    // fluent.New may return an error if invalid values were
    // passed to the constructor
    log.Printf("failed to create client: %s", err)
    return
  }

  // do not forget to shutdown this client at the end. otherwise
  // we would not know if we were able to flush the pending
  // buffer or not.
  defer func() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := client.Shutdown(ctx); err != nil {
      // Failed to shutdown properly. force-close it
      client.Close()
    }
  }()

  var payload = map[string]string{
    "foo": "bar",
  }
  if err := client.Post("tag", payload); err != nil {
    log.Printf("failed to post: %s", err)
    return
  }
}

func ExamplePing() {
  client, err := fluent.New()
  if err != nil {
    log.Printf("failed to create client: %s", err)
    return
  }

  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()

  // Goroutine to wait for errors
  errorCh := make(chan error, 1)
  go func() {
    // This is just an example to stop pinging on errors
    for {
      select {
      case <-ctx.Done():
        return
      case e := <-errorCh:
        log.Printf("got an error during ping: %s", e.Error())
        cancel()
        return
      }
    }
  }()

  go fluent.Ping(ctx, client, "ping", "hostname", fluent.WithPingResultChan(errorCh))

  // Do what you need with your main program...
}

```

# DESCRIPTION

This is a client to the fluentd log collection daemon.

# FEATURES

## Performance

Please see the BENCHMARK section.

## A well defined `Shutdown()` method

Because we expect to connect to remote daemons over the wire, the various fluentd clients all perform local buffering of data to be sent, then sends them when it can. At the end of your program, you should wait for your logs to be sent to the server, otherwise you might have pending writes that haven't gone through yet.

Calling either `Close()` or `Shutdown()` triggers the flushing of pending logs, but the former does not wait for this operation to be completed, while the latter does. With `Shutdown` you can either wait indefinitely, or timeout the operation after the desired period of time using `context.Context`

## A flexible `Post()` method

The `Post()` method provided by this module can either simply enqueue a new payload to be appended to the buffer mentioned in the previous section, and let it process asynchronously, or it can wait for confirmation that the payload has been properly enqueued. Other libraries usually only do one or the other, but we can handle either.

```go
// "fire-and-forget"
client.Post(tagName, payload)

// make sure that we receive confirmation the payload has been appended
if err := client.Post(tagName, payload, fluent.WithSyncAppend(true)); err != nil {
  ...
}
```

## Buffered/Unbuffered clients

By default, we create a "buffered" client. This means that we enqueue the data to be sent to the fluentd process locally until we can actually connect and send them. However, since this decouples the user from the actual timing when the message is sent to the server, it may not be a suitable solution in cases where immediate action must be taken in case a message could not be sent.

If you must detect if a message has been successfully sent to the fluentd server *immediately*, use the Unbuffered client by passing the `fluent.WithBuffered` option to the constructor:

```go
client, err := fluent.New(
  fluent.WithAddress("fluent.example.com"),
  fluent.WithBuffered(false), // Use the unbuffered client
)
```

The behavior will change as described above, but the interface is still the same.

# OPTIONS (fluent.New)

| Name | Short Description | Default Value | Bufferd | Unbuffered |
|:-----|:------------------|:--------------|:--------|:-----------|
| fluent.WithBuffered(bool)             | Use buffered/unbuffered client      | true              | - | - | 
| fluent.WithNetwork(string)            | Network type of address             | "tcp"             | Y | Y |
| fluent.WithAddress(string)            | Address to connect to               | "127.0.0.1:24224" | Y | Y |
| fluent.WithJSONMarshaler()            | Use JSON as serialization format    | -                 | Y | Y |
| fluent.WithMsgpackMarshaler()         | Use msgpack as serialization format | used by default   | Y | Y |
| fluent.WithTagPrefix(string)          | Tag prefix to prepend               | -                 | Y | Y |
| fluent.WithDialTimeout(time.Duration) | Timeout value when connecting       | 3 * time.Second   | Y | Y |
| fluent.WithConnectOnStart(bool)       | Attempt to connect immediately      | false             | Y | Y |
| fluent.WithSubsecond(bool)            | Use EventTime                       | false             | Y | Y |
| fluent.WithBufferLimit(int)           | Max buffer size to store            | 8 * 1024 * 1024   | Y | N |
| fluent.WithWriteThreshold(int)        | Min buffer size before writes start | 8 * 1024          | Y | N |
| fluent.WithMaxConnAttempts(int)       | Max attempts to make during close (buffered), or max attempts to make when connecting to the server (unbuffered)  | 64 | Y | Y |
| fluent.WithWriteQueueSize(int)        | Channel size for background reader  | 64                | Y | N |

# OPTIONS ((fluent.Client).Post)

| Name | Short Description | Default Value | Bufferd | Unbuffered |
|:-----|:------------------|:--------------|:--------|:-----------|
| fluent.WithTimestamp(time.Time)     | Timestamp to use for message        | current time      | Y | Y |
| fluent.WithContext(context.Context) | Context to use                      | none              | Y | N |
| fluent.WithSyncAppend(bool)         | Return failure if appending fails   | false             | Y | N |

# OPTIONS (fluent.Ping)

| Name | Short Description | Default Value |
|:-----|:------------------|:--------------|
| fluent.WithPingInterval(time.Duration) | Interval between pings    | 5 * time.Second |
| fluent.WithPingResultChan(chan error)  | Where to send ping errors | none |

# BENCHMARKS

instructions: make sure you have the required fluentd clients, start fluentd at 127.0.0.1:24224, and run

```
go test -run=none -bench=. -benchmem -tags bench
```

```
BenchmarkK0kubun-4              	  500000	      3220 ns/op	     968 B/op	      12 allocs/op
BenchmarkLestrrat-4             	  500000	      4122 ns/op	     530 B/op	       7 allocs/op
BenchmarkLestrratJSON-4         	  300000	      6639 ns/op	     741 B/op	      11 allocs/op
BenchmarkLestrratUnbuffered-4   	  100000	     11719 ns/op	     512 B/op	       7 allocs/op
BenchmarkOfficial-4             	  100000	     10443 ns/op	     896 B/op	       9 allocs/op
BenchmarkOfficialJSON-4         	  100000	     17796 ns/op	    1760 B/op	      25 allocs/op
PASS
ok  	github.com/lestrrat/go-fluent-client	10.287s
```

## Versions

| Name | Version |
|---------|---------|
| fluentd (td-agent) | 0.12.19 |
| github.com/lestrrat/go-fluent-client | f3fb05a2b7eb40bb426dd8e1ba43e9ff47b412ec |
| github.com/k0kubun/fluent-logger-go | e1cfc57bb12c99d7207d43b942527c9450d14382 |
| github.com/fluent/fluent-logger-golang | 8bbc2356beaf021b04c9bd5cdc76ea5a7ccb40ec |

## Analysis 

### github.com/lestrrat/go-fluent-client

#### Pros

* Lowest allocations per op
* Proper `Shutdown` method to flush buffers at the end.
* Tried very hard to avoid any race conditions.
* Has buffered/unbuffered clients

While `github.com/k0kubun/fluent-logger-go` is fastest, it does not check errors and does not handle some
synchronization edge cases. `github.com/lestrrat/go-fluent-client` goes into great pains to check these
things, and still manages to come almost as fast as `github.com/k0kubun/fluent-logger-go`, whihch is already
several times faster than the official library.

The buffered client is the default, but you may use the unbuffered client, which does not keep
the payload in an internal buffer before sending to the server. This is slightly more inefficient,
but has the advantage of allowing you to handle errors more gracefully.

#### Cons

* I'm biased (duh).

### github.com/k0kubun/fluent-logger-go

#### Pros

This library is fast. I believe this is due to the fact that it does very little error
handling and synchronization. If you use the msgpack serialization format and that's it,
you probably will be fine using this library.

#### Cons

Do note that as of the version I tested above, the `Logger.Log` method has a glaring race condition
that will probably corrupt your messages sooner than you can blink: DO NOT USE THE `Logger.Log` method.

Also, there is no way for the caller to check for serialization errors when using `Logger.Post`. You can get the
error status using `Logger.Log`, but as previously stated, you do not want to use it. 

Finally, there is no way to flush pending buffers: If you append a lot of buffers in succession, and
abruptly quit your program, you're done for. You lose all your data.

Oh, and it supports Msgpack only, but this is a very minor issue: a casual user really shouldn't have to care which
serialization format you're sending your format with.

### github.com/fluent/fluent-logger-golang

#### Pros

This official binding from the maitainers of fluentd is by far the most battle-tested library. It may be
a bit slow, but it's sturdy, period.

#### Cons

The benchmark scores are pretty low. This could just be my benchmark, so please take with a grain of salt.

Looking at the code, it looks non-gopher-ish. Use of `panic` is one such item. In Go you should avoid
casual panics, which causes long-running daemons to write code like this https://github.com/moby/moby/blob/1325f667eeb42b717c2f9d369f2ee6d701a280e3/daemon/logger/fluentd/fluentd.go#L46-L49

