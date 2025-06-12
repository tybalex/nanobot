package tools

import (
	"context"
	"fmt"
	"iter"
	"maps"
	"sync"

	"github.com/nanobot-ai/nanobot/pkg/expr"
	"github.com/nanobot-ai/nanobot/pkg/mcp"
	"github.com/nanobot-ai/nanobot/pkg/types"
	"github.com/nanobot-ai/nanobot/pkg/uuid"
	"golang.org/x/sync/errgroup"
)

type flowContext struct {
	ctx  context.Context
	opt  mcp.CallOption
	env  map[string]string
	data map[string]any
}

func (r *Service) startFlow(ctx context.Context, flowName string, args any, opt CallOptions) (*mcp.CallToolResult, error) {
	flow, ok := r.config.Flows[flowName]
	if !ok {
		return nil, fmt.Errorf("failed to find flow %s in config", flowName)
	}

	data := map[string]any{
		"id":    uuid.String(),
		"flow":  flowName,
		"input": args,
	}

	fCtx := flowContext{
		ctx: ctx,
		opt: mcp.CallOption{
			ProgressToken: opt.ProgressToken,
		},
		env:  mcp.SessionFromContext(ctx).EnvMap(),
		data: data,
	}

	return r.runSteps(fCtx, flow.Steps)
}

func (r *Service) runSteps(ctx flowContext, steps []types.Step) (*mcp.CallToolResult, error) {
	for i, step := range steps {
		if step.ID == "" {
			step.ID = uuid.String()
		}

		out, err := r.runStep(ctx, step)
		if err != nil {
			return nil, fmt.Errorf("failed to run step %d (%s): %w", i, step.ID, err)
		}

		if i == len(steps)-1 {
			// If this is the last step, we return the output directly.
			return out, nil
		}
	}

	return &mcp.CallToolResult{
		Content: make([]mcp.Content, 0),
	}, nil
}

func (r *Service) runStepForEach(ctx flowContext, step types.Step) (ret *mcp.CallToolResult, err error) {
	forEachData, err := expr.EvalList(ctx.ctx, ctx.env, ctx.data, step.ForEach)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate forEach for step %s: %w", step.ID, err)
	}

	return r.runStepEach(ctx, step, func(yield func(any) bool) {
		for _, val := range forEachData {
			if !yield(val) {
				return
			}
		}
	})
}

func (r *Service) runStepWhile(ctx flowContext, step types.Step) (ret *mcp.CallToolResult, err error) {
	var loopErr error
	defer func() {
		if err == nil && loopErr != nil {
			err = loopErr
		}
	}()

	step.Parallel = false
	return r.runStepEach(ctx, step, func(yield func(any) bool) {
		i := 0
		for {
			isTrue, err := expr.EvalBool(ctx.ctx, ctx.env, ctx.data, step.While)
			if err != nil {
				loopErr = err
				return
			}
			if !isTrue {
				return
			}
			if !yield(i) {
				return
			}
			i++
		}
	})
}

func (r *Service) runStepEach(ctx flowContext, step types.Step, forEachData iter.Seq[any]) (ret *mcp.CallToolResult, err error) {
	var (
		results     = make([]map[string]any, 0)
		itemVarName = "item"
		resultLock  sync.Mutex
		eg          errgroup.Group
	)

	if step.Parallel {
		eg.SetLimit(r.concurrency)
	} else {
		eg.SetLimit(1)
	}

	if step.ForEachVar != "" {
		itemVarName = step.ForEachVar
	}

	oldVar, hadOldVar := ctx.data[itemVarName]
	step.ForEach = nil
	step.While = ""

	for item := range forEachData {
		newCtx := ctx
		if step.Parallel {
			newCtx.data = maps.Clone(ctx.data)
		}
		ctx.data[itemVarName] = item
		eg.Go(func() error {
			result, err := r.runStep(newCtx, step)
			if err != nil {
				return fmt.Errorf("failed to run forEach step %s: %w", step.ID, err)
			}

			resultLock.Lock()
			defer resultLock.Unlock()
			results = append(results, toOutput(result))
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	if hadOldVar {
		ctx.data[itemVarName] = oldVar
	} else {
		delete(ctx.data, itemVarName)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			{
				StructuredContent: results,
			},
		},
	}, nil
}

func (r *Service) logFlowState(ctx flowContext) {
	if ctx.opt.ProgressToken == nil {
		return
	}
	session := mcp.SessionFromContext(ctx.ctx)
	if session == nil {
		return
	}

	_ = session.SendPayload(ctx.ctx, "notifications/progress", mcp.NotificationProgressRequest{
		ProgressToken: ctx.opt.ProgressToken,
		Data: map[string]any{
			"type": "nanobot/flow/state",
			"flow": ctx.data,
		},
	})
}

func getCall(step types.Step) string {
	if step.Agent.Name != "" {
		return step.Agent.Name
	}
	if step.Tool != "" {
		return step.Tool
	}
	if step.Flow != "" {
		return step.Flow
	}
	return ""
}

func toOutput(ret *mcp.CallToolResult) map[string]any {
	if ret == nil {
		return nil
	}

	output := map[string]any{
		"content": ret.Content,
		"isError": ret.IsError,
	}
	if ret.Content == nil {
		output["content"] = make([]mcp.Content, 0)
	}
	for i := len(ret.Content) - 1; i >= 0; i-- {
		if ret.Content[i].Text != "" {
			output["output"] = ret.Content[i].Text
		}
		if ret.Content[i].StructuredContent != nil {
			output["output"] = ret.Content[i].StructuredContent
		}
	}
	return output
}

func (r *Service) runStep(ctx flowContext, step types.Step) (ret *mcp.CallToolResult, err error) {
	defer func() {
		ctx.data[step.ID] = toOutput(ret)
		ctx.data["previous"] = ctx.data[step.ID]
		r.logFlowState(ctx)
	}()

	if step.ID == "" {
		step.ID = uuid.String()
	}

	call := getCall(step)
	if call != "" && len(step.Steps) > 0 {
		return nil, fmt.Errorf("step %s cannot have both agent/tool/flow (%s) and steps defined (count: %d)",
			step.ID, call, len(step.Steps))
	}

	for k, v := range step.Set {
		if v == nil {
			delete(ctx.data, k)
		} else {
			val, err := expr.EvalAny(ctx.ctx, ctx.env, ctx.data, v)
			if err != nil {
				return nil, err
			}
			ctx.data[k] = val
		}
	}

	if step.ForEach != nil {
		return r.runStepForEach(ctx, step)
	}

	if step.While != "" {
		return r.runStepWhile(ctx, step)
	}

	if step.If != "" {
		isTrue, err := expr.EvalBool(ctx.ctx, ctx.env, ctx.data, step.ForEach)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate if condition for step %s: %w", step.ID, err)
		}
		if !isTrue {
			return &mcp.CallToolResult{Content: make([]mcp.Content, 0)}, nil
		}
	}

	inputData, err := expr.EvalObject(ctx.ctx, ctx.env, ctx.data, step.Input)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate input for step %s: %w", step.ID, err)
	}

	if call != "" {
		ref := types.ParseToolRef(call)
		return r.Call(ctx.ctx, ref.Server, ref.Tool, inputData, CallOptions{
			ProgressToken: ctx.opt.ProgressToken,
			AgentOverride: step.Agent,
		})
	}

	return r.runSteps(ctx, step.Steps)
}
