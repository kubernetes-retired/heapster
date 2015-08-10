package extpoints

import (
	"github.com/progrium/go-extpoints/examples/tool/types"
)

type LifecycleParticipant interface {
	CommandStart(commandName string) error
	CommandFinish(commandName string)
}

type CommandProvider interface {
	Commands() []*types.Command
}
