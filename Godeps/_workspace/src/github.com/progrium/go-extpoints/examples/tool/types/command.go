package types

import (
	"flag"
	"fmt"
	"strings"
)

type Command struct {
	// args does not include the command name
	Run  func(cmd *Command, args []string)
	Flag flag.FlagSet

	Usage string // first word is the command name
	Short string // `tool help` output
	Long  string // `tool help cmd` output
}

func (c *Command) PrintUsage() {
	if c.Runnable() {
		fmt.Printf("Usage: tool %s\n\n", c.FullUsage())
	}
	fmt.Println(c.Long)
}

func (c *Command) FullUsage() string {
	return c.Usage
}

func (c *Command) Name() string {
	name := c.Usage
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

func (c *Command) Runnable() bool {
	return c.Run != nil
}

func (c *Command) List() bool {
	return c.Short != ""
}
