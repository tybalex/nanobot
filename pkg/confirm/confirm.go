package confirm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/obot-platform/nanobot/pkg/mcp"
	"github.com/obot-platform/nanobot/pkg/types"
	"github.com/obot-platform/nanobot/pkg/uuid"
)

const Timeout = 15 * time.Minute

type Service struct {
	cond     sync.Cond
	requests map[string]request
}

func NewService() *Service {
	return &Service{
		cond:     sync.Cond{L: &sync.Mutex{}},
		requests: make(map[string]request),
	}
}

type request struct {
	ID            string          `json:"id"`
	RequestedTime time.Time       `json:"requestedTime"`
	Accepted      *bool           `json:"-"`
	MCPServer     string          `json:"mcpServer,omitempty"`
	ToolName      string          `json:"toolName,omitempty"`
	Tool          mcp.Tool        `json:"tool,omitempty"`
	Invocation    *types.ToolCall `json:"invocation,omitempty"`
}

func (s *Service) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				s.cond.L.Lock()
				for _, req := range s.requests {
					if req.RequestedTime.Add(Timeout).Before(time.Now()) {
						delete(s.requests, req.ID)
					}
				}
				// We always broadcast so that waiters have a chance to check their ctx.Done()
				s.cond.Broadcast()
				s.cond.L.Unlock()
			}
		}
	}()
}

func (s *Service) Reply(id string, accepted bool) {
	if s == nil {
		return
	}

	s.cond.L.Lock()
	defer s.cond.L.Unlock()

	if req, ok := s.requests[id]; ok {
		req.Accepted = &accepted
		s.requests[id] = req
	}
	s.cond.Broadcast()
}

func (s *Service) Confirm(ctx context.Context, session *mcp.Session, target types.TargetMapping, funcCall *types.ToolCall) error {
	if s == nil {
		return nil
	}

	uid := uuid.String()
	req := request{
		ID:            uid,
		RequestedTime: time.Now(),
		MCPServer:     target.MCPServer,
		ToolName:      target.TargetName,
		Tool:          target.Target.(mcp.Tool),
		Invocation:    funcCall,
	}

	s.cond.L.Lock()
	s.requests[uid] = req
	s.cond.L.Unlock()

	for session.Parent != nil {
		session = session.Parent
	}

	err := session.SendPayload(ctx, "notifications/message", mcp.LoggingMessage{
		Level: "info",
		Data: map[string]any{
			"type":    "nanobot/confirm",
			"request": req,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to send confirmation message: %w", err)
	}

	return s.waitAccepted(ctx, uid)
}

func (s *Service) waitAccepted(ctx context.Context, id string) error {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if req, ok := s.requests[id]; ok {
			if req.Accepted != nil {
				if *req.Accepted {
					return nil
				}
				return fmt.Errorf("request %s was rejected", id)
			}
		} else {
			return fmt.Errorf("confirmation %s timed out", id)
		}
		s.cond.Wait()
	}
}
