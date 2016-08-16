# go-extpoints

This Go generator, named short for "extension points", provides a generic [inversion of control](http://en.wikipedia.org/wiki/Inversion_of_control) model for making extensible Go packages, libraries, and applications.

It generates package extension point singletons from extension types you define. Extension points are then used to both register extensions and use the registered extensions with a common [meta-API](#extension-point-api).

[Logspout](https://github.com/gliderlabs/logspout) is a real application built using go-extpoints. [Read about it here.](http://gliderlabs.com/blog/2015/03/31/new-logspout-extensible-docker-logging/)

## Getting the tool

	$ go install github.com/progrium/go-extpoints

## Concepts

#### Extension Types

These define your hooks. They can be Go interfaces or simple function signature types. Here are some generic examples:

```go
type ConfigStore interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Del(key string) error
}

type AuthProvider interface {
	Authenticate(user, pass string) bool
}

type EventListener interface {
	Notify(event Event)
}

type HttpEndpoint func() http.Handler

type RequestModifier func(request *http.Request)

```

#### Extension Points

With types defined, go-extpoints generates package singletons for each type. When your program starts, extensions are registered with extension points. Then you can then use registered extensions in a number of ways:

```go
// Lookup a single registered extension for drivers
config := ConfigStores.Lookup(configStore)
if config == nil {
	log.Fatalf("config store '%s' not registered", configStore)
}
config.Set("foo", "bar")

// Iterate until you get what you need
func authenticate(user, pass string) bool {
	for _, provider := range AuthProviders.All() {
		if provide.Authenticate(user, pass) {
			return true
		}
	}
	return false
}

// Fire and forget events to all extensions
for _, listener := range EventListeners.All() {
	listener.Notify(event)
}

// Use name and return value of all extensions for registration
for name, handler := range HttpEndpoints.All() {
	http.Handle("/"+name, handler())
}

// Pass by reference to all extensions for middleware
for _, modifier := range RequestModifiers.All() {
	modifier(req)
}

```

## Extension Point API

All extension types passed to go-extpoints will be turned into extension point singletons, using the pluralized name of the extension type. These extension point objects implement this simple meta-API:

```go
type <ExtensionPoint> interface {
	// if name is "", the specific extension type is used.
	// returns false if doesn't implement type or already registered.
	Register(extension <ExtensionType>, name string) bool

	// returns false if not registered to start with
	Unregister(name string) bool

	// returns nil if not registered
	Lookup(name string) <ExtensionType>

	// for sorted subsets. each name is looked up in order, nil or not
	Select(names []string) []<ExtensionType>

	// all registered, keyed by name
	All() map[string]<ExtensionType>

	// convenient list of names
	Names() []string

}
```

It also generates top-level registration functions that will run extensions through all known extension points, registering or unregistering with any that are based on an interface the extension implements. They return the names of the interfaces they were registered/unregistered with.

```go

func RegisterExtension(extension interface{}, name string) []string

func UnregisterExtension(name string) []string

```

## Example Application

Here is a full Go application that lets extensions hook into `main()` as subcommands simply by implementing an interface we'll make called `Subcommand`. This interface will have just one method `Run()`, but you can make extension points based on any interface.

Assuming our package lives under `$GOPATH/src/github.com/quick/example`, here is our `main.go`:

```go
//go:generate go-extpoints
package main

import (
	"fmt"
	"os"

	"github.com/quick/example/extpoints"
)

var subcommands = extpoints.Subcommands

func usage() {
	fmt.Println("Available commands:\n")
	for name, _ := range subcommands.All() {
		fmt.Println(" - ", name)
	}
	os.Exit(2)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cmd := subcommands.Lookup(os.Args[1])
	if cmd == nil {
		usage()
	}
	cmd.Run(os.Args[2:])
}
```
Two things to note. First, the `go:generate` directive at the top. This tells `go generate` it needs to run `go-extpoints`, which will happen in a moment.

Another thing to note, the extension point is accessed by a variable named by the plural of our interface `Subcommand` and in this case lives under a separate `extpoints` subpackage.

We need to create that subpackage with a Go file in it to define our interface used for this extension point. This is our `extpoints/interfaces.go`:

```go
package extpoints

type Subcommand interface {
	Run(args []string)
}
```

We use `go generate` now to produce the extension point code in our `extpoints` subpackage. These extension points are based on any Go interfaces you've defined in there. In our case, just `Subcommand`.

	$ go generate
	 ...
	$ go install

Okay, but it doesn't *do* anything! Let's make a builtin command extension that implements `Subcommand`. Add a `hello.go` file:

```go
package main

import (
	"fmt"
	"github.com/quick/example/extpoints"
)

func init() {
	extpoints.Register(new(HelloComponent), "hello")
}

type HelloComponent struct {}

func (p *HelloComponent) Run(args []string) {
	fmt.Println("Hello world!")
}
```

Now when we build and run the app, it shows `hello` as a subcommand. We've registered the component with the name `hello`, which we happen to use to identify the name of the subcommand. Component names are optional, but can be handy for situations like this one.

Certainly, the value of extension points becomes clearer with larger applications and more interesting interfaces. But just consider now that the component defined in `hello.go` *could* exist in another package in another repo. You'd just have to import it and rebuild to let it hook into our application.

There are two more in-deptch example applications in this repo to take a look at:

 * [tool](https://github.com/progrium/go-extpoints/tree/master/examples/tool) ([extpoints](http://godoc.org/github.com/progrium/go-extpoints/examples/tool/extpoints)), a more realistic CLI tool with subcommands and lifecycle hooks
 * [daemon](https://github.com/progrium/go-extpoints/tree/master/examples/daemon), ... doesn't exist yet



## Making it easy to install extensions

Assuming you tell third-party developers to call your package or extension point `Register` in their `init()`, you can link them with a side-effect import (using a blank import name).

You can make this easy for users to enable/disable via comments, or add their own without worrying about messing with your code by having a separate `extensions.go` or `plugins.go` file with just these imports:

```go
package yourpackage

import (
	_ "github.com/you/some-extension"
	_ "github.com/third-party/another-extension"
)

```

Users can now just edit this file and `go build` or `go install`.

## Usage Patterns

Here are different example ways to use extension points to interact with extensions:

#### Simple Iteration
```go
for _, listener := range extpoints.EventListeners.All() {
	listener.Notify(&MyEvent{})
}
```

#### Lookup Only One
```go
driverName := config.Get("storage-driver")
driver := extpoints.StorageDrivers.Lookup(driverName)
if driver == nil {
	log.Fatalf("storage driver '%s' not installed", driverName)
}
driver.StoreObject(object)
```

#### Passing by Reference
```go
for _, filter := range extpoints.RequestFilters.All() {
	filter.FilterRequest(req)
}
```

#### Match and Use
```go
for _, handler := range extpoints.RequestHandlers.All() {
	if handler.MatchRequest(req) {
		handler.HandleRequest(req)
		break
	}
}
```

## Why the `extpoints` subpackage?

Since we encourage the convention of a subpackage called `extpoints`, it makes it very easy to identify a package as having extension points from looking at the project tree. You then know where to look to find the interfaces that are exposed as extension points.

Third-party packages have a well known package to import for registering. Whether you have extension points for a library package or a command with just a `main` package, there's always a definite `extpoints` package there to import.

It also makes it clearer in your code when you're using extension points. You have to explicitly import the package, then call `extpoints.<ExtensionPoint>` when using them. This helps identify where extension points actually hook into your program.

## Groundwork for Dynamic Extensions

Although this only seems to allow for compile-time extensibility, this itself is quite a win. It means power users can build and compile in their own extensions that live outside your repository.

However, it also lays the groundwork for other dynamic extensions. I've used this model to wrap extension points for components in embedded scripting languages, as hook scripts, as remote plugin daemons via RPC, or all of the above implemented as components themselves!

No matter how you're thinking about dynamic extensions later on, using `go-extpoints` gives you a lot of options. Once Go supports dynamic libraries? This will work perfectly with that, too.

## Inspiration

This project and component model is a lightweight, Go idiomatic port of the [component architecture](http://trac.edgewall.org/wiki/TracDev/ComponentArchitecture) used in Trac, which is written in Python. It's taken about a year to get this right in Go.

![Trac Component Architecture](http://trac.edgewall.org/raw-attachment/wiki/TracDev/ComponentArchitecture/xtnpt.png)

## License

BSD
