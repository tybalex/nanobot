package llm

import (
	"encoding/json"

	"github.com/obot-platform/nanobot/pkg/llm/anthropic"
	"github.com/obot-platform/nanobot/pkg/llm/responses"
)

func PrintProgress(msg json.RawMessage) {
	if anthropic.PrintProgress(msg) {
		return
	}
	responses.PrintProgress(msg)
}
