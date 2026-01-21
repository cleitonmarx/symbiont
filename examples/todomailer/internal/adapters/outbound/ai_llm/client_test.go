package aillm

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func NoTestExampleChatCall(t *testing.T) {
	client, err := NewDockerModelAPIClient("http://localhost:12434", "test", http.DefaultClient)
	assert.NoError(t, err)

	req := ChatRequest{
		Model: "ai/mistral",
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: "Hello, how are you?",
			},
		},
	}

	resp, err := client.Chat(context.Background(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Choices)
	t.Logf("LLM Response: %s", resp.Choices[0].Message.Content)
}

func NoTest(t *testing.T) {
	client, err := NewDockerModelAPIClient("http://localhost:12434", "test", http.DefaultClient)
	assert.NoError(t, err)
	gen := NewBoardSummaryGenerator(client, "ai/mistral")

	todos := generateRandomTodos()

	sum, err := gen.GenerateBoardSummary(context.Background(), todos)
	assert.NoError(t, err)
	t.Logf("Board Summary: %+v", sum)
}

func generateRandomTodos() []domain.Todo {
	now := time.Now()
	todos := []domain.Todo{
		{
			ID:        uuid.New(),
			Title:     "Pay electricity bill",
			Status:    domain.TodoStatus_OPEN,
			DueDate:   now.AddDate(0, 0, -6),
			CreatedAt: now.AddDate(0, 0, -10),
			UpdatedAt: now,
		},
		{
			ID:        uuid.New(),
			Title:     "Renew car insurance",
			Status:    domain.TodoStatus_OPEN,
			DueDate:   now.AddDate(0, 0, -3),
			CreatedAt: now.AddDate(0, 0, -15),
			UpdatedAt: now,
		},
		{
			ID:        uuid.New(),
			Title:     "Schedule annual medical checkup",
			Status:    domain.TodoStatus_OPEN,
			DueDate:   now.AddDate(0, 0, 1),
			CreatedAt: now.AddDate(0, 0, -5),
			UpdatedAt: now,
		},
		{
			ID:        uuid.New(),
			Title:     "Buy groceries for the week",
			Status:    domain.TodoStatus_OPEN,
			DueDate:   now.AddDate(0, 0, 2),
			CreatedAt: now.AddDate(0, 0, -2),
			UpdatedAt: now,
		},
		{
			ID:        uuid.New(),
			Title:     "Plan summer vacation",
			Status:    domain.TodoStatus_OPEN,
			DueDate:   now.AddDate(0, 2, 0),
			CreatedAt: now.AddDate(0, -1, 0),
			UpdatedAt: now,
		},
		{
			ID:        uuid.New(),
			Title:     "Clean garage and organize tools",
			Status:    domain.TodoStatus_OPEN,
			DueDate:   time.Time{},
			CreatedAt: now.AddDate(0, 0, -20),
			UpdatedAt: now,
		},
		{
			ID:        uuid.New(),
			Title:     "Submit quarterly taxes",
			Status:    domain.TodoStatus_DONE,
			DueDate:   now.AddDate(0, 1, 0),
			CreatedAt: now.AddDate(0, -2, 0),
			UpdatedAt: now.AddDate(0, 0, -1),
		},
		{
			ID:        uuid.New(),
			Title:     "Book dentist appointment",
			Status:    domain.TodoStatus_DONE,
			DueDate:   now.AddDate(0, 0, 20),
			CreatedAt: now.AddDate(0, 0, -10),
			UpdatedAt: now.AddDate(0, 0, -2),
		},
		{
			ID:        uuid.New(),
			Title:     "Review project documentation",
			Status:    domain.TodoStatus_OPEN,
			DueDate:   now.AddDate(0, 0, 3),
			CreatedAt: now.AddDate(0, 0, -8),
			UpdatedAt: now,
		},
		{
			ID:        uuid.New(),
			Title:     "Update personal website",
			Status:    domain.TodoStatus_OPEN,
			DueDate:   now.AddDate(0, 0, 10),
			CreatedAt: now.AddDate(0, 0, -25),
			UpdatedAt: now,
		},
	}

	return todos
}
