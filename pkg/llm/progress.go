package llm

import (
	"encoding/json"

	"github.com/obot-platform/nanobot/pkg/llm/anthropic"
	"github.com/obot-platform/nanobot/pkg/llm/responses"
)

func PrintProgress(msg json.RawMessage) bool {
	if anthropic.PrintProgress(msg) {
		return true
	}
	return responses.PrintProgress(msg)
}
