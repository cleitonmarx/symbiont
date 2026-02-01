package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStreamChatImpl_Execute(t *testing.T) {
	userMsgID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	assistantMsgID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		userMessage string
		setupDomain func(
			*domain.MockChatMessageRepository,
			*domain.MockCurrentTimeProvider,
			*domain.MockLLMClient,
			*MockLLMToolRegistry,
		)
		expectErr       bool
		expectedContent string
	}{
		"success": {
			userMessage: "Hello, how are you?",
			setupDomain: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *MockLLMToolRegistry,
			) {

				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMTool{})

				timeProvider.EXPECT().
					Now().
					Return(fixedTime).Twice()

				// history: empty
				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{
						{
							ID:             uuid.New(),
							ConversationID: domain.GlobalConversationID,
							ChatRole:       domain.ChatRole_User,
							Content:        "Previous message",
							CreatedAt:      fixedTime.Add(-time.Minute),
						},
					}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) {
						// assert.Contains(t, req.Messages[0].Content, "Task: Test Todo | Status: OPEN | Due: 2026-01-24")

						// Simulate events
						_ = onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
							ConversationID:     domain.GlobalConversationID,
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
							StartedAt:          fixedTime,
						})
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "I'm "})
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "doing "})
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "great!"})
						_ = onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				// user and assistant saves...
				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole_User
					})).
					Run(func(ctx context.Context, msg domain.ChatMessage) {
						assert.Equal(t, userMsgID, msg.ID)
						assert.Equal(t, "Hello, how are you?", msg.Content)
					}).
					Return(nil).
					Once()

				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole_Assistant
					})).
					Run(func(ctx context.Context, msg domain.ChatMessage) {
						assert.Equal(t, assistantMsgID, msg.ID)
						assert.Equal(t, "I'm doing great!", msg.Content)
					}).
					Return(nil).
					Once()
			},
			expectErr:       false,
			expectedContent: "I'm doing great!",
		},
		"success-with-function-call": {
			userMessage: "Call a tool",
			setupDomain: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *MockLLMToolRegistry,
			) {
				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMTool{})

				toolRegistry.EXPECT().
					Call(
						mock.Anything,
						domain.LLMStreamEventFunctionCall{
							ID:        "func-123",
							Index:     0,
							Function:  "list_todos",
							Arguments: "{\"page\": 1, \"page_size\": 5, \"search_term\": \"searchTerm\"}",
						},
					).
					Return(domain.LLMChatMessage{Role: domain.ChatRole_Tool})

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				timeProvider.EXPECT().
					Now().
					Return(fixedTime)

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					RunAndReturn(
						toolFunctionCallback(userMsgID, assistantMsgID, fixedTime),
					)

				// user and assistant saves...
				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole_User && msg.Content == "Call a tool"
					})).
					Return(nil).
					Once()

				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole_Assistant && len(msg.ToolCalls) > 0
					})).
					Return(nil).
					Once()

				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole_Tool &&
							msg.ToolCallID != nil &&
							*msg.ToolCallID == "func-123"
					})).
					Return(nil).
					Once()

				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole_Assistant &&
							msg.Content == "Tool called successfully."
					})).
					Return(nil).
					Once()
			},
			expectErr:       false,
			expectedContent: "",
		},

		"llm-chatstream-error": {
			userMessage: "Test",
			setupDomain: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *MockLLMToolRegistry,
			) {

				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMTool{})

				timeProvider.EXPECT().
					Now().
					Return(fixedTime)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("llm error"))
			},
			expectErr: true,
		},
		"user-message-save-error": {
			userMessage: "Test",
			setupDomain: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *MockLLMToolRegistry,
			) {

				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMTool{})

				timeProvider.EXPECT().
					Now().
					Return(fixedTime)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) {
						_ = onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
							ConversationID:     domain.GlobalConversationID,
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
							StartedAt:          fixedTime,
						})
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "OK"})
						_ = onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole_User
					})).
					Return(errors.New("db error")).
					Once()
			},
			expectErr: true,
		},
		"assistant-message-save-error": {
			userMessage: "Test",
			setupDomain: func(
				chatRepo *domain.MockChatMessageRepository,
				timeProvider *domain.MockCurrentTimeProvider,
				client *domain.MockLLMClient,
				toolRegistry *MockLLMToolRegistry,
			) {

				toolRegistry.EXPECT().
					List().
					Return([]domain.LLMTool{})

				timeProvider.EXPECT().
					Now().
					Return(fixedTime)

				chatRepo.EXPECT().
					ListChatMessages(mock.Anything, MAX_CHAT_HISTORY_MESSAGES).
					Return([]domain.ChatMessage{}, false, nil).
					Once()

				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) {
						_ = onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
							ConversationID:     domain.GlobalConversationID,
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
							StartedAt:          fixedTime,
						})
						_ = onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "OK"})
						_ = onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole_User
					})).
					Return(nil).
					Once()

				chatRepo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole_Assistant
					})).
					Return(errors.New("db error")).
					Once()
			},
			expectErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			chatRepo := domain.NewMockChatMessageRepository(t)
			timeProvider := domain.NewMockCurrentTimeProvider(t)
			llmClient := domain.NewMockLLMClient(t)
			lltToolRegistry := NewMockLLMToolRegistry(t)
			tt.setupDomain(chatRepo, timeProvider, llmClient, lltToolRegistry)

			useCase := NewStreamChatImpl(chatRepo, timeProvider, llmClient, lltToolRegistry, "test-model", "test-embedding-model")

			var capturedContent string
			err := useCase.Execute(context.Background(), tt.userMessage, func(eventType domain.LLMStreamEventType, data any) error {
				if eventType == domain.LLMStreamEventType_Delta {
					delta := data.(domain.LLMStreamEventDelta)
					capturedContent += delta.Text
				}
				return nil
			})

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectedContent != "" {
					assert.Equal(t, tt.expectedContent, capturedContent)
				}
			}
		})
	}
}

func toolFunctionCallback(userMsgID, assistantMsgID uuid.UUID, fixedTime time.Time) func(_ context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) error {
	return func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) error {
		if err := onEvent(domain.LLMStreamEventType_Meta, domain.LLMStreamEventMeta{
			ConversationID:     domain.GlobalConversationID,
			UserMessageID:      userMsgID,
			AssistantMessageID: assistantMsgID,
			StartedAt:          fixedTime,
		}); err != nil {
			return err
		}

		lastMsg := req.Messages[len(req.Messages)-1]
		if lastMsg.Content == "Call a tool" {
			err := onEvent(domain.LLMStreamEventType_FunctionCall, domain.LLMStreamEventFunctionCall{
				ID:        "func-123",
				Index:     0,
				Function:  "list_todos",
				Arguments: `{"page": 1, "page_size": 5, "search_term": "searchTerm"}`,
			})
			return err
		}

		if lastMsg.Role == domain.ChatRole_Tool {
			if err := onEvent(domain.LLMStreamEventType_Delta, domain.LLMStreamEventDelta{Text: "Tool called successfully."}); err != nil {
				return err
			}
		}

		if err := onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
			AssistantMessageID: assistantMsgID.String(),
			CompletedAt:        fixedTime.Format(time.RFC3339),
		}); err != nil {
			return err
		}
		return nil
	}
}

func TestInitStreamChat_Initialize(t *testing.T) {
	i := InitStreamChat{}

	ctx, err := i.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	// Verify that the StreamChat use case is registered
	streamChatUseCase, err := depend.Resolve[StreamChat]()
	assert.NoError(t, err)
	assert.NotNil(t, streamChatUseCase)
}
