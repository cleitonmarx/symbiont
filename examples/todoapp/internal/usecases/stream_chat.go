package usecases

import (
	"context"
	"encoding/json"
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
	TodoRepo        domain.TodoRepository        `resolve:""`
	LLMClient       domain.LLMClient             `resolve:""`
	llmModel        string
}

// NewStreamChatImpl creates a new instance of StreamChatImpl
func NewStreamChatImpl(chatMessageRepo domain.ChatMessageRepository, todoRepo domain.TodoRepository, llmClient domain.LLMClient, llmModel string) StreamChatImpl {
	return StreamChatImpl{
		ChatMessageRepo: chatMessageRepo,
		TodoRepo:        todoRepo,
		LLMClient:       llmClient,
		llmModel:        llmModel,
	}
}

// buildTodosJSON creates the todos JSON for the prompt
func buildTodosJSON(todos []domain.Todo) string {
	jsonBytes, _ := json.Marshal(todos)
	return string(jsonBytes)
}

// buildSystemPrompt creates a system prompt with current todos context
func (sc StreamChatImpl) buildSystemPrompt(ctx context.Context) (string, error) {
	// Fetch all todos
	todos, _, err := sc.TodoRepo.ListTodos(ctx, 1, 1000)
	if err != nil {
		return "", err
	}

	// Build todos JSON
	todosJSON := buildTodosJSON(todos)

	// Build prompt with todo context
	prompt := "You are a helpful assistant for managing todos.\n\n"
	prompt += "Current todos:\n"
	prompt += todosJSON + "\n\n"
	prompt += "Help the user manage their todos effectively."

	return prompt, nil
}

// Execute streams a chat response and persists the conversation
func (sc StreamChatImpl) Execute(ctx context.Context, userMessage string, onEvent domain.LLMStreamEventCallback) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	// Build system prompt with todo context
	systemPrompt, err := sc.buildSystemPrompt(spanCtx)
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	// Load prior conversation to preserve context
	history, _, err := sc.ChatMessageRepo.ListChatMessages(spanCtx, 0) // full history (or paginated by repo)
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	// Build chat request: system + history + current user turn
	messages := make([]domain.LLMChatMessage, 0, len(history)+2)
	messages = append(messages, domain.LLMChatMessage{
		Role:    domain.ChatRole("system"),
		Content: systemPrompt,
	})
	for _, m := range history {
		messages = append(messages, domain.LLMChatMessage{
			Role:    domain.ChatRole(m.ChatRole),
			Content: m.Content,
		})
	}
	messages = append(messages, domain.LLMChatMessage{
		Role:    domain.ChatRole("user"),
		Content: userMessage,
	})

	req := domain.LLMChatRequest{
		Model:    sc.llmModel,
		Messages: messages,
		Stream:   true,
	}

	// Track metadata and accumulate content
	var assistantMessageID uuid.UUID
	var userMessageID uuid.UUID
	var fullContent strings.Builder
	var finalUsage *domain.LLMUsage

	// Stream from LLM client
	err = sc.LLMClient.ChatStream(spanCtx, req, func(eventType string, data interface{}) error {
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
	TodoRepo        domain.TodoRepository        `resolve:""`
	LLMClient       domain.LLMClient             `resolve:""`
	LLMModel        string                       `config:"LLM_MODEL" default:"ai/gpt-oss"`
}

// Initialize registers the StreamChat use case in the dependency container
func (i InitStreamChat) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[StreamChat](NewStreamChatImpl(i.ChatMessageRepo, i.TodoRepo, i.LLMClient, i.LLMModel))
	return ctx, nil
}
