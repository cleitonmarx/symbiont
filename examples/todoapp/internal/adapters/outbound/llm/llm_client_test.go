package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLLMClientAdapter_ChatStream(t *testing.T) {
	userMsgID := uuid.New()
	assistantMsgID := uuid.New()
	fixedTime := time.Date(2026, 1, 24, 15, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		serverHandler   http.HandlerFunc
		expectErr       bool
		validateEvents  func(*testing.T, []string)
		validateContent func(*testing.T, []string)
		validateUsage   func(*testing.T, *domain.LLMStreamEventDone)
	}{
		"success-with-all-events-per-spec": {
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)

				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)

				flusher := w.(http.Flusher)

				// Per OpenAPI spec: meta event first
				fmt.Fprintf( //nolint:errcheck
					w,
					"event: meta\ndata: {\"conversation_id\":\"global\",\"user_message_id\":\"%s\",\"assistant_message_id\":\"%s\",\"started_at\":\"%s\"}\n\n",
					userMsgID.String(),
					assistantMsgID.String(),
					fixedTime.Format(time.RFC3339),
				)
				flusher.Flush()

				// Per OpenAPI spec: delta events with streaming text
				fmt.Fprintf( //nolint:errcheck
					w,
					"event: delta\ndata: {\"text\":\"You have 2 overdue todos\"}\n\n",
				)
				flusher.Flush()

				// Per OpenAPI spec: done event with usage
				fmt.Fprintf( //nolint:errcheck
					w,
					"event: done\ndata: {\"assistant_message_id\":\"%s\",\"completed_at\":\"%s\",\"usage\":{\"prompt_tokens\":123,\"completion_tokens\":45,\"total_tokens\":168}}\n\n",
					assistantMsgID.String(),
					fixedTime.Format(time.RFC3339),
				)
				flusher.Flush()
			},
			expectErr: false,
			validateEvents: func(t *testing.T, eventTypes []string) {
				// Per spec: must have meta, delta, and done
				assert.Contains(t, eventTypes, "meta", "spec requires meta event")
				assert.Contains(t, eventTypes, "delta", "spec requires at least one delta event")
				assert.Contains(t, eventTypes, "done", "spec requires done event")
			},
			validateContent: func(t *testing.T, deltaTexts []string) {
				// Verify delta content was captured
				assert.GreaterOrEqual(t, len(deltaTexts), 1, "should have at least one delta event")
				combined := strings.Join(deltaTexts, "")
				assert.Contains(t, combined, "overdue todos")
			},
			validateUsage: func(t *testing.T, done *domain.LLMStreamEventDone) {
				if done != nil && done.Usage != nil {
					assert.Equal(t, 123, done.Usage.PromptTokens)
					assert.Equal(t, 45, done.Usage.CompletionTokens)
					assert.Equal(t, 168, done.Usage.TotalTokens)
				}
			},
		},
		"delta-events-streaming": {
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)

				flusher := w.(http.Flusher)

				fmt.Fprintf( //nolint:errcheck
					w,
					"event: meta\ndata: {\"conversation_id\":\"global\",\"user_message_id\":\"%s\",\"assistant_message_id\":\"%s\",\"started_at\":\"%s\"}\n\n",
					userMsgID.String(),
					assistantMsgID.String(),
					fixedTime.Format(time.RFC3339),
				)
				flusher.Flush()

				// Multiple delta events as per streaming spec
				texts := []string{"Hello", " ", "world", "!"}
				for _, txt := range texts {
					fmt.Fprintf(w, "event: delta\ndata: {\"text\":\"%s\"}\n\n", txt) //nolint:errcheck
					flusher.Flush()
				}

				fmt.Fprintf( //nolint:errcheck
					w,
					"event: done\ndata: {\"assistant_message_id\":\"%s\",\"completed_at\":\"%s\"}\n\n",
					assistantMsgID.String(),
					fixedTime.Format(time.RFC3339),
				)
				flusher.Flush()
			},
			expectErr: false,
			validateEvents: func(t *testing.T, eventTypes []string) {
				assert.Contains(t, eventTypes, "meta")
				assert.Contains(t, eventTypes, "delta")
				assert.Contains(t, eventTypes, "done")
				// Count delta events
				deltaCount := 0
				for _, et := range eventTypes {
					if et == "delta" {
						deltaCount++
					}
				}
				assert.GreaterOrEqual(t, deltaCount, 1, "should have multiple delta events")
			},
			validateContent: func(t *testing.T, deltaTexts []string) {
				assert.GreaterOrEqual(t, len(deltaTexts), 1)
				combined := strings.Join(deltaTexts, "")
				assert.Equal(t, "Hello world!", combined)
			},
		},
		"delta-with-special-characters-per-spec": {
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)

				flusher := w.(http.Flusher)

				fmt.Fprintf( //nolint:errcheck
					w,
					"event: meta\ndata: {\"conversation_id\":\"global\",\"user_message_id\":\"%s\",\"assistant_message_id\":\"%s\",\"started_at\":\"%s\"}\n\n",
					userMsgID.String(),
					assistantMsgID.String(),
					fixedTime.Format(time.RFC3339),
				)
				flusher.Flush()

				// Delta with escaped JSON characters
				fmt.Fprintf(w, "event: delta\ndata: {\"text\":\"Line 1\\nLine 2\\tTabbed\"}\n\n") //nolint:errcheck
				flusher.Flush()

				fmt.Fprintf(w, "event: done\ndata: {\"assistant_message_id\":\"%s\",\"completed_at\":\"%s\"}\n\n", //nolint:errcheck
					assistantMsgID.String(), fixedTime.Format(time.RFC3339))
				flusher.Flush()
			},
			expectErr: false,
			validateEvents: func(t *testing.T, eventTypes []string) {
				assert.Contains(t, eventTypes, "delta")
			},
			validateContent: func(t *testing.T, deltaTexts []string) {
				assert.GreaterOrEqual(t, len(deltaTexts), 1)
			},
		},
		"meta-then-delta-then-done-sequence": {
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)

				flusher := w.(http.Flusher)

				// Strict event order per spec: meta, delta(s), done
				fmt.Fprintf(
					w,
					"event: meta\ndata: {\"conversation_id\":\"global\",\"user_message_id\":\"%s\",\"assistant_message_id\":\"%s\",\"started_at\":\"%s\"}\n\n", //nolint:errcheck
					userMsgID.String(),
					assistantMsgID.String(),
					fixedTime.Format(time.RFC3339),
				)
				flusher.Flush()

				fmt.Fprintf(w, "event: delta\ndata: {\"text\":\"Response text\"}\n\n") //nolint:errcheck
				flusher.Flush()

				fmt.Fprintf( //nolint:errcheck
					w,
					"event: done\ndata: {\"assistant_message_id\":\"%s\",\"completed_at\":\"%s\",\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\n",
					assistantMsgID.String(), fixedTime.Format(time.RFC3339))
				flusher.Flush()
			},
			expectErr: false,
			validateEvents: func(t *testing.T, eventTypes []string) {
				assert.Equal(t, 3, len(eventTypes), "should have exactly 3 events: meta, delta, done")
				assert.Equal(t, "meta", eventTypes[0], "first event must be meta")
				assert.Equal(t, "delta", eventTypes[1], "second event must be delta")
				assert.Equal(t, "done", eventTypes[2], "third event must be done")
			},
			validateContent: func(t *testing.T, deltaTexts []string) {
				assert.Len(t, deltaTexts, 1)
				assert.Equal(t, "Response text", deltaTexts[0])
			},
		},
		"delta-empty-text": {
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)

				flusher := w.(http.Flusher)

				fmt.Fprintf(w, "event: meta\ndata: {\"conversation_id\":\"global\",\"user_message_id\":\"%s\",\"assistant_message_id\":\"%s\",\"started_at\":\"%s\"}\n\n", //nolint:errcheck
					userMsgID.String(), assistantMsgID.String(), fixedTime.Format(time.RFC3339))
				flusher.Flush()

				// Empty delta should still be sent per spec
				fmt.Fprintf(w, "event: delta\ndata: {\"text\":\"\"}\n\n") //nolint:errcheck
				flusher.Flush()

				fmt.Fprintf(w, "event: done\ndata: {\"assistant_message_id\":\"%s\",\"completed_at\":\"%s\"}\n\n", //nolint:errcheck
					assistantMsgID.String(), fixedTime.Format(time.RFC3339))
				flusher.Flush()
			},
			expectErr: false,
			validateEvents: func(t *testing.T, eventTypes []string) {
				assert.Contains(t, eventTypes, "delta")
			},
		},
		"server-error": {
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error")) //nolint:errcheck
			},
			expectErr: true,
		},
		"connection-close-during-stream": {
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(http.StatusOK)

				flusher := w.(http.Flusher)

				fmt.Fprintf(w, "event: meta\ndata: {\"conversation_id\":\"global\",\"user_message_id\":\"%s\",\"assistant_message_id\":\"%s\",\"started_at\":\"%s\"}\n\n", //nolint:errcheck
					userMsgID.String(), assistantMsgID.String(), fixedTime.Format(time.RFC3339))
				flusher.Flush()

				// Connection closes without sending done event
			},
			expectErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			client := NewDockerModelAPIClient(server.URL, "", server.Client())

			adapter := NewLLMClientAdapter(client)

			domainReq := domain.LLMChatRequest{
				Model:  "test-model",
				Stream: true,
				Messages: []domain.LLMChatMessage{
					{Role: domain.ChatRole("system"), Content: "You are helpful"},
					{Role: domain.ChatRole("user"), Content: "Hello"},
				},
			}

			var eventTypes []string
			var deltaTexts []string
			var metaEvent *domain.LLMStreamEventMeta
			var doneEvent *domain.LLMStreamEventDone

			err := adapter.ChatStream(context.Background(), domainReq, func(eventType string, data interface{}) error {
				eventTypes = append(eventTypes, eventType)

				switch eventType {
				case "meta":
					meta := data.(domain.LLMStreamEventMeta)
					metaEvent = &meta
				case "delta":
					delta := data.(domain.LLMStreamEventDelta)
					deltaTexts = append(deltaTexts, delta.Text)
				case "done":
					done := data.(domain.LLMStreamEventDone)
					doneEvent = &done
				}
				return nil
			})

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validateEvents != nil {
					tt.validateEvents(t, eventTypes)
				}
				if tt.validateContent != nil {
					tt.validateContent(t, deltaTexts)
				}
				if tt.validateUsage != nil {
					tt.validateUsage(t, doneEvent)
				}

				if metaEvent != nil {
					assert.Equal(t, "global", metaEvent.ConversationID)
					assert.NotZero(t, metaEvent.UserMessageID)
				}

				if doneEvent != nil {
					assert.NotEmpty(t, doneEvent.AssistantMessageID)
				}
			}
		})
	}
}

func TestLLMClientAdapter_Chat(t *testing.T) {
	temp := 0.5
	topP := 0.9

	tests := map[string]struct {
		serverHandler func(*ChatRequest) http.HandlerFunc
		req           domain.LLMChatRequest
		expectErr     bool
		expectedResp  string
		validateReq   func(*testing.T, ChatRequest)
	}{
		"success-with-messages-and-params": {
			serverHandler: func(captured *ChatRequest) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, http.MethodPost, r.Method)
					if err := json.NewDecoder(r.Body).Decode(captured); err != nil {
						t.Fatalf("decode request: %v", err)
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
				}
			},
			req: domain.LLMChatRequest{
				Model:       "test-model",
				Stream:      false,
				Temperature: &temp,
				TopP:        &topP,
				Messages: []domain.LLMChatMessage{
					{Role: domain.ChatRole("system"), Content: "sys msg"},
					{Role: domain.ChatRole("user"), Content: "hi"},
				},
			},
			expectErr:    false,
			expectedResp: "ok",
			validateReq: func(t *testing.T, got ChatRequest) {
				assert.Equal(t, "test-model", got.Model)
				assert.False(t, got.Stream)
				if assert.Len(t, got.Messages, 2) {
					assert.Equal(t, "system", got.Messages[0].Role)
					assert.Equal(t, "sys msg", got.Messages[0].Content)
					assert.Equal(t, "user", got.Messages[1].Role)
					assert.Equal(t, "hi", got.Messages[1].Content)
				}
				if assert.NotNil(t, got.Temperature) {
					assert.InDelta(t, 0.5, *got.Temperature, 1e-6)
				}
				if assert.NotNil(t, got.TopP) {
					assert.InDelta(t, 0.9, *got.TopP, 1e-6)
				}
			},
		},
		"no-choices-error": {
			serverHandler: func(_ *ChatRequest) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"choices":[]}`))
				}
			},
			req: domain.LLMChatRequest{
				Model: "test-model",
			},
			expectErr: true,
		},
		"http-error": {
			serverHandler: func(_ *ChatRequest) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					http.Error(w, "boom", http.StatusInternalServerError)
				}
			},
			req: domain.LLMChatRequest{
				Model: "test-model",
			},
			expectErr: true,
		},
		"unexpected-payload-shape": {
			serverHandler: func(_ *ChatRequest) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":123}}]}`))
				}
			},
			req: domain.LLMChatRequest{
				Model: "test-model",
			},
			expectErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var capturedReq ChatRequest
			server := httptest.NewServer(tt.serverHandler(&capturedReq))
			defer server.Close()

			client := NewDockerModelAPIClient(server.URL, "", server.Client())
			adapter := NewLLMClientAdapter(client)

			resp, err := adapter.Chat(context.Background(), tt.req)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResp, resp)

			if tt.validateReq != nil {
				tt.validateReq(t, capturedReq)
			}
		})
	}
}
