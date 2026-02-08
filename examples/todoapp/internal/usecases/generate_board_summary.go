package usecases

import (
	"context"
	"embed"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
	"github.com/google/uuid"
	"github.com/toon-format/toon-go"
	"go.yaml.in/yaml/v3"
)

// CompletedSummaryQueue is a channel type for sending processed domain.BoardSummary items.
// It is used in integration tests to verify summary generation.
type CompletedSummaryQueue chan domain.BoardSummary

// GenerateBoardSummary is the use case interface for generating a summary of the todo board.
type GenerateBoardSummary interface {
	Execute(ctx context.Context) error
}

// GenerateBoardSummaryImpl is the implementation of the GenerateBoardSummary use case.
type GenerateBoardSummaryImpl struct {
	summaryRepo  domain.BoardSummaryRepository
	timeProvider domain.CurrentTimeProvider
	llmClient    domain.LLMClient
	model        string
	queue        CompletedSummaryQueue
	// Add dependencies here if needed
}

// NewGenerateBoardSummaryImpl creates a new instance of GenerateBoardSummaryImpl.
func NewGenerateBoardSummaryImpl(
	bsr domain.BoardSummaryRepository,
	tp domain.CurrentTimeProvider,
	c domain.LLMClient,
	m string,
	q CompletedSummaryQueue,

) GenerateBoardSummaryImpl {
	return GenerateBoardSummaryImpl{
		summaryRepo:  bsr,
		timeProvider: tp,
		llmClient:    c,
		model:        m,
		queue:        q,
	}
}

// Execute runs the use case to generate the board summary.
func (gs GenerateBoardSummaryImpl) Execute(ctx context.Context) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	summary, err := gs.generateBoardSummary(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	err = gs.summaryRepo.StoreSummary(spanCtx, summary)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	if gs.queue != nil {
		gs.queue <- summary
	}

	return nil
}

func (gs GenerateBoardSummaryImpl) generateBoardSummary(ctx context.Context) (domain.BoardSummary, error) {

	new, err := gs.summaryRepo.CalculateSummaryContent(ctx)
	if err != nil {
		return domain.BoardSummary{}, fmt.Errorf("failed to calculate summary content: %w", err)
	}

	previous, found, err := gs.summaryRepo.GetLatestSummary(ctx)
	if err != nil {
		return domain.BoardSummary{}, fmt.Errorf("failed to get latest summary: %w", err)
	}
	if !found {
		previous.Content.Summary = "no previous summary"
	}

	now := gs.timeProvider.Now()
	promptMessages, err := buildPromptMessages(new, previous.Content)
	if err != nil {
		return domain.BoardSummary{}, fmt.Errorf("failed to build prompt: %w", err)
	}

	req := domain.LLMChatRequest{
		Model:       gs.model,
		Stream:      false,
		Temperature: common.Ptr(1.2),
		TopP:        common.Ptr(0.95),
		Messages:    promptMessages,
	}

	resp, err := gs.llmClient.Chat(ctx, req)
	if err != nil {
		return domain.BoardSummary{}, err
	}

	new.Summary = strings.TrimSpace(resp.Content)
	new.Summary = applySummarySafetyGuards(new.Summary, new)

	RecordLLMTokensUsed(ctx, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	summary := domain.BoardSummary{
		ID:            uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Content:       new,
		Model:         gs.model,
		GeneratedAt:   now,
		SourceVersion: 1,
	}

	return summary, nil
}

//go:embed prompts/summary.yml
var summaryPrompt embed.FS

// buildPromptMessages constructs the LLM messages for the summary prompt.
func buildPromptMessages(new domain.BoardSummaryContent, previous domain.BoardSummaryContent) ([]domain.LLMChatMessage, error) {
	inputTOON, err := marshalSummaryContent(new)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal summary content: %w", err)
	}

	previousTOON, err := marshalSummaryContent(previous)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal previous summary content: %w", err)
	}

	completedCandidates, doneDelta := completedProgressHints(new, previous)
	completedCandidatesText := "none"
	if len(completedCandidates) > 0 {
		completedCandidatesText = strings.Join(completedCandidates, "; ")
	}
	overdueTitlesText, nearDeadlineTitlesText, nextUpOverdueText, nextUpDueSoonText, nextUpUpcomingText, nextUpFutureText := urgencyHints(new)

	file, err := summaryPrompt.Open("prompts/summary.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to open summary prompt: %w", err)
	}
	defer file.Close() //nolint:errcheck

	messages := []domain.LLMChatMessage{}
	err = yaml.NewDecoder(file).Decode(&messages)
	if err != nil {
		return nil, fmt.Errorf("failed to decode summary prompt: %w", err)
	}

	for i, msg := range messages {
		msg.Content = fmt.Sprintf(
			msg.Content,
			inputTOON,
			previousTOON,
			completedCandidatesText,
			doneDelta,
			overdueTitlesText,
			nearDeadlineTitlesText,
			nextUpOverdueText,
			nextUpDueSoonText,
			nextUpUpcomingText,
			nextUpFutureText,
		)
		messages[i] = msg
	}

	return messages, nil
}

// urgencyHints processes the BoardSummaryContent to extract and format titles of tasks based on their urgency categories for LLM hints.
func urgencyHints(content domain.BoardSummaryContent) (string, string, string, string, string, string) {
	overdueTitles := normalizeTitles(content.Overdue)
	nearDeadlineTitles := normalizeTitles(content.NearDeadline)

	nextUpOverdue := []string{}
	nextUpDueSoon := []string{}
	nextUpUpcoming := []string{}
	nextUpFuture := []string{}

	for _, item := range content.NextUp {
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(item.Reason)) {
		case "overdue":
			nextUpOverdue = append(nextUpOverdue, title)
		case "due within 7 days":
			nextUpDueSoon = append(nextUpDueSoon, title)
		case "upcoming":
			nextUpUpcoming = append(nextUpUpcoming, title)
		case "future":
			nextUpFuture = append(nextUpFuture, title)
		}
	}

	return formatTitleHints(overdueTitles),
		formatTitleHints(nearDeadlineTitles),
		formatTitleHints(normalizeTitles(nextUpOverdue)),
		formatTitleHints(normalizeTitles(nextUpDueSoon)),
		formatTitleHints(normalizeTitles(nextUpUpcoming)),
		formatTitleHints(normalizeTitles(nextUpFuture))
}

// normalizeTitles trims whitespace, removes empty titles, deduplicates, and sorts the list of titles.
func normalizeTitles(titles []string) []string {
	uniq := make(map[string]struct{}, len(titles))
	for _, title := range titles {
		trimmed := strings.TrimSpace(title)
		if trimmed == "" {
			continue
		}
		uniq[trimmed] = struct{}{}
	}

	items := make([]string, 0, len(uniq))
	for title := range uniq {
		items = append(items, title)
	}
	sort.Strings(items)

	return items
}

// formatTitleHints formats a list of titles into a single string for LLM hints, or returns "none" if the list is empty.
func formatTitleHints(titles []string) string {
	if len(titles) == 0 {
		return "none"
	}
	return strings.Join(titles, "; ")
}

var (
	reNoOverdueTasks    = regexp.MustCompile(`(?i)\bno overdue tasks?\b`)
	reNoTasksAreOverdue = regexp.MustCompile(`(?i)\bno tasks are overdue\b`)
	reNothingIsOverdue  = regexp.MustCompile(`(?i)\bnothing is overdue\b`)
	reOverdueQualifier  = regexp.MustCompile(`(?i)\boverdue\s+`)
	reLateQualifier     = regexp.MustCompile(`(?i)\blate\s+`)
	rePastDueQualifier  = regexp.MustCompile(`(?i)\bpast[- ]due\s+`)
	reExtraSpaces       = regexp.MustCompile(`\s{2,}`)
	reSpaceBeforePunct  = regexp.MustCompile(`\s+([,.;:!?])`)
)

// applySummarySafetyGuards cleans the generated summary text to prevent certain phrases
// from appearing if they are not supported by the current board facts.
func applySummarySafetyGuards(summary string, content domain.BoardSummaryContent) string {
	cleaned := strings.TrimSpace(summary)
	if cleaned == "" {
		return cleaned
	}

	// Guardrail for weaker models: if there are no overdue tasks in current facts,
	// do not allow overdue/late phrasing to leak into the final summary text.
	if len(content.Overdue) == 0 {
		cleaned = reNoOverdueTasks.ReplaceAllString(cleaned, "__NO_OVERDUE_TASKS__")
		cleaned = reNoTasksAreOverdue.ReplaceAllString(cleaned, "__NO_TASKS_ARE_OVERDUE__")
		cleaned = reNothingIsOverdue.ReplaceAllString(cleaned, "__NOTHING_IS_OVERDUE__")

		cleaned = reOverdueQualifier.ReplaceAllString(cleaned, "")
		cleaned = reLateQualifier.ReplaceAllString(cleaned, "")
		cleaned = rePastDueQualifier.ReplaceAllString(cleaned, "")
		cleaned = reExtraSpaces.ReplaceAllString(cleaned, " ")
		cleaned = reSpaceBeforePunct.ReplaceAllString(cleaned, "$1")
		cleaned = strings.TrimSpace(cleaned)

		cleaned = strings.ReplaceAll(cleaned, "__NO_OVERDUE_TASKS__", "no overdue tasks")
		cleaned = strings.ReplaceAll(cleaned, "__NO_TASKS_ARE_OVERDUE__", "no tasks are overdue")
		cleaned = strings.ReplaceAll(cleaned, "__NOTHING_IS_OVERDUE__", "nothing is overdue")
	}

	return cleaned
}

// completedProgressHints compares the current and previous board summary content to identify
// recently completed items and returns hints for the LLM.
func completedProgressHints(current, previous domain.BoardSummaryContent) ([]string, int) {
	doneDelta := current.Counts.Done - previous.Counts.Done

	currentTitles := make(map[string]struct{})
	addSummaryTitles(currentTitles, current)

	previousTitles := make(map[string]struct{})
	addSummaryTitles(previousTitles, previous)

	candidates := make([]string, 0, len(previousTitles))
	for title := range previousTitles {
		if _, exists := currentTitles[title]; exists {
			continue
		}
		candidates = append(candidates, title)
	}
	sort.Strings(candidates)

	return candidates, doneDelta
}

// addSummaryTitles adds the titles of summary items to the provided map for easy lookup.
func addSummaryTitles(dst map[string]struct{}, content domain.BoardSummaryContent) {
	for _, item := range content.NextUp {
		title := strings.TrimSpace(item.Title)
		if title == "" {
			continue
		}
		dst[title] = struct{}{}
	}
	for _, title := range content.Overdue {
		trimmed := strings.TrimSpace(title)
		if trimmed == "" {
			continue
		}
		dst[trimmed] = struct{}{}
	}
	for _, title := range content.NearDeadline {
		trimmed := strings.TrimSpace(title)
		if trimmed == "" {
			continue
		}
		dst[trimmed] = struct{}{}
	}
}

// marshalSummaryContent converts the BoardSummaryContent struct into a TOON string for LLM input.
func marshalSummaryContent(sc domain.BoardSummaryContent) (string, error) {
	summaryContentTOON, err := toon.MarshalString(sc, toon.WithLengthMarkers(true))
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary content: %w", err)
	}

	return summaryContentTOON, nil
}

// InitGenerateBoardSummary initializes the GenerateBoardSummary use case.
type InitGenerateBoardSummary struct {
	SummaryRepo  domain.BoardSummaryRepository `resolve:""`
	TimeProvider domain.CurrentTimeProvider    `resolve:""`
	LLMClient    domain.LLMClient              `resolve:""`
	Model        string                        `config:"LLM_SUMMARY_MODEL"`
}

// Initialize registers the GenerateBoardSummary use case implementation.
func (igbs InitGenerateBoardSummary) Initialize(ctx context.Context) (context.Context, error) {
	queue, _ := depend.Resolve[CompletedSummaryQueue]()
	depend.Register[GenerateBoardSummary](NewGenerateBoardSummaryImpl(
		igbs.SummaryRepo, igbs.TimeProvider, igbs.LLMClient, igbs.Model, queue,
	))
	return ctx, nil
}
