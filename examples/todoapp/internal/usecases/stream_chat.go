package usecases

import (
	"context"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/google/uuid"
)

// StreamChat defines the interface for the StreamChat use case
type StreamChat interface {
	Execute(ctx context.Context, userMessage string, onEvent domain.LLMStreamEventCallback) error
}

// StreamChatImpl is the implementation of the StreamChat use case
type StreamChatImpl struct {
	ChatMessageRepo domain.ChatMessageRepository `resolve:""`
	LLMClient       domain.LLMClient             `resolve:""`
}

// NewStreamChatImpl creates a new instance of StreamChatImpl
func NewStreamChatImpl(chatMessageRepo domain.ChatMessageRepository, llmClient domain.LLMClient) StreamChatImpl {
	return StreamChatImpl{
		ChatMessageRepo: chatMessageRepo,
		LLMClient:       llmClient,
	}
}

// Execute streams a chat response and persists the conversation
func (sc StreamChatImpl) Execute(ctx context.Context, userMessage string, onEvent domain.LLMStreamEventCallback) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	// Build chat request with system message and user message
	req := domain.LLMChatRequest{
		Model: "ai/gpt-oss",
		Messages: []domain.LLMChatMessage{
			{
				Role:    domain.ChatRole("system"),
				Content: "You are a helpful assistant for managing todos.",
			},
			{
				Role:    domain.ChatRole("user"),
				Content: userMessage,
			},
		},
		Stream: true,
	}

	// Track metadata and accumulate content
	var assistantMessageID uuid.UUID
	var userMessageID uuid.UUID
	var fullContent strings.Builder
	var finalUsage *domain.LLMUsage

	// Stream from LLM client
	err := sc.LLMClient.ChatStream(spanCtx, req, func(eventType string, data interface{}) error {
		// Forward all events to the caller
		if err := onEvent(eventType, data); err != nil {
			return err
		}

		// Capture metadata from meta event
		if eventType == "meta" {
			meta := data.(domain.LLMStreamEventMeta)
			assistantMessageID = meta.AssistantMessageID
			userMessageID = meta.UserMessageID
		}

		// Accumulate content from delta events
		if eventType == "delta" {
			delta := data.(domain.LLMStreamEventDelta)
			fullContent.WriteString(delta.Text)
		}

		// Capture usage from done event
		if eventType == "done" {
			done := data.(domain.LLMStreamEventDone)
			finalUsage = done.Usage
		}

		return nil
	})

	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	// Create and persist the user message
	userMsg := domain.ChatMessage{
		ID:             userMessageID,
		ConversationID: domain.GlobalConversationID,
		ChatRole:       domain.ChatRole("user"),
		Content:        userMessage,
		Model:          req.Model,
		CreatedAt:      time.Now().UTC(),
	}

	if err := sc.ChatMessageRepo.CreateChatMessage(spanCtx, userMsg); tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	// Create and persist the assistant message
	assistantMsg := domain.ChatMessage{
		ID:             assistantMessageID,
		ConversationID: domain.GlobalConversationID,
		ChatRole:       domain.ChatRole("assistant"),
		Content:        fullContent.String(),
		Model:          req.Model,
		CreatedAt:      time.Now().UTC(),
	}

	if finalUsage != nil {
		assistantMsg.PromptTokens = finalUsage.PromptTokens
		assistantMsg.CompletionTokens = finalUsage.CompletionTokens
	}

	if err := sc.ChatMessageRepo.CreateChatMessage(spanCtx, assistantMsg); tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	return nil
}

// InitStreamChat is the initializer for the StreamChat use case
type InitStreamChat struct {
	ChatMessageRepo domain.ChatMessageRepository `resolve:""`
	LLMClient       domain.LLMClient             `resolve:""`
}

// Initialize registers the StreamChat use case in the dependency container
func (i InitStreamChat) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[StreamChat](NewStreamChatImpl(i.ChatMessageRepo, i.LLMClient))
	return ctx, nil
}
