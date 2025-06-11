package responses

import (
	"encoding/json"
	"fmt"

	"github.com/nanobot-ai/nanobot/pkg/printer"
)

func PrintProgress(msg json.RawMessage) bool {
	var delta Progress
	if err := json.Unmarshal(msg, &struct {
		Data *Progress
	}{
		Data: &delta,
	}); err != nil {
		return false
	}

	switch delta.Type {
	case "response.created":
	case "response.output_item.added":
		if delta.Item.Type == "function_call" {
			printer.Prefix("<-(llm)", fmt.Sprintf("Preparing to call (%s) with args: ", delta.Item.Name))
		}
	case "response.function_call_arguments.delta":
		printer.Prefix("<-(llm)", delta.Delta)
	case "response.output_item.done":
		printer.Prefix("<-(llm)", "\n")
	case "response.content_part.added":
	case "response.output_text.delta":
		printer.Prefix("<-(llm)", delta.Delta)
	case "response.content_part.done":
	case "message_delta":
	case "message_stop":
	default:
		return false
	}

	return true
}
