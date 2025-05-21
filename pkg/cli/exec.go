package cli

import (
	"os"

	"github.com/obot-platform/nanobot/pkg/chat"
	"github.com/spf13/cobra"
)

type Exec struct {
	File   string `usage:"File to read input from" default:"" short:"f"`
	Output string `usage:"Output format (json, pretty)" default:"pretty" short:"o"`
	n      *Nanobot
}

func NewExec(n *Nanobot) *Exec {
	return &Exec{
		n: n,
	}
}

func (e *Exec) Customize(cmd *cobra.Command) {
	cmd.Hidden = true
	cmd.Use = "exec [flags] NANOBOT_CONFIG TOOL_NAME|AGENT_NAME [AGENT PROMPT]"
	cmd.Short = "Execute a single tool or agent in the nanobot"
	cmd.Example = `
  # Run a tool, passing in a JSON object as input. Tools expect a JSON object as input.
  nanobot exec . server1/tool1 '{"arg1": "value1", "arg2": "value2"}'

  # Run a tool, passing in the same input as above, but using a friendly format.
  nanobot exec . server1/tool1 --arg1=value1 --arg2 value2

  # Run an agent, passing in a string as input. If the input is JSON it will be based as is.
  # Note: The current working directory of nanobot will be changed to ./other/path
  nanobot exec . agent1 "What is the weather like today?"
`
	cmd.Args = cobra.MinimumNArgs(2)
}

func (e *Exec) Run(cmd *cobra.Command, args []string) error {
	runtime, err := e.n.GetRuntime(args[0])
	if err != nil {
		return err
	}

	ctx := runtime.WithTempSession(cmd.Context())

	result, err := runtime.CallFromCLI(ctx, args[1], args[2:]...)
	if err != nil {
		return err
	}

	if display(result, e.Output) {
		return nil
	}

	return chat.PrintResult(os.Stdout, result)
}
