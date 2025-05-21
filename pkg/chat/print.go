package chat

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/obot-platform/nanobot/pkg/mcp"
)

func writeData(result mcp.Content) error {
	data, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		return err
	}
	i := 1
	for {
		filename := fmt.Sprintf("output%d.data", i)
		_, err := os.Stat(filename)
		if !errors.Is(err, fs.ErrNotExist) {
			i++
			continue
		}

		if err := os.WriteFile(filename, data, 0644); err != nil {
			return err
		}
		name := result.Type
		if result.MIMEType != "" {
			name += "(" + result.MIMEType + ")"
		}
		fmt.Printf("%s written to %s\n", name, filename)
		return nil
	}
}

func PrintResult(result *mcp.CallToolResult) error {
	for _, out := range result.Content {
		if out.Text != "" {
			fmt.Println(out.Text)
		} else if out.Data != "" {
			if err := writeData(out); err != nil {
				return err
			}
		}
	}

	return nil
}
