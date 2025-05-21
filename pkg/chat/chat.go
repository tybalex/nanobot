package chat

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/obot-platform/nanobot/pkg/mcp"
	"github.com/obot-platform/nanobot/pkg/types"
	"github.com/obot-platform/nanobot/pkg/uuid"
)

func Chat(ctx context.Context, listenAddress, prompt string) error {
	progressToken := uuid.String()

	c, err := mcp.NewClient(ctx, "nanobot", mcp.MCPServer{
		BaseURL: "http://" + listenAddress,
		Headers: nil,
	}, mcp.ClientOption{
		OnLogging: func(ctx context.Context, logMsg mcp.LoggingMessage) error {
			fmt.Printf("%s: %v\n", logMsg.Level, logMsg.Data)
			return nil
		},
		OnNotify: func(ctx context.Context, msg mcp.Message) error {
			//log.Messages(ctx, "cli", true, msg.Params)
			return nil
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create chat client: %w", err)
	}

	if prompt != "" {
		resp, err := c.Call(ctx, types.AgentTool, map[string]any{
			"prompt": prompt,
		}, mcp.CallOption{
			ProgressToken: progressToken,
		})
		if err != nil {
			return fmt.Errorf("failed to call agent tool: %w", err)
		}
		if err := PrintResult(resp); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		return nil
	}

	scanner := bufio.NewScanner(os.Stdin)

	next := func() bool {
		fmt.Println()
		fmt.Print("> ")
		return scanner.Scan()
	}

	line := scanner.Text()
	for next() {
		resp, err := c.Call(ctx, types.AgentTool, map[string]any{
			"prompt": line,
		}, mcp.CallOption{
			ProgressToken: progressToken,
		})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		if resp.IsError {
			fmt.Println("ERROR:")
		}
		if err := PrintResult(resp); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
	}

	return nil
}
