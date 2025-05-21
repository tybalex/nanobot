package types

import (
	"encoding/json"
	"strings"

	"github.com/obot-platform/nanobot/pkg/mcp"
)

type Config struct {
	Publish    Publish                  `json:"publish,omitempty"`
	Runtime    Runtime                  `json:"runtime,omitempty"`
	Agents     map[string]Agent         `json:"agents,omitempty"`
	MCPServers map[string]mcp.MCPServer `json:"mcpServers,omitempty"`
}

type Runtime struct {
	BaseImage   string   `json:"baseImage,omitempty"`
	Root        string   `json:"root,omitempty"`
	Distro      string   `json:"distro,omitempty"`
	Packages    []string `json:"packages,omitempty"`
	SetupScript string   `json:"setupScript,omitempty"`
}

type Publish struct {
	Name              string     `json:"name,omitempty"`
	Description       string     `json:"description,omitempty"`
	Version           string     `json:"version,omitempty"`
	Instructions      string     `json:"instructions,omitempty"`
	Tools             StringList `json:"tools,omitzero"`
	Prompts           StringList `json:"prompts,omitzero"`
	Resources         StringList `json:"resources,omitzero"`
	ResourceTemplates StringList `json:"resourceTemplates,omitzero"`
	Entrypoint        string     `json:"entrypoint,omitempty"`
}

type ToolRef struct {
	Server string
	Tool   string
	As     string
}

func (t ToolRef) PublishedName(name string) string {
	if t.As != "" {
		return t.As
	}
	if t.Tool != "" {
		return t.Tool
	}
	return name
}

func ParseToolRef(ref string) ToolRef {
	name, as, _ := strings.Cut(ref, ":")
	server, tool, _ := strings.Cut(name, "/")
	return ToolRef{
		Server: server,
		Tool:   tool,
		As:     as,
	}
}

type ResourceMappings map[string]TargetMapping

type ResourceTemplateMappings map[string]TargetMapping

type PromptMappings map[string]TargetMapping

type TargetMapping struct {
	MCPServer  string `json:"mcpServer,omitempty"`
	TargetName string `json:"targetName,omitempty"`
	Target     any    `json:"target,omitempty"`
}

type ToolMappings map[string]TargetMapping

type StringList []string

func (s *StringList) UnmarshalJSON(data []byte) error {
	if data[0] == '[' && data[len(data)-1] == ']' {
		var raw []string
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		*s = raw
	} else {
		var raw string
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		*s = StringList{raw}
	}
	return nil
}

type Agent struct {
	Description    string                    `json:"description,omitempty"`
	Instructions   DynamicInstructions       `json:"instructions,omitempty"`
	Model          string                    `json:"model,omitempty"`
	Tools          []string                  `json:"tools,omitempty"`
	History        *bool                     `json:"history,omitempty"`
	ToolExtensions map[string]map[string]any `json:"toolExtensions,omitempty"`
	ToolChoice     string                    `json:"toolChoice,omitempty"`
	Temperature    *json.Number              `json:"temperature,omitempty"`
	TopP           *json.Number              `json:"topP,omitempty"`
	Output         *OutputSchema             `json:"output,omitempty"`
	Truncation     string                    `json:"truncation,omitempty"`
	MaxTokens      int                       `json:"maxTokens,omitempty"`

	// Selection criteria fields

	Aliases      []string `json:"aliases,omitempty"`
	Cost         float64  `json:"cost,omitempty"`
	Speed        float64  `json:"speed,omitempty"`
	Intelligence float64  `json:"intelligence,omitempty"`
}

type DynamicInstructions struct {
	Instructions string            `json:"-"`
	MCPServer    string            `json:"mcpServer"`
	Prompt       string            `json:"prompt"`
	Args         map[string]string `json:"args"`
}

func (a DynamicInstructions) IsPrompt() bool {
	return a.MCPServer != "" && a.Prompt != ""
}

func (a DynamicInstructions) IsSet() bool {
	return a.IsPrompt() || a.Instructions != ""
}

func (a *DynamicInstructions) UnmarshalJSON(data []byte) error {
	if data[0] == '"' && data[len(data)-1] == '"' {
		var raw string
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		a.Instructions = raw
		return nil
	}
	type Alias DynamicInstructions
	return json.Unmarshal(data, (*Alias)(a))
}

func (a DynamicInstructions) MarshalJSON() ([]byte, error) {
	if a.Instructions != "" {
		return json.Marshal(a.Instructions)
	}
	type Alias DynamicInstructions
	return json.Marshal(Alias(a))
}

type OutputSchema struct {
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Schema      json.RawMessage `json:"schema,omitzero"`
	Strict      bool            `json:"strict,omitempty"`
}
