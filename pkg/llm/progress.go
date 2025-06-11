package llm

import (
	"encoding/json"

	"github.com/nanobot-ai/nanobot/pkg/llm/anthropic"
	"github.com/nanobot-ai/nanobot/pkg/llm/responses"
)

func PrintProgress(msg json.RawMessage) bool {
	if anthropic.PrintProgress(msg) {
		return true
	}
	return responses.PrintProgress(msg)
}
