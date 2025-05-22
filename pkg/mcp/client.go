package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/obot-platform/nanobot/pkg/log"
)

type Client struct {
	Session *Session
}

type ClientOption struct {
	OnSampling    func(ctx context.Context, sampling CreateMessageRequest) (CreateMessageResult, error)
	OnRoots       func(ctx context.Context, msg Message) error
	OnLogging     func(ctx context.Context, logMsg LoggingMessage) error
	OnMessage     func(ctx context.Context, msg Message) error
	OnNotify      func(ctx context.Context, msg Message) error
	Env           map[string]string
	ParentSession *Session
	SessionID     string
}

type MCPServer struct {
	Env     map[string]string `json:"env,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	BaseURL string            `json:"baseURL,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

func toHandler(opts ClientOption) MessageHandler {
	return MessageHandlerFunc(func(ctx context.Context, msg Message) {
		if msg.Method == "sampling/createMessage" && opts.OnSampling != nil {
			var param CreateMessageRequest
			if err := json.Unmarshal(msg.Params, &param); err != nil {
				msg.SendUnknownError(ctx, fmt.Errorf("failed to unmarshal sampling/createMessage: %w", err))
				return
			}
			go func() {
				resp, err := opts.OnSampling(ctx, param)
				if err != nil {
					msg.SendUnknownError(ctx, fmt.Errorf("failed to handle sampling/createMessage: %w", err))
					return
				}
				err = msg.Reply(ctx, resp)
				if err != nil {
					log.Errorf(ctx, "failed to reply to sampling/createMessage: %v", err)
				}
			}()
		} else if msg.Method == "roots/list" && opts.OnRoots != nil {
			go func() {
				if err := opts.OnRoots(ctx, msg); err != nil {
					msg.SendUnknownError(ctx, fmt.Errorf("failed to handle roots/list: %w", err))
					return
				}
			}()
		} else if msg.Method == "notifications/message" && opts.OnLogging != nil {
			var param LoggingMessage
			if err := json.Unmarshal(msg.Params, &param); err != nil {
				msg.SendUnknownError(ctx, fmt.Errorf("failed to unmarshal notifications/message: %w", err))
				return
			}
			if err := opts.OnLogging(ctx, param); err != nil {
				msg.SendUnknownError(ctx, fmt.Errorf("failed to handle notifications/message: %w", err))
				return
			}
		} else if strings.HasPrefix(msg.Method, "notifications/") && opts.OnNotify != nil {
			if err := opts.OnNotify(ctx, msg); err != nil {
				log.Errorf(ctx, "failed to handle notification: %v", err)
			}
		} else if opts.OnMessage != nil {
			if err := opts.OnMessage(ctx, msg); err != nil {
				log.Errorf(ctx, "failed to handle message: %v", err)
			}
		}
	})
}

func complete(opts ...ClientOption) ClientOption {
	var all ClientOption
	for _, opt := range opts {
		if opt.OnRoots != nil {
			if all.OnRoots != nil {
				panic("multiple OnRoots handlers provided")
			}
			all.OnRoots = opt.OnRoots
		}
		if opt.OnSampling != nil {
			if all.OnSampling != nil {
				panic("multiple OnSampling handlers provided")
			}
			all.OnSampling = opt.OnSampling
		}
		if opt.OnNotify != nil {
			if all.OnNotify != nil {
				panic("multiple OnNotify handlers provided")
			}
			all.OnNotify = opt.OnNotify
		}
		if opt.OnMessage != nil {
			if all.OnMessage != nil {
				panic("multiple OnMessage handlers provided")
			}
			all.OnMessage = opt.OnMessage
		}
		if opt.OnLogging != nil {
			if all.OnLogging != nil {
				panic("multiple OnLogging handlers provided")
			}
			all.OnLogging = opt.OnLogging
		}
		if len(opt.Env) > 0 {
			if all.Env == nil {
				all.Env = make(map[string]string)
			}
			for k, v := range opt.Env {
				all.Env[k] = v
			}
		}
		if opt.SessionID != "" {
			if all.SessionID != "" {
				panic("multiple SessionID provided")
			}
			all.SessionID = opt.SessionID
		}
		if opt.ParentSession != nil {
			if all.ParentSession != nil {
				panic("multiple ParentSession provided")
			}
			all.ParentSession = opt.ParentSession
		}
	}
	return all
}

func ReplaceString(envs map[string]string, str string) string {
	return os.Expand(str, func(key string) string {
		if val, ok := envs[key]; ok {
			return val
		}
		return "${" + key + "}"
	})
}

func ReplaceMap(envs map[string]string, m map[string]string) map[string]string {
	newMap := make(map[string]string, len(m))
	for k, v := range m {
		newMap[ReplaceString(envs, k)] = ReplaceString(envs, v)
	}
	return newMap
}

func replaceEnv(envs map[string]string, command string, args []string, env map[string]string) (string, []string, []string) {
	newEnvMap := maps.Clone(envs)
	if newEnvMap == nil {
		newEnvMap = make(map[string]string, len(env))
	}
	maps.Copy(newEnvMap, ReplaceMap(envs, env))

	newEnv := make([]string, 0, len(env))
	for _, k := range slices.Sorted(maps.Keys(newEnvMap)) {
		newEnv = append(newEnv, fmt.Sprintf("%s=%s", k, newEnvMap[k]))
	}

	newArgs := make([]string, len(args))
	for i, arg := range args {
		newArgs[i] = ReplaceString(envs, arg)
	}
	return ReplaceString(envs, command), newArgs, newEnv
}

func NewSession(ctx context.Context, serverName string, config MCPServer, opts ...ClientOption) (*Session, error) {
	var (
		wire wire
		err  error
		opt  = complete(opts...)
	)

	cmd, args, env := replaceEnv(opt.Env, config.Command, config.Args, config.Env)
	headers := ReplaceMap(opt.Env, config.Headers)

	if config.Command == "" && config.BaseURL == "" {
		return nil, fmt.Errorf("no command or base URL provided")
	} else if config.BaseURL != "" {
		if config.Command != "" {
			if err := runCommand(ctx, cmd, args, env); err != nil {
				return nil, err
			}
		}
		wire = NewHTTPClient(serverName, config.BaseURL, headers)
	} else {
		wire, err = newStdioClient(ctx, serverName, cmd, args, env)
		if err != nil {
			return nil, err
		}
	}

	session, err := newSession(ctx, wire, toHandler(opt), opt.SessionID, nil)
	if err != nil {
		return nil, err
	}
	session.Parent = opt.ParentSession
	return session, nil
}

func runCommand(ctx context.Context, cmd string, args []string, env []string) error {
	osCmd := exec.CommandContext(ctx, cmd, args...)
	osCmd.Env = env
	osCmd.Stdout = os.Stdout
	osCmd.Stderr = os.Stderr
	if err := osCmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}
	return nil
}

func NewClient(ctx context.Context, serverName string, config MCPServer, opts ...ClientOption) (*Client, error) {
	var (
		opt = complete(opts...)
	)

	session, err := NewSession(ctx, serverName, config, opts...)
	if err != nil {
		return nil, err
	}

	c := &Client{
		Session: session,
	}

	var (
		sampling *struct{}
		roots    *RootsCapability
	)
	if opt.OnSampling != nil {
		sampling = &struct{}{}
	}
	if opt.OnRoots != nil {
		roots = &RootsCapability{}
	}
	_, err = c.Initialize(ctx, InitializeRequest{
		ProtocolVersion: "2025-03-26",
		Capabilities: ClientCapabilities{
			Sampling: sampling,
			Roots:    roots,
		},
		ClientInfo: ClientInfo{},
	})
	return c, err
}

func (c *Client) Initialize(ctx context.Context, param InitializeRequest) (result InitializeResult, err error) {
	err = c.Session.Exchange(ctx, "initialize", param, &result)
	if err == nil {
		err = c.Session.Send(ctx, Message{
			Method: "notifications/initialized",
		})
	}
	c.Session.ServerCapabilities = &result.Capabilities
	return
}

func (c *Client) ReadResource(ctx context.Context, uri string) (*ReadResourceResult, error) {
	var result ReadResourceResult
	err := c.Session.Exchange(ctx, "resources/read", ReadResourceRequest{
		URI: uri,
	}, &result)
	return &result, err
}

func (c *Client) ListResourceTemplates(ctx context.Context) (*ListResourceTemplatesResult, error) {
	var result ListResourceTemplatesResult
	if c.Session.ServerCapabilities == nil || c.Session.ServerCapabilities.Resources == nil {
		return &result, nil
	}
	err := c.Session.Exchange(ctx, "resources/templates/list", struct{}{}, &result)
	return &result, err
}

func (c *Client) ListResources(ctx context.Context) (*ListResourcesResult, error) {
	var result ListResourcesResult
	if c.Session.ServerCapabilities == nil || c.Session.ServerCapabilities.Resources == nil {
		return &result, nil
	}
	err := c.Session.Exchange(ctx, "resources/list", struct{}{}, &result)
	return &result, err
}

func (c *Client) ListPrompts(ctx context.Context) (*ListPromptsResult, error) {
	var prompts ListPromptsResult
	if c.Session.ServerCapabilities == nil || c.Session.ServerCapabilities.Prompts == nil {
		return &prompts, nil
	}
	err := c.Session.Exchange(ctx, "prompts/list", struct{}{}, &prompts)
	return &prompts, err
}

func (c *Client) GetPrompt(ctx context.Context, name string, args map[string]string) (*GetPromptResult, error) {
	var result GetPromptResult
	err := c.Session.Exchange(ctx, "prompts/get", GetPromptRequest{
		Name:      name,
		Arguments: args,
	}, &result)
	return &result, err
}

func (c *Client) ListTools(ctx context.Context) (*ListToolsResult, error) {
	var tools ListToolsResult
	err := c.Session.Exchange(ctx, "tools/list", struct{}{}, &tools)
	return &tools, err
}

type CallOption struct {
	ID            string
	ProgressToken any
}

func CompleteCallOptions(opts ...CallOption) CallOption {
	var all CallOption
	for _, opt := range opts {
		if opt.ID != "" {
			all.ID = opt.ID
		}
		if opt.ProgressToken != nil {
			all.ProgressToken = opt.ProgressToken
		}
	}
	return all
}

func (c *Client) Call(ctx context.Context, tool string, args any, opts ...CallOption) (result *CallToolResult, err error) {
	opt := CompleteCallOptions(opts...)
	result = new(CallToolResult)

	err = c.Session.Exchange(ctx, "tools/call", struct {
		Name      string `json:"name"`
		Arguments any    `json:"arguments,omitempty"`
	}{
		Name:      tool,
		Arguments: args,
	}, result, ExchangeOption{
		ID:            opt.ID,
		ProgressToken: opt.ProgressToken,
	})

	return
}
