package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/google/uuid"
)

// BoardSummaryGenerator implements domain.BoardSummaryGenerator using Docker Models API.
type BoardSummaryGenerator struct {
	timeProvider domain.CurrentTimeProvider
	client       DockerModelAPIClient
	model        string
}

// NewBoardSummaryGenerator creates a new BoardSummaryGenerator instance.
func NewBoardSummaryGenerator(timeProvider domain.CurrentTimeProvider, client DockerModelAPIClient, model string) BoardSummaryGenerator {
	return BoardSummaryGenerator{
		timeProvider: timeProvider,
		client:       client,
		model:        model,
	}
}

// GenerateBoardSummary generates a board summary from todos using the LLM.
func (bsg BoardSummaryGenerator) GenerateBoardSummary(ctx context.Context, todos []domain.Todo) (domain.BoardSummary, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	now := bsg.timeProvider.Now()
	prompt := buildPrompt(todos, now.Format("2006-01-02"))

	req := ChatRequest{
		Model:       bsg.model,
		Stream:      false,
		Temperature: common.Ptr[float64](0),
		TopP:        common.Ptr(0.1),
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "You are a JSON-only processor. Output ONLY raw JSON object. NO markdown code blocks, NO triple backticks, NO text before or after. Output starts with { and ends with }.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	resp, err := bsg.client.Chat(spanCtx, req)
	if tracing.RecordErrorAndStatus(span, err) {
		return domain.BoardSummary{}, fmt.Errorf("failed to generate summary: %w", err)
	}

	if len(resp.Choices) == 0 {
		err := fmt.Errorf("llm: no choices in response")
		tracing.RecordErrorAndStatus(span, err)
		return domain.BoardSummary{}, err
	}

	content := resp.Choices[0].Message.Content
	summary, err := parseResponse(content)
	if tracing.RecordErrorAndStatus(span, err) {
		return domain.BoardSummary{}, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	summary.ID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	summary.Model = bsg.model
	summary.GeneratedAt = now
	summary.SourceVersion = 1

	return summary, nil
}

// buildPrompt creates a detailed prompt for the LLM based on todos.
func buildPrompt(todos []domain.Todo, today string) string {
	todosJSON := buildTodosJSON(todos)

	prompt := fmt.Sprintf(`You are categorizing todos by DATE ONLY using strict mathematical comparison.

TODAY'S DATE: %s

DATE MATH RULES (use simple string comparison YYYY-MM-DD):
- overdue: due_date < %s AND status = OPEN
- near_deadline: due_date >= %s AND due_date <= %s AND status = OPEN
- future: due_date > %s AND status = OPEN
- done: status = DONE (NEVER include in overdue, near_deadline, future, or next_up)

IGNORE ALL TITLE KEYWORDS: "urgent", "important", "ASAP", "critical", "priority" are IRRELEVANT.

TODAY = %s
TODAY+7 = %s

STRICT EXAMPLE (TODAY = %s):
- Due 2026-01-20 < 2026-01-21? YES → overdue ✓
- Due 2026-01-21 < 2026-01-21? NO → NOT overdue
- Due 2026-01-21 >= 2026-01-21? YES → near_deadline ✓
- Due 2026-01-21 <= 2026-01-28? YES → near_deadline ✓
- Due 2026-01-23 >= 2026-01-21? YES AND Due 2026-01-23 <= 2026-01-28? YES → near_deadline ✓
- Due 2026-01-29 > 2026-01-28? YES → future ✓

OUTPUT REQUIREMENTS:
Return ONLY valid JSON (no markdown, no code blocks, no extra text):
{
  "counts": { "OPEN": number, "DONE": number },
  "next_up": [ { "title": string, "reason": string } ],
  "overdue": [ string ],
  "near_deadline": [ string ],
  "summary": string
}

OUTPUT RULES:
- "counts": Count OPEN and DONE statuses
- "next_up": ONLY OPEN todos. Order: [overdue (oldest first), near_deadline (earliest first), future (earliest first)]. Max 10. Reason: "overdue", "due within 7 days", or "upcoming"
- "overdue": ONLY OPEN todos with due_date < %s. Sort by most overdue first
- "near_deadline": ONLY OPEN todos with %s <= due_date <= %s. Sort by due_date ascending
- "summary": 1 friendly sentence
- Output ONLY the JSON object.

Todos JSON INPUT:
%s`,
		today,
		today,
		today,
		addDays(today, 7),
		addDays(today, 7),
		today,
		addDays(today, 7),
		today,
		today,
		today,
		addDays(today, 7),
		todosJSON)

	return prompt
}

// addDays adds days to a date string in YYYY-MM-DD format
func addDays(dateStr string, days int) string {
	t, _ := time.Parse("2006-01-02", dateStr)
	return t.AddDate(0, 0, days).Format("2006-01-02")
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
	HttpClient   *http.Client               `resolve:""`
	TimeProvider domain.CurrentTimeProvider `resolve:""`
	LLMHost      string                     `config:"LLM_MODEL_HOST"`
	LLMModel     string                     `config:"LLM_MODEL" default:"ai/gpt-oss"`
	LLMAPIKey    string                     `config:"LLM_MODEL_API_KEY" default:"none"`
}

// Initialize registers the BoardSummaryGenerator in the dependency container.
func (i InitBoardSummaryGenerator) Initialize(ctx context.Context) (context.Context, error) {
	client, err := NewDockerModelAPIClient(i.LLMHost, i.LLMAPIKey, i.HttpClient)
	if err != nil {
		return ctx, fmt.Errorf("failed to create DockerModelAPI client: %w", err)
	}

	generator := NewBoardSummaryGenerator(i.TimeProvider, client, i.LLMModel)
	depend.Register[domain.BoardSummaryGenerator](generator)

	return ctx, nil
}
