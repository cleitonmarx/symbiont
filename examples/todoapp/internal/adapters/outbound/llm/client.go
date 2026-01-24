// Package llm provides a small, backend-agnostic client for a Docker-hosted
// OpenAI-compatible chat-completions endpoint (e.g. llama.cpp server).
//
// It intentionally ignores any non-standard fields such as "reasoning_content"
// and returns the assistant "content" (which may itself be JSON if you prompt
// the model to output JSON).
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DockerModelAPIClient calls a Docker-hosted OpenAI-compatible API.
type DockerModelAPIClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewDockerModelAPIClient creates a new client.
// baseURL example: "http://localhost:12434" (no trailing slash required)
// apiKey is optional for local deployments.
func NewDockerModelAPIClient(baseURL string, apiKey string, httpClient *http.Client) DockerModelAPIClient {
	return DockerModelAPIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    httpClient,
	}
}

// ChatRequest is an OpenAI-compatible chat completions request.
type ChatRequest struct {
	Model       string          `json:"model"`
	Messages    []ChatMessage   `json:"messages"`
	Stream      bool            `json:"stream,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	MaxTokens   *int            `json:"max_tokens,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
	Format      string          `json:"format,omitempty"`          // optional; some servers accept it (e.g. "json")
	ResponseFmt *ResponseFormat `json:"response_format,omitempty"` // optional; some servers accept it
}

// ResponseFormat is an OpenAI-style response_format. Support varies by server.
type ResponseFormat struct {
	Type string `json:"type"` // e.g. "json_object"
}

// ChatMessage is an OpenAI-compatible message.
type ChatMessage struct {
	Role    string `json:"role"` // "system" | "user" | "assistant"
	Content string `json:"content"`
}

// ChatResponse is an OpenAI-compatible chat completions response.
// Note: some servers include extra fields; json decoding will ignore them.
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

// Choice is one completion choice.
type Choice struct {
	Index        int     `json:"index"`
	FinishReason string  `json:"finish_reason"`
	Message      Message `json:"message"`
}

// Message is the assistant message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`

	// Some servers (llama.cpp) include this field.
	// We ignore it but keep it to avoid losing data if you want to log it.
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

// Usage holds token usage metrics (if provided by the server).
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Chat sends a chat completion request and returns the parsed response.
func (c DockerModelAPIClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		return nil, errors.New("llm: request model is required")
	}
	if len(req.Messages) == 0 {
		return nil, errors.New("llm: request messages are required")
	}

	url, err := url.JoinPath(c.baseURL, "/v1/chat/completions")
	if err != nil {
		return nil, fmt.Errorf("llm: invalid base URL: %w", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("llm: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("llm: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("llm: http do: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("llm: read response body: %w", readErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Keep response text for debugging local servers.
		return nil, fmt.Errorf("llm: non-2xx response: %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var out ChatResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("llm: unmarshal response: %w; body=%s", err, strings.TrimSpace(string(respBody)))
	}

	return &out, nil
}

// StreamChunkDelta represents the delta content in a streaming response
type StreamChunkDelta struct {
	Content string `json:"content"`
}

// StreamChunkChoice represents a choice in a streaming response chunk
type StreamChunkChoice struct {
	Delta StreamChunkDelta `json:"delta"`
}

// StreamChunk represents a single chunk from the streaming API
type StreamChunk struct {
	Choices []StreamChunkChoice `json:"choices"`
	Usage   *Usage              `json:"usage,omitempty"`
	Timings *Timings            `json:"timings,omitempty"`
}

// Timings contains performance metrics from the API
type Timings struct {
	CacheN              int     `json:"cache_n"`
	PromptN             int     `json:"prompt_n"`
	PromptMS            float64 `json:"prompt_ms"`
	PromptPerTokenMS    float64 `json:"prompt_per_token_ms"`
	PromptPerSecond     float64 `json:"prompt_per_second"`
	PredictedN          int     `json:"predicted_n"`
	PredictedMS         float64 `json:"predicted_ms"`
	PredictedPerTokenMS float64 `json:"predicted_per_token_ms"`
	PredictedPerSecond  float64 `json:"predicted_per_second"`
}

// StreamEventMeta contains metadata for a streaming chat session
type StreamEventMeta struct {
	ConversationID     string    `json:"conversation_id"`
	UserMessageID      uuid.UUID `json:"user_message_id"`
	AssistantMessageID uuid.UUID `json:"assistant_message_id"`
	StartedAt          time.Time `json:"started_at"`
}

// StreamEventDelta contains a text delta from the stream
type StreamEventDelta struct {
	Text string `json:"text"`
}

// StreamEventDone contains completion metadata and token usage
type StreamEventDone struct {
	AssistantMessageID uuid.UUID `json:"assistant_message_id"`
	CompletedAt        string    `json:"completed_at"`
	Usage              Usage     `json:"usage"`
}

// StreamEventCallback is called for each event in the stream
type StreamEventCallback func(eventType string, data interface{}) error

// ChatStream streams assistant output as events from an OpenAI-compatible server.
// It calls onEvent with each event (meta, delta, done) and returns any error.
func (c DockerModelAPIClient) ChatStream(ctx context.Context, req ChatRequest, onEvent StreamEventCallback) error {
	if req.Model == "" {
		return errors.New("llm: request model is required")
	}
	if len(req.Messages) == 0 {
		return errors.New("llm: request messages are required")
	}
	url, err := url.JoinPath(c.baseURL, "/v1/chat/completions")
	if err != nil {
		return fmt.Errorf("llm: invalid base URL: %w", err)
	}

	req.Stream = true

	estimatedPromptTokens := estimateTokenCount(req.Messages)

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("llm: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("llm: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("llm: http do: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("llm: non-2xx response: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	meta := StreamEventMeta{
		ConversationID:     "global",
		UserMessageID:      uuid.New(),
		AssistantMessageID: uuid.New(),
		StartedAt:          time.Now().UTC(),
	}
	if err := onEvent("meta", meta); err != nil {
		return err
	}

	rd := bufio.NewReader(resp.Body)
	finalUsage, completedAt, err := c.consumeStream(rd, onEvent)
	if err != nil {
		return err
	}

	// Fallbacks and adjustments for usage/completedAt
	if finalUsage != nil && finalUsage.PromptTokens < estimatedPromptTokens {
		finalUsage.PromptTokens = estimatedPromptTokens
		finalUsage.TotalTokens = finalUsage.PromptTokens + finalUsage.CompletionTokens
	} else if finalUsage == nil {
		finalUsage = &Usage{
			PromptTokens:     estimatedPromptTokens,
			CompletionTokens: 0,
			TotalTokens:      estimatedPromptTokens,
		}
	}

	if completedAt == "" {
		completedAt = time.Now().UTC().Format(time.RFC3339)
	}

	done := StreamEventDone{
		AssistantMessageID: meta.AssistantMessageID,
		CompletedAt:        completedAt,
		Usage:              *finalUsage,
	}
	return onEvent("done", done)
}

// consumeStream reads the SSE stream, forwarding meta/delta/done events.
// It returns the final usage and completedAt captured from the stream (if any).
func (c DockerModelAPIClient) consumeStream(rd *bufio.Reader, onEvent StreamEventCallback) (*Usage, string, error) {
	var finalUsage *Usage
	var completedAt string
	currentEvent := ""

	for {
		line, readErr := rd.ReadString('\n')
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return nil, "", fmt.Errorf("llm: read stream: %w", readErr)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		if payload == "[DONE]" {
			break
		}

		handled, doneFromServer, usage, compAt, err := c.handleExplicitEvent(currentEvent, payload, onEvent)
		if err != nil {
			return nil, "", err
		}
		if usage != nil {
			finalUsage = usage
		}
		if compAt != "" {
			completedAt = compAt
		}
		if doneFromServer {
			break
		}
		if handled {
			continue
		}

		// OpenAI-style chunk parsing
		var chunk StreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			// Ignore non-JSON/keepalive lines
			continue
		}

		if chunk.Usage != nil {
			finalUsage = chunk.Usage
		}

		if chunk.Timings != nil && finalUsage == nil {
			finalUsage = &Usage{
				PromptTokens:     chunk.Timings.PromptN,
				CompletionTokens: chunk.Timings.PredictedN,
				TotalTokens:      chunk.Timings.PromptN + chunk.Timings.PredictedN,
			}
		}

		for _, ch := range chunk.Choices {
			if ch.Delta.Content != "" {
				if err := onEvent("delta", StreamEventDelta{Text: ch.Delta.Content}); err != nil {
					return nil, "", err
				}
			}
		}
	}

	return finalUsage, completedAt, nil
}

// handleExplicitEvent processes SSE frames that use explicit event types (e.g., llama.cpp: event: delta/done).
// It returns: handled(bool), doneFromServer(bool), usage(*Usage), completedAt(string), error.
func (c DockerModelAPIClient) handleExplicitEvent(
	currentEvent string,
	payload string,
	onEvent StreamEventCallback,
) (handled bool, doneFromServer bool, usage *Usage, completedAt string, err error) {

	switch currentEvent {
	case "delta":
		var d struct {
			Text string `json:"text"`
		}
		if json.Unmarshal([]byte(payload), &d) == nil {
			// forward delta even if text is empty (delta-empty-text test)
			if err := onEvent("delta", StreamEventDelta{Text: d.Text}); err != nil {
				return true, false, nil, "", err
			}
		}
		return true, false, nil, "", nil

	case "done":
		var dn struct {
			AssistantMessageID uuid.UUID `json:"assistant_message_id"`
			CompletedAt        string    `json:"completed_at"`
			Usage              *Usage    `json:"usage,omitempty"`
		}
		if json.Unmarshal([]byte(payload), &dn) == nil {
			if dn.Usage != nil {
				usage = dn.Usage
			}
			if dn.CompletedAt != "" {
				completedAt = dn.CompletedAt
			}
		}
		return true, true, usage, completedAt, nil
	}

	return false, false, nil, "", nil
}

// estimateTokenCount estimates the number of tokens in messages
// Uses a simple heuristic: ~1.3 tokens per word (common for English text)
func estimateTokenCount(messages []ChatMessage) int {
	totalChars := 0
	for _, msg := range messages {
		// Count role overhead (approximately 4 tokens per message for formatting)
		totalChars += 4
		// Count content
		totalChars += len(strings.Fields(msg.Content))
	}
	// Rough estimate: 1.3 tokens per word for English
	return int(float64(totalChars) * 1.3)
}
