// Package llm provides a small, backend-agnostic client for a Docker-hosted
// OpenAI-compatible chat-completions endpoint (e.g. llama.cpp server).
//
// It intentionally ignores any non-standard fields such as "reasoning_content"
// and returns the assistant "content" (which may itself be JSON if you prompt
// the model to output JSON).
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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
func NewDockerModelAPIClient(baseURL string, apiKey string, httpClient *http.Client) (DockerModelAPIClient, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return DockerModelAPIClient{}, errors.New("aillm: baseURL is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return DockerModelAPIClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    httpClient,
	}, nil
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

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("llm: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
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
