package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// LLMStreamEventType represents the type of event in an LLM stream
type LLMStreamEventType string

const (
	LLMStreamEventType_Meta  LLMStreamEventType = "meta"
	LLMStreamEventType_Delta LLMStreamEventType = "delta"
	LLMStreamEventType_Done  LLMStreamEventType = "done"
)

// LLMChatMessage represents a message in a chat request to the LLM API
type LLMChatMessage struct {
	Role    ChatRole `json:"role" yaml:"role"`
	Content string   `json:"content" yaml:"content"`
}

// LLMChatRequest represents a request to the LLM API
type LLMChatRequest struct {
	Model    string           `json:"model"`
	Messages []LLMChatMessage `json:"messages"`
	Stream   bool             `json:"stream"`
	// Optional parameters
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
}

// LLMUsage represents token usage information from the LLM
type LLMUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMStreamEventMeta contains metadata for a streaming chat session
type LLMStreamEventMeta struct {
	ConversationID     string    `json:"conversation_id"`
	UserMessageID      uuid.UUID `json:"user_message_id"`
	AssistantMessageID uuid.UUID `json:"assistant_message_id"`
	StartedAt          time.Time `json:"started_at"`
}

// LLMStreamEventDelta contains a text delta from the stream
type LLMStreamEventDelta struct {
	Text string `json:"text"`
}

// LLMStreamEventDone contains completion metadata and token usage
type LLMStreamEventDone struct {
	AssistantMessageID string    `json:"assistant_message_id"`
	CompletedAt        string    `json:"completed_at"`
	Usage              *LLMUsage `json:"usage,omitempty"`
}

// LLMStreamEventCallback is called for each event in the stream
type LLMStreamEventCallback func(eventType LLMStreamEventType, data any) error

// LLMClient defines the interface for interacting with an LLM API
type LLMClient interface {
	// ChatStream streams assistant output as events from an LLM server
	// It calls onEvent with each event (meta, delta, done) and returns any error
	ChatStream(ctx context.Context, req LLMChatRequest, onEvent LLMStreamEventCallback) error

	// Chat sends a chat request to the LLM and returns the full assistant response
	Chat(ctx context.Context, req LLMChatRequest) (string, error)
}
