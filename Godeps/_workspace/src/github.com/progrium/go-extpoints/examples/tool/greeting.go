package main

import (
	"fmt"
	"strings"

	"github.com/progrium/go-extpoints/examples/tool/types"
)

func init() {
	commandProviders.Register(new(greetingComponent), "")
}

type greetingComponent struct{}

func (h *greetingComponent) Commands() []*types.Command {
	return []*types.Command{cmdHello, cmdGoodbye}
}

var cmdHello = &types.Command{
	Run:   runHello,
	Usage: "hello [<name>]",
	Short: "greets you with a hello, optionally by name",
	Long: `Provides you with a personalized greeting.

Examples:

	$ tool hello
	Hello!

	$ tool hello Jeff
	Hello, Jeff!
`,
}

func runHello(cmd *types.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("Hello!")
	} else {
		fmt.Printf("Hello, %s!\n", strings.Join(args, " "))
	}
}

var cmdGoodbye = &types.Command{
	Run:   runGoodbye,
	Usage: "goodbye",
	Short: "wishes you goodbye, optionally by name",
	Long: `Provides you with a personalized goodbye.

Examples:

	$ tool goodbye
	Goodbye!

	$ tool goodbye Jeff
	Goodbye, Jeff!
`,
}

func runGoodbye(cmd *types.Command, args []string) {
	if len(args) == 0 {
		fmt.Println("Goodbye!")
	} else {
		fmt.Printf("Goodbye, %s!\n", strings.Join(args, " "))
	}
}
