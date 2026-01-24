package modelrunner

import (
	"context"
	"errors"
	"net/http"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
)

// LLMClient adapts Docker Model Runner API Client to the domain.LLMClient interface
type LLMClient struct {
	client DRMAPIClient
}

// NewLLMClientAdapter creates a new adapter for the LLM client
func NewLLMClientAdapter(client DRMAPIClient) LLMClient {
	return LLMClient{
		client: client,
	}
}

// ChatStream implements domain.LLMClient.ChatStream by adapting the underlying client
func (a LLMClient) ChatStream(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	// Adapt domain request to adapter request
	adapterReq := ChatRequest{
		Model:    req.Model,
		Stream:   req.Stream,
		Messages: make([]ChatMessage, len(req.Messages)),
	}

	for i, msg := range req.Messages {
		adapterReq.Messages[i] = ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	// Call the underlying client with adapted callback
	return a.client.ChatStream(spanCtx, adapterReq, func(eventType string, data any) error {
		// Adapt events from adapter domain to domain layer
		switch eventType {
		case "meta":
			meta := data.(StreamEventMeta)
			return onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
				ConversationID:     meta.ConversationID,
				UserMessageID:      meta.UserMessageID,
				AssistantMessageID: meta.AssistantMessageID,
				StartedAt:          meta.StartedAt,
			})

		case "delta":
			delta := data.(StreamEventDelta)
			return onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{
				Text: delta.Text,
			})

		case "done":
			done := data.(StreamEventDone)
			var domainUsage *domain.LLMUsage
			if done.Usage != (Usage{}) {
				domainUsage = &domain.LLMUsage{
					PromptTokens:     done.Usage.PromptTokens,
					CompletionTokens: done.Usage.CompletionTokens,
					TotalTokens:      done.Usage.TotalTokens,
				}
			}
			return onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
				AssistantMessageID: done.AssistantMessageID.String(),
				CompletedAt:        done.CompletedAt,
				Usage:              domainUsage,
			})

		default:
			return onEvent(domain.LLMStreamEventType(eventType), data)
		}
	})
}

// Chat implements domain.LLMClient.Chat by adapting the underlying client
func (a LLMClient) Chat(ctx context.Context, req domain.LLMChatRequest) (string, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	// Adapt domain request to adapter request
	adapterReq := ChatRequest{
		Model:       req.Model,
		Stream:      req.Stream,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Messages:    make([]ChatMessage, len(req.Messages)),
	}

	for i, msg := range req.Messages {
		adapterReq.Messages[i] = ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		}
	}

	resp, err := a.client.Chat(spanCtx, adapterReq)
	if tracing.RecordErrorAndStatus(span, err) {
		return "", err
	}

	if len(resp.Choices) == 0 {
		err := errors.New("llm: no choices in response")
		tracing.RecordErrorAndStatus(span, err)
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

// InitLLMClient initializes the LLMClient dependency
type InitLLMClient struct {
	HttpClient *http.Client `resolve:""`
	LLMHost    string       `config:"LLM_MODEL_HOST"`
}

// Initialize registers the LLMClient in the dependency container
func (i InitLLMClient) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.LLMClient](NewLLMClientAdapter(
		NewDRMAPIClient(i.LLMHost, "", i.HttpClient),
	))
	return ctx, nil
}
