package sampling

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"

	"github.com/obot-platform/nanobot/pkg/mcp"
	"github.com/obot-platform/nanobot/pkg/types"
)

type Sampler struct {
	config    types.Config
	completer types.Completer
}

func NewSampler(config types.Config, completer types.Completer) *Sampler {
	return &Sampler{
		config:    config,
		completer: completer,
	}
}

type scored struct {
	score float64
	model string
}

func (s *Sampler) sortModels(preferences mcp.ModelPreferences) []string {
	var scoredModels []scored

	for _, modelKey := range slices.Sorted(maps.Keys(s.config.Agents)) {
		model := s.config.Agents[modelKey]
		cost := model.Cost
		if preferences.CostPriority != nil {
			cost *= *preferences.CostPriority
		}
		speed := model.Speed
		if preferences.SpeedPriority != nil {
			speed *= *preferences.SpeedPriority
		}
		intelligence := model.Intelligence
		if preferences.IntelligencePriority != nil {
			intelligence *= *preferences.IntelligencePriority
		}
		scoredModels = append(scoredModels, scored{
			score: cost + speed + intelligence,
			model: modelKey,
		})
	}

	sort.Slice(scoredModels, func(i, j int) bool {
		return scoredModels[i].score > scoredModels[j].score
	})

	models := make([]string, len(scoredModels))
	for i, scoredModel := range scoredModels {
		models[i] = scoredModel.model
	}
	return models
}

func (s *Sampler) getMatchingModel(req *mcp.CreateMessageRequest) (string, bool) {
	// Agent by name
	for _, model := range req.ModelPreferences.Hints {
		if _, ok := s.config.Agents[model.Name]; ok {
			return model.Name, true
		}
	}

	// Model by alias
	for _, model := range req.ModelPreferences.Hints {
		for _, modelKey := range slices.Sorted(maps.Keys(s.config.Agents)) {
			if slices.Contains(s.config.Agents[modelKey].Aliases, model.Name) {
				return modelKey, true
			}
		}
	}

	models := s.sortModels(req.ModelPreferences)
	if len(models) == 0 {
		return "", false
	}

	return models[0], true
}

type SamplerOptions struct {
	Progress chan<- json.RawMessage
	Continue bool
}

func completeSamplerOptions(opts ...SamplerOptions) SamplerOptions {
	var all SamplerOptions
	for _, opt := range opts {
		if opt.Progress != nil {
			if all.Progress != nil {
				panic("multiple progress handlers provided")
			}
			all.Progress = opt.Progress
		}
		if opt.Continue {
			all.Continue = true
		}
	}
	return all
}

func (s *Sampler) Sample(ctx context.Context, req mcp.CreateMessageRequest, opts ...SamplerOptions) (result mcp.CreateMessageResult, _ error) {
	opt := completeSamplerOptions(opts...)

	model, ok := s.getMatchingModel(&req)
	if !ok {
		return result, fmt.Errorf("no matching model found")
	}

	request := types.CompletionRequest{
		Model: model,
	}

	if req.MaxTokens != 0 {
		request.MaxTokens = req.MaxTokens
	}
	if req.SystemPrompt != "" {
		request.SystemPrompt = req.SystemPrompt
	}
	if req.Temperature != nil {
		request.Temperature = req.Temperature
	}

	for _, content := range req.Messages {
		request.Input = append(request.Input, types.CompletionInput{
			Message: &content,
		})
	}

	resp, err := s.completer.Complete(ctx, request, types.CompletionOptions{
		Progress: opt.Progress,
	})
	if err != nil {
		return result, err
	}

	result.Model = request.Model

	for _, output := range resp.Output {
		if output.Message == nil {
			continue
		}
		result.Role = output.Message.Role
		result.Content = output.Message.Content
	}

	if result.Content.Type == "" {
		result.Content.Type = "text"
		result.Content.Text = "[NO CONTENT]"
	}

	if result.Role == "" {
		result.Role = "assistant"
	}

	return result, nil
}
