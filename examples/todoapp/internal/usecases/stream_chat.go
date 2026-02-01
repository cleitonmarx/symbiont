package usecases

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
)

const (
	// Maximum number of chat history messages to include in the context
	MAX_CHAT_HISTORY_MESSAGES = 10
	// Maximum number of todos to include in the context
	MAX_TODO_CONTEXT = 20
)

//go:embed prompts/chat.yml
var chatPrompt embed.FS

// StreamChat defines the interface for the StreamChat use case
type StreamChat interface {
	Execute(ctx context.Context, userMessage string, onEvent domain.LLMStreamEventCallback) error
}

// StreamChatImpl is the implementation of the StreamChat use case
type StreamChatImpl struct {
	chatMessageRepo   domain.ChatMessageRepository
	timeProvider      domain.CurrentTimeProvider
	llmClient         domain.LLMClient
	llmToolRegistry   LLMToolRegistry
	llmModel          string
	llmEmbeddingModel string
}

// NewStreamChatImpl creates a new instance of StreamChatImpl
func NewStreamChatImpl(
	chatMessageRepo domain.ChatMessageRepository,
	timeProvider domain.CurrentTimeProvider,
	llmClient domain.LLMClient,
	llmToolRegistry LLMToolRegistry,
	llmModel string,
	llmEmbeddingModel string,
) StreamChatImpl {
	return StreamChatImpl{
		chatMessageRepo:   chatMessageRepo,
		timeProvider:      timeProvider,
		llmClient:         llmClient,
		llmToolRegistry:   llmToolRegistry,
		llmModel:          llmModel,
		llmEmbeddingModel: llmEmbeddingModel,
	}
}

// Execute streams a chat response and persists the conversation
func (sc StreamChatImpl) Execute(ctx context.Context, userMessage string, onEvent domain.LLMStreamEventCallback) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	// Fetch chat history and append user message
	messages, err := sc.fetchChatHistory(spanCtx)
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}
	messages = append(messages, domain.LLMChatMessage{
		Role:    domain.ChatRole_User,
		Content: userMessage,
	})

	req := domain.LLMChatRequest{
		Model:       sc.llmModel,
		Messages:    messages,
		Stream:      true,
		Temperature: common.Ptr(0.7),
		TopP:        common.Ptr(0.9),
		Tools:       sc.llmToolRegistry.List(),
	}

	var (
		assistantMsgContent strings.Builder
		chatMessages        []*domain.ChatMessage
		assistantMsgID      uuid.UUID
	)

	// Append user message first
	userMsg := &domain.ChatMessage{
		ConversationID: domain.GlobalConversationID,
		ChatRole:       domain.ChatRole_User,
		Content:        userMessage,
		Model:          req.Model,
		CreatedAt:      sc.timeProvider.Now().UTC(),
	}
	chatMessages = append(chatMessages, userMsg)

	for continueChatStreaming := true; continueChatStreaming; {
		continueChatStreaming = false
		err = sc.llmClient.ChatStream(spanCtx, req, func(eventType domain.LLMStreamEventType, data any) error {
			switch eventType {
			case domain.LLMStreamEventType_Meta:
				meta := data.(domain.LLMStreamEventMeta)
				assistantMsgID = meta.AssistantMessageID
				userMsg.ID = meta.UserMessageID

			case domain.LLMStreamEventType_FunctionCall:
				if err := onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "⏳ Processing request...\n\n"}); err != nil {
					return err
				}
				continueChatStreaming = true
				fc := data.(domain.LLMStreamEventFunctionCall)
				// Append assistant message for function call
				assistantMsg := &domain.ChatMessage{
					ID:             uuid.New(),
					ConversationID: domain.GlobalConversationID,
					ChatRole:       domain.ChatRole_Assistant,
					ToolCalls:      []domain.LLMStreamEventFunctionCall{fc},
					Model:          req.Model,
					CreatedAt:      sc.timeProvider.Now().UTC(),
				}
				chatMessages = append(chatMessages, assistantMsg)

				// Process and append tool message
				toolMessage := sc.llmToolRegistry.Call(spanCtx, fc)
				toolMsg := &domain.ChatMessage{
					ID:             uuid.New(),
					ConversationID: domain.GlobalConversationID,
					ChatRole:       domain.ChatRole_Tool,
					ToolCallID:     &fc.ID,
					Content:        toolMessage.Content,
					Model:          req.Model,
					// Increment CreatedAt to ensure ordering
					CreatedAt: sc.timeProvider.Now().UTC().Add(3 * time.Millisecond),
				}
				chatMessages = append(chatMessages, toolMsg)

				req.Messages = append(req.Messages,
					domain.LLMChatMessage{
						Role:      domain.ChatRole_Assistant,
						ToolCalls: []domain.LLMStreamEventFunctionCall{fc},
					},
					toolMessage,
				)
			case domain.LLMStreamEventType_Delta:
				delta := data.(domain.LLMStreamEventDelta)
				assistantMsgContent.WriteString(delta.Text)
				if err := onEvent(eventType, data); err != nil {
					return err
				}
			case domain.LLMStreamEventType_Done:
				if err := onEvent(eventType, data); err != nil {
					return err
				}
			}
			return nil
		})
		if tracing.RecordErrorAndStatus(span, err) {
			return err
		}
	}

	// Append the final assistant message with the full content only if there is content
	if assistantContent := assistantMsgContent.String(); assistantContent != "" {
		assistantMsg := &domain.ChatMessage{
			ID:             assistantMsgID,
			ConversationID: domain.GlobalConversationID,
			ChatRole:       domain.ChatRole_Assistant,
			Content:        assistantContent,
			Model:          req.Model,
			CreatedAt:      sc.timeProvider.Now().UTC(),
		}
		chatMessages = append(chatMessages, assistantMsg)
	}

	// Persist all messages in order
	for _, msg := range chatMessages {
		if err := sc.chatMessageRepo.CreateChatMessage(spanCtx, *msg); tracing.RecordErrorAndStatus(span, err) {
			return err
		}
	}
	return nil
}

// buildSystemPrompt creates a system prompt with current todos context
func (sc StreamChatImpl) buildSystemPrompt() ([]domain.LLMChatMessage, error) {
	file, err := chatPrompt.Open("prompts/chat.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to open chat prompt: %w", err)
	}
	defer file.Close() //nolint:errcheck

	messages := []domain.LLMChatMessage{}
	err = yaml.NewDecoder(file).Decode(&messages)
	if err != nil {
		return nil, fmt.Errorf("failed to decode summary prompt: %w", err)
	}
	for i, msg := range messages {
		if msg.Role == domain.ChatRole_Developer {
			messages[i].Content = fmt.Sprintf(
				msg.Content,
				sc.timeProvider.Now().Unix(),
				sc.timeProvider.Now().Format(time.DateOnly),
			)
		}
	}
	// Fetch current todos for context

	return messages, nil
}

func (sc StreamChatImpl) fetchChatHistory(ctx context.Context) ([]domain.LLMChatMessage, error) {
	// Build system prompt with todo context
	systemPrompt, err := sc.buildSystemPrompt()
	if err != nil {
		return nil, err
	}

	// Load prior conversation to preserve context
	history, _, err := sc.chatMessageRepo.ListChatMessages(ctx, MAX_CHAT_HISTORY_MESSAGES)
	if err != nil {
		return nil, err
	}

	// Build chat request: system + history (excluding old system messages) + current user turn
	messages := make([]domain.LLMChatMessage, 0, len(systemPrompt)+len(history)+1)
	messages = append(messages, systemPrompt...)

	//Remove orfaned tool messages from history
	// If the first message in history is a tool message, remove it
	if len(history) > 0 {
		if history[0].ChatRole == domain.ChatRole_Tool {
			history = history[1:]
		}
	}

	// Append prior conversation history, skipping previous system messages
	for _, msg := range history {
		if msg.ChatRole != domain.ChatRole_System {
			messages = append(messages, domain.LLMChatMessage{
				Role:       msg.ChatRole,
				Content:    msg.Content,
				ToolCallID: msg.ToolCallID,
				ToolCalls:  msg.ToolCalls,
			})
		}
	}
	return messages, nil
}

// InitStreamChat is the initializer for the StreamChat use case
type InitStreamChat struct {
	ChatMessageRepo domain.ChatMessageRepository `resolve:""`
	TimeProvider    domain.CurrentTimeProvider   `resolve:""`
	Uow             domain.UnitOfWork            `resolve:""`
	TodoCreator     TodoCreator                  `resolve:""`
	TodoUpdater     TodoUpdater                  `resolve:""`
	TodoDeleter     TodoDeleter                  `resolve:""`
	TodoRepo        domain.TodoRepository        `resolve:""`
	LLMClient       domain.LLMClient             `resolve:""`
	LLMModel        string                       `config:"LLM_MODEL"`
	EmbeddingModel  string                       `config:"LLM_EMBEDDING_MODEL"`
}

// Initialize registers the StreamChat use case in the dependency container
func (i InitStreamChat) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[StreamChat](NewStreamChatImpl(
		i.ChatMessageRepo,
		i.TimeProvider,
		i.LLMClient,
		NewLLMToolManager(
			NewTodoFetcherTool(
				i.TodoRepo,
				i.LLMClient,
				i.EmbeddingModel,
			),
			NewTodoCreatorTool(
				i.Uow,
				i.TodoCreator,
			),
			NewTodoUpdaterTool(
				i.Uow,
				i.TodoUpdater,
			),
			NewTodoDeleterTool(
				i.Uow,
				i.TodoDeleter,
			),
		),
		i.LLMModel,
		i.EmbeddingModel,
	))
	return ctx, nil
}
