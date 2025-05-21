package responses

import (
	"encoding/json"
	"fmt"
	"os"
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
			_, _ = fmt.Fprintf(os.Stderr, "* Preparing to call (%s) with args: ", delta.Item.Name)
		}
	case "response.function_call_arguments.delta":
		_, _ = fmt.Fprint(os.Stderr, delta.Delta)
	case "response.output_item.done":
		_, _ = fmt.Fprintln(os.Stderr)
	case "response.content_part.added":
		_, _ = fmt.Fprint(os.Stderr, "< ")
	case "response.output_text.delta":
		_, _ = fmt.Fprint(os.Stderr, delta.Delta)
	case "response.content_part.done":
		_, _ = fmt.Fprintln(os.Stderr)
	case "message_delta":
	case "message_stop":
		_, _ = fmt.Fprintln(os.Stderr)
	default:
		return false
	}

	return true
}
