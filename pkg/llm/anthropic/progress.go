package anthropic

import (
	"encoding/json"
	"fmt"
	"os"
)

func PrintProgress(msg json.RawMessage) bool {
	var delta DeltaEvent
	if err := json.Unmarshal(msg, &struct {
		Data *DeltaEvent
	}{
		Data: &delta,
	}); err != nil {
		return false
	}

	switch delta.Type {
	case "message_start":
	case "content_block_start":
		if delta.ContentBlock.Type == "tool_use" {
			_, _ = fmt.Fprintf(os.Stderr, "* Preparing to call (%s) with args: ", delta.ContentBlock.Name)
		} else if delta.ContentBlock.Type == "text" {
			_, _ = fmt.Fprint(os.Stderr, "< ")
		}
	case "content_block_delta":
		switch delta.Delta.Type {
		case "text_delta":
			_, _ = fmt.Fprint(os.Stderr, delta.Delta.Text)
		case "input_json_delta":
			_, _ = fmt.Fprint(os.Stderr, delta.Delta.PartialJSON)
		}
	case "content_block_stop":
		_, _ = fmt.Fprintln(os.Stderr)
	case "message_delta":
	case "message_stop":
		_, _ = fmt.Fprintln(os.Stderr)
	default:
		return false
	}

	return true
}
