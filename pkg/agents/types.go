package agents

import (
	"github.com/obot-platform/nanobot/pkg/types"
)

type run struct {
	Request          types.CompletionRequest   `json:"request,omitempty"`
	Done             bool                      `json:"done,omitempty"`
	PopulatedRequest *types.CompletionRequest  `json:"populatedRequest,omitempty"`
	ToolToMCPServer  types.ToolMappings        `json:"toolToMCPServer,omitempty"`
	Response         *types.CompletionResponse `json:"response,omitempty"`
	ToolOutputs      map[string]toolCall       `json:"toolOutputs,omitempty"`
}

type toolCall struct {
	Output []types.CompletionInput `json:"output,omitempty"`
	Done   bool                    `json:"done,omitempty"`
}
