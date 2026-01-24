package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStreamChatImpl_Execute(t *testing.T) {
	userMsgID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	assistantMsgID := uuid.MustParse("223e4567-e89b-12d3-a456-426614174001")
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		userMessage     string
		setupMocks      func(*mocks.MockChatMessageRepository, *mocks.MockLLMClient)
		expectErr       bool
		expectedContent string
	}{
		"success-with-usage": {
			userMessage: "Hello, how are you?",
			setupMocks: func(repo *mocks.MockChatMessageRepository, client *mocks.MockLLMClient) {
				// Mock LLM streaming
				client.EXPECT().
					ChatStream(mock.Anything, mock.MatchedBy(func(req domain.LLMChatRequest) bool {
						return req.Model == "ai/gpt-oss" &&
							len(req.Messages) == 2 &&
							req.Messages[0].Role == domain.ChatRole("system") &&
							req.Messages[1].Content == "Hello, how are you?"
					}), mock.Anything).
					Run(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) {
						// Simulate meta event
						_ = onEvent("meta", domain.LLMStreamEventMeta{
							ConversationID:     domain.GlobalConversationID,
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
							StartedAt:          fixedTime,
						})
						// Simulate delta events
						_ = onEvent("delta", domain.LLMStreamEventDelta{Text: "I'm "})
						_ = onEvent("delta", domain.LLMStreamEventDelta{Text: "doing "})
						_ = onEvent("delta", domain.LLMStreamEventDelta{Text: "great!"})
						// Simulate done event
						_ = onEvent("done", domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
							Usage: &domain.LLMUsage{
								PromptTokens:     10,
								CompletionTokens: 5,
								TotalTokens:      15,
							},
						})
					}).
					Return(nil)

				// Expect user message to be saved
				repo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ID == userMsgID &&
							msg.ConversationID == domain.GlobalConversationID &&
							msg.ChatRole == domain.ChatRole("user") &&
							msg.Content == "Hello, how are you?" &&
							msg.Model == "ai/gpt-oss"
					})).
					Return(nil)

				// Expect assistant message to be saved
				repo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ID == assistantMsgID &&
							msg.ConversationID == domain.GlobalConversationID &&
							msg.ChatRole == domain.ChatRole("assistant") &&
							msg.Content == "I'm doing great!" &&
							msg.Model == "ai/gpt-oss" &&
							msg.PromptTokens == 10 &&
							msg.CompletionTokens == 5
					})).
					Return(nil)
			},
			expectErr:       false,
			expectedContent: "I'm doing great!",
		},
		"success-without-usage": {
			userMessage: "Test",
			setupMocks: func(repo *mocks.MockChatMessageRepository, client *mocks.MockLLMClient) {
				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) {
						_ = onEvent("meta", domain.LLMStreamEventMeta{
							ConversationID:     domain.GlobalConversationID,
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
							StartedAt:          fixedTime,
						})
						_ = onEvent("delta", domain.LLMStreamEventDelta{Text: "OK"})
						_ = onEvent("done", domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
							Usage:              nil, // No usage
						})
					}).
					Return(nil)

				repo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole("user")
					})).
					Return(nil)

				repo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole("assistant") &&
							msg.PromptTokens == 0 &&
							msg.CompletionTokens == 0
					})).
					Return(nil)
			},
			expectErr:       false,
			expectedContent: "OK",
		},
		"llm-client-error": {
			userMessage: "Test",
			setupMocks: func(repo *mocks.MockChatMessageRepository, client *mocks.MockLLMClient) {
				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("llm error"))
			},
			expectErr: true,
		},
		"user-message-save-error": {
			userMessage: "Test",
			setupMocks: func(repo *mocks.MockChatMessageRepository, client *mocks.MockLLMClient) {
				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) {
						_ = onEvent("meta", domain.LLMStreamEventMeta{
							ConversationID:     domain.GlobalConversationID,
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
							StartedAt:          fixedTime,
						})
						_ = onEvent("delta", domain.LLMStreamEventDelta{Text: "OK"})
						_ = onEvent("done", domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				repo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole("user")
					})).
					Return(errors.New("db error"))
			},
			expectErr: true,
		},
		"assistant-message-save-error": {
			userMessage: "Test",
			setupMocks: func(repo *mocks.MockChatMessageRepository, client *mocks.MockLLMClient) {
				client.EXPECT().
					ChatStream(mock.Anything, mock.Anything, mock.Anything).
					Run(func(ctx context.Context, req domain.LLMChatRequest, onEvent domain.LLMStreamEventCallback) {
						_ = onEvent("meta", domain.LLMStreamEventMeta{
							ConversationID:     domain.GlobalConversationID,
							UserMessageID:      userMsgID,
							AssistantMessageID: assistantMsgID,
							StartedAt:          fixedTime,
						})
						_ = onEvent("delta", domain.LLMStreamEventDelta{Text: "OK"})
						_ = onEvent("done", domain.LLMStreamEventDone{
							AssistantMessageID: assistantMsgID.String(),
							CompletedAt:        fixedTime.Format(time.RFC3339),
						})
					}).
					Return(nil)

				repo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole("user")
					})).
					Return(nil)

				repo.EXPECT().
					CreateChatMessage(mock.Anything, mock.MatchedBy(func(msg domain.ChatMessage) bool {
						return msg.ChatRole == domain.ChatRole("assistant")
					})).
					Return(errors.New("db error"))
			},
			expectErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mockRepo := mocks.NewMockChatMessageRepository(t)
			mockClient := mocks.NewMockLLMClient(t)

			tt.setupMocks(mockRepo, mockClient)

			useCase := NewStreamChatImpl(mockRepo, mockClient)

			var capturedContent string
			err := useCase.Execute(context.Background(), tt.userMessage, func(eventType string, data interface{}) error {
				if eventType == "delta" {
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

			mockRepo.AssertExpectations(t)
			mockClient.AssertExpectations(t)
		})
	}
}
