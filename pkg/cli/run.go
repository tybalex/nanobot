package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/obot-platform/nanobot/pkg/chat"
	"github.com/obot-platform/nanobot/pkg/log"
	"github.com/obot-platform/nanobot/pkg/mcp"
	"github.com/obot-platform/nanobot/pkg/runtime"
	"github.com/obot-platform/nanobot/pkg/server"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

type Run struct {
	MCP           bool   `usage:"Run the nanobot as an MCP server" default:"false" short:"m"`
	ListenAddress string `usage:"Address to listen on (ex: localhost:8099)" default:"stdio" short:"a"`
	n             *Nanobot
}

func NewRun(n *Nanobot) *Run {
	return &Run{
		n: n,
	}
}

func (r *Run) Customize(cmd *cobra.Command) {
	cmd.Use = "run [flags] NANOBOT [PROMPT]"
	cmd.Short = "Run the nanobot with the specified config file"
	cmd.Example = `
  # Run the nanobot.yaml in the current directory
  nanobot run .

  # Run a custom config file in another directory with a custom address using HTTP instead of stdio.
  # Note: The current working directory of nanobot will be changed to ./other/path
  nanobot run ./other/path/custom.yaml --address localhost:8099
`
	cmd.Args = cobra.MinimumNArgs(1)
}

func (r *Run) Run(cmd *cobra.Command, args []string) error {
	runtime, err := r.n.GetRuntime(args[0])
	if err != nil {
		return err
	}

	if r.MCP {
		return r.runMCP(cmd.Context(), runtime, nil)
	}

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return fmt.Errorf("failed to pick a local port: %w", err)
	}
	r.ListenAddress = l.Addr().String()

	eg, ctx := errgroup.WithContext(cmd.Context())
	ctx, cancel := context.WithCancel(ctx)
	eg.Go(func() error {
		return r.runMCP(ctx, runtime, l)
	})
	eg.Go(func() error {
		defer cancel()
		return chat.Chat(ctx, r.ListenAddress, strings.Join(args[1:], " "))
	})
	return eg.Wait()
}

func (r *Run) runMCP(ctx context.Context, runtime *runtime.Runtime, l net.Listener) error {
	address := r.ListenAddress
	if strings.HasPrefix("address", "http://") {
		address = strings.TrimPrefix(address, "http://")
	} else if strings.HasPrefix(address, "https://") {
		return fmt.Errorf("https:// is not supported, use http:// instead")
	}

	mcpServer := server.NewServer(runtime)

	if address == "stdio" {
		stdio := mcp.NewStdioServer(mcpServer)
		if err := stdio.Start(ctx, os.Stdin, os.Stdout); err != nil {
			return fmt.Errorf("failed to start stdio server: %w", err)
		}

		stdio.Wait()
		return nil
	}

	httpServer := mcp.HTTPServer{
		MessageHandler: mcpServer,
	}

	s := &http.Server{
		Addr:    address,
		Handler: &httpServer,
	}

	_, _ = fmt.Fprintf(os.Stderr, "Starting server on %s\n", address)
	context.AfterFunc(ctx, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.Shutdown(ctx)
	})

	var err error
	if l == nil {
		err = s.ListenAndServe()
	} else {
		err = s.Serve(l)
	}
	log.Debugf(ctx, "Server stopped: %v", err)
	return nil
}
