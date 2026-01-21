package aillm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/google/uuid"
)

// BoardSummaryGenerator implements domain.BoardSummaryGenerator using Docker Models API.
type BoardSummaryGenerator struct {
	client *DockerModelAPIClient
	model  string
}

// NewBoardSummaryGenerator creates a new BoardSummaryGenerator instance.
func NewBoardSummaryGenerator(client *DockerModelAPIClient, model string) *BoardSummaryGenerator {
	return &BoardSummaryGenerator{
		client: client,
		model:  model,
	}
}

// GenerateBoardSummary generates a board summary from todos using the LLM.
func (bsg *BoardSummaryGenerator) GenerateBoardSummary(ctx context.Context, todos []domain.Todo) (domain.BoardSummary, error) {
	prompt := buildPrompt(todos)

	req := ChatRequest{
		Model:  bsg.model,
		Stream: false,
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "You are an assistant embedded in a todo application. Use ONLY the provided data. Do NOT invent todos, dates, or statuses. Do NOT reveal reasoning or explanations. Output ONLY a valid JSON object matching the schema exactly. No extra keys. No markdown. No commentary.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	resp, err := bsg.client.Chat(ctx, req)
	if err != nil {
		return domain.BoardSummary{}, fmt.Errorf("failed to generate summary: %w", err)
	}

	if len(resp.Choices) == 0 {
		return domain.BoardSummary{}, fmt.Errorf("no response from LLM")
	}

	content := resp.Choices[0].Message.Content
	summary, err := parseResponse(content)
	if err != nil {
		return domain.BoardSummary{}, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	summary.ID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	summary.Model = bsg.model
	summary.GeneratedAt = time.Now()
	summary.SourceVersion = 1

	return summary, nil
}

// buildPrompt creates a detailed prompt for the LLM based on todos.
func buildPrompt(todos []domain.Todo) string {
	today := time.Now().Format("2006-01-02")

	todosJSON := buildTodosJSON(todos)

	prompt := fmt.Sprintf(`Generate a short, user-facing board summary.

Today (for calculations): %s

Definitions:
- overdue: due_date < Today AND status != DONE
- near_deadline: due_date within the next 7 days (inclusive) AND status != DONE
- priority order for completion: overdue first, then near_deadline, then remaining OPEN ordered by earliest due_date (null due_date goes last)

Requirements:
1) Count todos per status (include every status present).
2) Recommend up to 3 todos to complete next, in priority order.
3) List overdue and near-deadline todos (titles only).
4) Summary must be user-friendly, concise, and non-technical.

Todos:
%s

Return ONLY this JSON schema:
{
  "counts": { "OPEN": number, "DONE": number },
  "next_up": [ { "title": string, "reason": string } ],
  "overdue": [ string ],
  "near_deadline": [ string ],
  "summary": string
}

Rules:
- next_up: maximum 3 items, ordered by priority.
- overdue and near_deadline: titles only.
- summary: exactly 1 short sentence, friendly and encouraging.
- Do not include IDs or technical language.
- Output JSON only.`, today, todosJSON)

	return prompt
}

// buildTodosJSON creates the todos JSON for the prompt.
func buildTodosJSON(todos []domain.Todo) string {
	type TodoItem struct {
		Title   string  `json:"title"`
		Status  string  `json:"status"`
		DueDate *string `json:"due_date"`
	}

	type TodosWrapper struct {
		Items []TodoItem `json:"items"`
	}

	items := make([]TodoItem, 0, len(todos))
	for _, todo := range todos {
		var dueDate *string
		if !todo.DueDate.IsZero() {
			dueStr := todo.DueDate.Format("2006-01-02")
			dueDate = &dueStr
		}

		items = append(items, TodoItem{
			Title:   todo.Title,
			Status:  string(todo.Status),
			DueDate: dueDate,
		})
	}

	wrapper := TodosWrapper{Items: items}
	jsonBytes, _ := json.Marshal(wrapper)

	return string(jsonBytes)
}

// parseResponse extracts the BoardSummary from the LLM response.
func parseResponse(response string) (domain.BoardSummary, error) {
	// Extract JSON from response (in case there's extra text)
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}") + 1

	if jsonStart == -1 || jsonEnd <= jsonStart {
		return domain.BoardSummary{}, fmt.Errorf("no JSON found in response: %s", response)
	}

	jsonStr := response[jsonStart:jsonEnd]

	var content domain.BoardSummaryContent
	if err := json.Unmarshal([]byte(jsonStr), &content); err != nil {
		return domain.BoardSummary{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return domain.BoardSummary{
		Content: content,
	}, nil
}

// InitBoardSummaryGenerator is the initializer for BoardSummaryGenerator.
type InitBoardSummaryGenerator struct {
	Host  string `config:"DOCKER_MODEL_HOST"`
	Model string `config:"DOCKER_MODEL" default:"ai/mistral"`
}

// Initialize registers the BoardSummaryGenerator in the dependency container.
func (i *InitBoardSummaryGenerator) Initialize(ctx context.Context) (context.Context, error) {
	client, err := NewDockerModelAPIClient(i.Host, "", http.DefaultClient)
	if err != nil {
		return ctx, fmt.Errorf("failed to create DockerModelAPI client: %w", err)
	}

	generator := NewBoardSummaryGenerator(client, i.Model)
	depend.Register[domain.BoardSummaryGenerator](generator)

	return ctx, nil
}
