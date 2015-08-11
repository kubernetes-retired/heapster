package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/progrium/go-extpoints/examples/tool/types"
)

func init() {
	commandProviders.Register(new(helpComponent), "")
}

type helpComponent struct{}

func (h *helpComponent) Commands() []*types.Command {
	return []*types.Command{cmdHelp}
}

var cmdHelp = &types.Command{
	Run:   runHelp,
	Usage: "help [<topic>]",
	Long:  `Help shows usage for a command or other topic.`,
}

var helpCommands = &types.Command{
	Usage: "commands",
	Short: "list all commands with usage",
	Long:  "(not displayed; see special case in runHelp)",
}

func runHelp(cmd *types.Command, args []string) {
	if len(args) == 0 {
		printUsage()
		return // not os.Exit(2); success
	}
	if len(args) != 1 {
		log.Fatal("too many arguments")
	}
	switch args[0] {
	case helpCommands.Name():
		printAllUsage()
		return
	}

	for _, cmd := range commands {
		if cmd.Name() == args[0] {
			cmd.PrintUsage()
			return
		}
	}

	log.Printf("Unknown help topic: %q. Run 'tool help'.\n", args[0])
	os.Exit(2)
}

func maxStrLen(strs []string) (strlen int) {
	for i := range strs {
		if len(strs[i]) > strlen {
			strlen = len(strs[i])
		}
	}
	return
}

var usageTemplate = template.Must(template.New("usage").Parse(`
Usage: tool <command> [options] [arguments]

Commands:
{{range .Commands}}{{if .Runnable}}{{if .List}}
    {{.Name | printf (print "%-" $.MaxRunListName "s")}}  {{.Short}}{{end}}{{end}}{{end}}

Run 'tool help [command]' for details.

`[1:]))

func printUsage() {
	var runListNames []string
	for i := range commands {
		if commands[i].Runnable() && commands[i].List() {
			runListNames = append(runListNames, commands[i].Name())
		}
	}

	usageTemplate.Execute(os.Stderr, struct {
		Commands       []*types.Command
		MaxRunListName int
	}{
		commands,
		maxStrLen(runListNames),
	})
}

func printAllUsage() {
	w := tabwriter.NewWriter(os.Stdout, 1, 2, 2, ' ', 0)
	defer w.Flush()
	cl := commandList(commands)
	sort.Sort(cl)
	for i := range cl {
		if cl[i].Runnable() {
			listRec(w, "tool "+cl[i].FullUsage(), "# "+cl[i].Short)
		}
	}
}

func listRec(w io.Writer, a ...interface{}) {
	for i, x := range a {
		fmt.Fprint(w, x)
		if i+1 < len(a) {
			w.Write([]byte{'\t'})
		} else {
			w.Write([]byte{'\n'})
		}
	}
}

func usage() {
	printUsage()
	os.Exit(2)
}

type commandList []*types.Command

func (cl commandList) Len() int           { return len(cl) }
func (cl commandList) Swap(i, j int)      { cl[i], cl[j] = cl[j], cl[i] }
func (cl commandList) Less(i, j int) bool { return cl[i].Name() < cl[j].Name() }

type commandMap map[string]commandList
