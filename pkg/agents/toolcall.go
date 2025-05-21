package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/obot-platform/nanobot/pkg/types"
)

func (a *Agents) toolCalls(ctx context.Context, run *run) error {
	for _, output := range run.Response.Output {
		functionCall := output.ToolCall

		if functionCall == nil {
			continue
		}

		if run.ToolOutputs[functionCall.CallID].Done {
			continue
		}

		targetServer, ok := run.ToolToMCPServer[functionCall.Name]
		if !ok {
			return fmt.Errorf("can not map tool %s to a MCP server", functionCall.Name)
		}

		callOutput, err := a.invoke(ctx, targetServer, functionCall)
		if err != nil {
			return fmt.Errorf("failed to invoke tool %s on MCP server %s: %w", functionCall.Name, targetServer, err)
		}

		if run.ToolOutputs == nil {
			run.ToolOutputs = make(map[string]toolCall)
		}

		run.ToolOutputs[functionCall.CallID] = toolCall{
			Output: callOutput,
			Done:   true,
		}
	}

	if len(run.ToolOutputs) == 0 {
		run.Done = true
	}

	return nil
}

func (a *Agents) invoke(ctx context.Context, target types.ToolMapping, funcCall *types.ToolCall) ([]types.CompletionInput, error) {
	var (
		data map[string]any
	)

	if funcCall.Arguments != "" {
		data = make(map[string]any)
		if err := json.Unmarshal([]byte(funcCall.Arguments), &data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal function call arguments: %w", err)
		}
	}

	response, err := a.registry.Call(ctx, target.MCPServer, target.ToolName, data)
	if err != nil {
		return nil, fmt.Errorf("failed to invoke tool %s on mcp server %s: %w", target.Tool, target.MCPServer, err)
	}

	return []types.CompletionInput{
		{
			ToolCallResult: &types.ToolCallResult{
				CallID: funcCall.CallID,
				Output: *response,
			},
		},
	}, nil
}
