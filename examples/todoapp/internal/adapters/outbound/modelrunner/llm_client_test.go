package modelrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cleitonmarx/symbiont/depend"
	//rest "github.com/cleitonmarx/symbiont/examples/todoapp/internal/adapters/inbound/http/gen"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	//"github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
)

// createStreamingServer creates a test server that sends OpenAI-style streaming chunks
func createStreamingServer(chunks []StreamChunk) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher := w.(http.Flusher)
		for _, chunk := range chunks {
			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", data) //nolint:errcheck
			flusher.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n") //nolint:errcheck
		flusher.Flush()
	}))
}

// collectStreamEvents collects all events from a stream
func collectStreamEvents(adapter LLMClient, req domain.LLMChatRequest) ([]domain.LLMStreamEventType, []string, *domain.LLMStreamEventDone, error) {
	var eventTypes []domain.LLMStreamEventType
	var deltaTexts []string
	var doneEvent *domain.LLMStreamEventDone

	err := adapter.ChatStream(context.Background(), req, func(eventType domain.LLMStreamEventType, data any) error {
		eventTypes = append(eventTypes, eventType)

		switch eventType {
		case domain.LLMStreamEventType_Delta:
			delta := data.(domain.LLMStreamEventDelta)
			deltaTexts = append(deltaTexts, delta.Text)
		case domain.LLMStreamEventType_Done:
			done := data.(domain.LLMStreamEventDone)
			doneEvent = &done
		}
		return nil
	})

	return eventTypes, deltaTexts, doneEvent, err
}

func TestLLMClientAdapter_ChatStream(t *testing.T) {
	req := domain.LLMChatRequest{
		Model: "test-model",
		Messages: []domain.LLMChatMessage{
			{Role: "user", Content: "test"},
		},
	}
	tests := map[string]struct {
		req             domain.LLMChatRequest
		chunks          []StreamChunk
		expectErr       bool
		expectedEvents  []domain.LLMStreamEventType
		expectedContent string
	}{
		"multiple-deltas": {
			req: req,
			chunks: []StreamChunk{
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: "Hello"}}}},
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: " "}}}},
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: "world"}}}},
			},
			expectedEvents:  []domain.LLMStreamEventType{"meta", "delta", "delta", "delta", "done"},
			expectedContent: "Hello world",
		},
		"empty-delta": {
			req: req,
			chunks: []StreamChunk{
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: ""}}}},
			},
			expectedEvents:  []domain.LLMStreamEventType{"meta", "done"},
			expectedContent: "",
		},
		"no-usage-fallback": {
			req: req,
			chunks: []StreamChunk{
				{Choices: []StreamChunkChoice{{Delta: StreamChunkDelta{Content: "test"}}}},
			},
			expectedEvents:  []domain.LLMStreamEventType{"meta", "delta", "done"},
			expectedContent: "test",
		},
		"with-tool-calls": {
			req: domain.LLMChatRequest{
				Model: "test-model",
				Messages: []domain.LLMChatMessage{
					{
						Role: domain.ChatRole_Tool,
						ToolCalls: []domain.LLMStreamEventFunctionCall{
							{
								ID:        "toolcall-1",
								Function:  "list_todos",
								Arguments: `{"search_term":"books","page":1,"page_size":5}`,
							},
						},
					},
				},
				Tools: []domain.LLMTool{
					{
						Type: "search_web",
						Function: domain.LLMToolFunction{
							Name: "search_web",
							Parameters: domain.LLMToolFunctionParameters{
								Type: "object",
								Properties: map[string]domain.LLMToolFunctionParameterDetail{
									"search_term": {Type: "string", Description: "The search query", Required: true},
								},
							},
						},
					},
				},
			},
			chunks: []StreamChunk{
				{
					Choices: []StreamChunkChoice{
						{
							Delta: StreamChunkDelta{
								ToolCalls: []ToolCallChunk{
									{
										ID: "toolcall-1",
										Function: ToolCallChunkFunction{
											Name: "list_todos", Arguments: `{"search_term":"books","page":1,"page_size":5}`,
										},
									},
								},
							},
						},
					},
				},
			},

			expectedEvents:  []domain.LLMStreamEventType{"meta", "function_call", "done"},
			expectedContent: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server := createStreamingServer(tt.chunks)
			defer server.Close()

			client := NewDRMAPIClient(server.URL, "", server.Client())
			adapter := NewLLMClientAdapter(client)

			eventTypes, deltaTexts, _, err := collectStreamEvents(adapter, tt.req)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedEvents, eventTypes)

			combined := strings.Join(deltaTexts, "")
			assert.Equal(t, tt.expectedContent, combined)

		})
	}
}

func TestLLMClientAdapter_ChatStream_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewDRMAPIClient(server.URL, "", server.Client())
	adapter := NewLLMClientAdapter(client)

	req := domain.LLMChatRequest{
		Model: "test-model",
		Messages: []domain.LLMChatMessage{
			{Role: "user", Content: "test"},
		},
	}

	err := adapter.ChatStream(context.Background(), req, func(eventType domain.LLMStreamEventType, data interface{}) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestLLMClientAdapter_Chat(t *testing.T) {
	temp := 0.5
	topP := 0.9

	tests := map[string]struct {
		response     string
		statusCode   int
		req          domain.LLMChatRequest
		expectErr    bool
		expectedResp string
		validateReq  func(*testing.T, *ChatRequest)
	}{
		"success": {
			response:   `{"choices":[{"message":{"role":"assistant","content":"Hello!"}}]}`,
			statusCode: http.StatusOK,
			req: domain.LLMChatRequest{
				Model: "test-model",
				Messages: []domain.LLMChatMessage{
					{Role: "user", Content: "hi"},
				},
			},
			expectedResp: "Hello!",
		},
		"with-params": {
			response:   `{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`,
			statusCode: http.StatusOK,
			req: domain.LLMChatRequest{
				Model:       "test-model",
				Temperature: &temp,
				TopP:        &topP,
				Messages: []domain.LLMChatMessage{
					{Role: "system", Content: "sys"},
					{Role: "user", Content: "hi"},
				},
			},
			expectedResp: "ok",
			validateReq: func(t *testing.T, req *ChatRequest) {
				assert.Equal(t, "test-model", req.Model)
				assert.NotNil(t, req.Temperature)
				assert.InDelta(t, 0.5, *req.Temperature, 1e-6)
				assert.NotNil(t, req.TopP)
				assert.InDelta(t, 0.9, *req.TopP, 1e-6)
				assert.Len(t, req.Messages, 2)
			},
		},
		"no-choices": {
			response:   `{"choices":[]}`,
			statusCode: http.StatusOK,
			req: domain.LLMChatRequest{
				Model: "test-model",
				Messages: []domain.LLMChatMessage{
					{Role: "user", Content: "hi"},
				},
			},
			expectErr: true,
		},
		"server-error": {
			response:   `Internal Server Error`,
			statusCode: http.StatusInternalServerError,
			req: domain.LLMChatRequest{
				Model: "test-model",
				Messages: []domain.LLMChatMessage{
					{Role: "user", Content: "hi"},
				},
			},
			expectErr: true,
		},
		"invalid-json": {
			response:   `{invalid json}`,
			statusCode: http.StatusOK,
			req: domain.LLMChatRequest{
				Model: "test-model",
				Messages: []domain.LLMChatMessage{
					{Role: "user", Content: "hi"},
				},
			},
			expectErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var capturedReq *ChatRequest

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.validateReq != nil {
					var req ChatRequest
					json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
					capturedReq = &req
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response)) //nolint:errcheck
			}))
			defer server.Close()

			client := NewDRMAPIClient(server.URL, "", server.Client())
			adapter := NewLLMClientAdapter(client)

			resp, err := adapter.Chat(context.Background(), tt.req)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedResp, resp)

			if tt.validateReq != nil && capturedReq != nil {
				tt.validateReq(t, capturedReq)
			}
		})
	}
}

func TestLLMClientAdapter_Chat_ValidationErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`)) //nolint:errcheck
	}))
	defer server.Close()

	client := NewDRMAPIClient(server.URL, "", server.Client())
	adapter := NewLLMClientAdapter(client)

	tests := map[string]struct {
		req domain.LLMChatRequest
	}{
		"no-model":    {req: domain.LLMChatRequest{Messages: []domain.LLMChatMessage{{Role: "user", Content: "hi"}}}},
		"no-messages": {req: domain.LLMChatRequest{Model: "test"}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := adapter.Chat(context.Background(), tt.req)
			assert.Error(t, err)
		})
	}
}

func TestLLMClientAdapter_Embed(t *testing.T) {
	tests := map[string]struct {
		response    string
		statusCode  int
		model       string
		input       string
		expectErr   bool
		expectedVec []float64
	}{
		"success": {
			response: `{
                "model": "ai/qwen3-embedding",
                "object": "list",
                "usage": { "prompt_tokens": 6, "total_tokens": 6 },
                "data": [
                    {
                        "embedding": [1.1, 2.2, 3.3],
                        "index": 0,
                        "object": "embedding"
                    }
                ]
            }`,
			statusCode:  http.StatusOK,
			model:       "ai/qwen3-embedding",
			input:       "A dog is an animal",
			expectedVec: []float64{1.1, 2.2, 3.3},
		},
		"no-embedding-data": {
			response: `{
                "model": "ai/qwen3-embedding",
                "object": "list",
                "usage": { "prompt_tokens": 6, "total_tokens": 6 },
                "data": []
            }`,
			statusCode: http.StatusOK,
			model:      "ai/qwen3-embedding",
			input:      "A dog is an animal",
			expectErr:  true,
		},
		"server-error": {
			response:   `Internal Server Error`,
			statusCode: http.StatusInternalServerError,
			model:      "ai/qwen3-embedding",
			input:      "A dog is an animal",
			expectErr:  true,
		},
		"invalid-json": {
			response:   `{invalid json}`,
			statusCode: http.StatusOK,
			model:      "ai/qwen3-embedding",
			input:      "A dog is an animal",
			expectErr:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response)) //nolint:errcheck
			}))
			defer server.Close()

			client := NewDRMAPIClient(server.URL, "", server.Client())
			adapter := NewLLMClientAdapter(client)

			vec, err := adapter.Embed(context.Background(), tt.model, tt.input)

			if tt.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedVec, vec)
		})
	}
}

func TestInitLLMClient_Initialize(t *testing.T) {
	i := InitLLMClient{}

	_, err := i.Initialize(context.Background())
	assert.NoError(t, err)

	r, err := depend.Resolve[domain.LLMClient]()
	assert.NotNil(t, r)
	assert.NoError(t, err)
}

func TestCTT(t *testing.T) {
	// 100 sample todos with meaningful, searchable titles
	// var dt types.Date = types.Date{Time: time.Now()}
	// todos := []rest.CreateTodoRequest{
	// 	{Title: "Buy groceries", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 1)}},
	// 	{Title: "Finish project report", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 2)}},
	// 	{Title: "Call Alice about meeting", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 3)}},
	// 	{Title: "Schedule dentist appointment", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 4)}},
	// 	{Title: "Book flight tickets to NYC", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 5)}},
	// 	{Title: "Prepare for Monday presentation", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 6)}},
	// 	{Title: "Renew car insurance", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 7)}},
	// 	{Title: "Submit tax documents", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 8)}},
	// 	{Title: "Organize home office", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 9)}},
	// 	{Title: "Read 'Deep Work' book", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 10)}},
	// 	{Title: "Update LinkedIn profile", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 11)}},
	// 	{Title: "Plan weekend hiking trip", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 12)}},
	// 	{Title: "Clean out garage", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 13)}},
	// 	{Title: "Backup laptop files", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 14)}},
	// 	{Title: "Research new laptop models", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 15)}},
	// 	{Title: "Pay electricity bill", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 16)}},
	// 	{Title: "Arrange birthday party for Sam", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 17)}},
	// 	{Title: "Order new running shoes", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 18)}},
	// 	{Title: "Write blog post on productivity", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 19)}},
	// 	{Title: "Practice guitar for 30 minutes", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 20)}},
	// 	{Title: "Update project roadmap", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 21)}},
	// 	{Title: "Review quarterly budget", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 22)}},
	// 	{Title: "Send thank you notes", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 23)}},
	// 	{Title: "Fix leaky kitchen faucet", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 24)}},
	// 	{Title: "Prepare for coding interview", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 25)}},
	// 	{Title: "Organize photo album", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 26)}},
	// 	{Title: "Update resume", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 27)}},
	// 	{Title: "Research investment options", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 28)}},
	// 	{Title: "Plan family vacation", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 29)}},
	// 	{Title: "Buy birthday gift for Emma", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 30)}},
	// 	{Title: "Attend online marketing webinar", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 31)}},
	// 	{Title: "Set up home Wi-Fi network", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 32)}},
	// 	{Title: "Read 'Atomic Habits'", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 33)}},
	// 	{Title: "Organize bookshelf", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 34)}},
	// 	{Title: "Schedule annual health checkup", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 35)}},
	// 	{Title: "Clean windows", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 36)}},
	// 	{Title: "Write thank you email to mentor", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 37)}},
	// 	{Title: "Practice meditation", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 38)}},
	// 	{Title: "Update household budget", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 39)}},
	// 	{Title: "Buy groceries for the week", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 40)}},
	// 	{Title: "Fix bike tire", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 41)}},
	// 	{Title: "Plan team meeting agenda", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 42)}},
	// 	{Title: "Organize digital files", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 43)}},
	// 	{Title: "Read industry news", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 44)}},
	// 	{Title: "Prepare for book club", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 45)}},
	// 	{Title: "Update emergency contacts", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 46)}},
	// 	{Title: "Clean out email inbox", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 47)}},
	// 	{Title: "Research online courses", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 48)}},
	// 	{Title: "Buy new headphones", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 49)}},
	// 	{Title: "Schedule eye exam", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 50)}},
	// 	{Title: "Organize kitchen pantry", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 51)}},
	// 	{Title: "Write monthly newsletter", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 52)}},
	// 	{Title: "Plan charity event", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 53)}},
	// 	{Title: "Update website portfolio", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 54)}},
	// 	{Title: "Clean car interior", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 55)}},
	// 	{Title: "Buy pet food", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 56)}},
	// 	{Title: "Practice Spanish vocabulary", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 57)}},
	// 	{Title: "Organize closet", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 58)}},
	// 	{Title: "Read 'Clean Code'", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 59)}},
	// 	{Title: "Schedule parent-teacher meeting", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 60)}},
	// 	{Title: "Update insurance policy", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 61)}},
	// 	{Title: "Plan weekend getaway", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 62)}},
	// 	{Title: "Fix broken shelf", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 63)}},
	// 	{Title: "Write thank you card", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 64)}},
	// 	{Title: "Organize garden tools", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 65)}},
	// 	{Title: "Buy new backpack", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 66)}},
	// 	{Title: "Research healthy recipes", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 67)}},
	// 	{Title: "Clean bathroom", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 68)}},
	// 	{Title: "Update phone contacts", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 69)}},
	// 	{Title: "Plan study schedule", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 70)}},
	// 	{Title: "Buy new phone charger", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 71)}},
	// 	{Title: "Organize travel documents", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 72)}},
	// 	{Title: "Read 'The Pragmatic Programmer'", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 73)}},
	// 	{Title: "Clean living room", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 74)}},
	// 	{Title: "Update LinkedIn connections", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 75)}},
	// 	{Title: "Plan team lunch", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 76)}},
	// 	{Title: "Buy new water bottle", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 77)}},
	// 	{Title: "Organize receipts", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 78)}},
	// 	{Title: "Read 'Design Patterns'", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 79)}},
	// 	{Title: "Schedule car maintenance", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 80)}},
	// 	{Title: "Clean out fridge", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 81)}},
	// 	{Title: "Write cover letter", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 82)}},
	// 	{Title: "Plan birthday dinner", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 83)}},
	// 	{Title: "Buy new planner", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 84)}},
	// 	{Title: "Organize digital photos", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 85)}},
	// 	{Title: "Read 'Refactoring'", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 86)}},
	// 	{Title: "Schedule vet appointment", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 87)}},
	// 	{Title: "Clean patio", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 88)}},
	// 	{Title: "Update emergency kit", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 89)}},
	// 	{Title: "Buy new shoes", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 90)}},
	// 	{Title: "Organize music playlist", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 91)}},
	// 	{Title: "Read 'The Clean Coder'", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 92)}},
	// 	{Title: "Plan weekend picnic", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 93)}},
	// 	{Title: "Buy new lamp", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 94)}},
	// 	{Title: "Organize sports equipment", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 95)}},
	// 	{Title: "Read 'Effective Java'", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 96)}},
	// 	{Title: "Schedule dentist cleaning", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 97)}},
	// 	{Title: "Clean bedroom", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 98)}},
	// 	{Title: "Update family calendar", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 99)}},
	// 	{Title: "Buy new coffee maker", DueDate: types.Date{Time: time.Now().AddDate(0, 0, 100)}},
	// }

	// cli, _ := rest.NewClientWithResponses("http://localhost:8080")

	// for _, todo := range todos {
	// 	// Use the todo item as needed for seeding or testing
	// 	_, err := cli.CreateTodoWithResponse(t.Context(), todo)
	// 	require.NoError(t, err, "failed to create todo: %v", todo.Title)
	// }
}
