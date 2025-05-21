package llm

import (
	"context"
	"strings"

	"github.com/obot-platform/nanobot/pkg/llm/anthropic"
	"github.com/obot-platform/nanobot/pkg/llm/responses"
	"github.com/obot-platform/nanobot/pkg/types"
)

var _ types.Completer = (*Client)(nil)

type Config struct {
	Responses responses.Config
	Anthropic anthropic.Config
}

func NewClient(cfg Config, config types.Config) *Client {
	return &Client{
		responses: responses.NewClient(cfg.Responses, config),
		anthropic: anthropic.NewClient(cfg.Anthropic, config),
	}
}

type Client struct {
	responses *responses.Client
	anthropic *anthropic.Client
}

func (c Client) Complete(ctx context.Context, req types.CompletionRequest, opts ...types.CompletionOptions) (*types.CompletionResponse, error) {
	if strings.HasPrefix(req.Model, "claude") {
		return c.anthropic.Complete(ctx, req, opts...)
	}
	return c.responses.Complete(ctx, req, opts...)
}
