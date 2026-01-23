package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
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
	generator   domain.BoardSummaryGenerator
	summaryRepo domain.BoardSummaryRepository
	todoRepo    domain.TodoRepository
	queue       CompletedSummaryQueue
	// Add dependencies here if needed
}

// NewGenerateBoardSummaryImpl creates a new instance of GenerateBoardSummaryImpl.
func NewGenerateBoardSummaryImpl(g domain.BoardSummaryGenerator, r domain.BoardSummaryRepository, t domain.TodoRepository, q CompletedSummaryQueue) GenerateBoardSummaryImpl {
	return GenerateBoardSummaryImpl{
		generator:   g,
		summaryRepo: r,
		todoRepo:    t,
		queue:       q,
	}
}

// Execute runs the use case to generate the board summary.
func (gs GenerateBoardSummaryImpl) Execute(ctx context.Context) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	todos, _, err := gs.todoRepo.ListTodos(
		spanCtx,
		1,
		1000,
	)
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	summary, err := gs.generator.GenerateBoardSummary(spanCtx, todos)
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	err = gs.summaryRepo.StoreSummary(spanCtx, summary)
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	if gs.queue != nil {
		gs.queue <- summary
	}

	return nil
}

// InitGenerateBoardSummary initializes the GenerateBoardSummary use case.
type InitGenerateBoardSummary struct {
	Generator   domain.BoardSummaryGenerator  `resolve:""`
	SummaryRepo domain.BoardSummaryRepository `resolve:""`
	TodoRepo    domain.TodoRepository         `resolve:""`
}

// Initialize registers the GenerateBoardSummary use case implementation.
func (igbs InitGenerateBoardSummary) Initialize(ctx context.Context) (context.Context, error) {
	queue, _ := depend.Resolve[CompletedSummaryQueue]()
	depend.Register[GenerateBoardSummary](NewGenerateBoardSummaryImpl(igbs.Generator, igbs.SummaryRepo, igbs.TodoRepo, queue))
	return ctx, nil
}
