package usecases

import (
	"context"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/tracing"
)

type GenerateBoardSummary interface {
	Execute(ctx context.Context) error
}

type GenerateBoardSummaryImpl struct {
	generator   domain.BoardSummaryGenerator
	summaryRepo domain.BoardSummaryRepository
	todoRepo    domain.TodoRepository
	// Add dependencies here if needed
}

func NewGenerateBoardSummaryImpl(g domain.BoardSummaryGenerator, r domain.BoardSummaryRepository, t domain.TodoRepository) GenerateBoardSummaryImpl {
	return GenerateBoardSummaryImpl{
		generator:   g,
		summaryRepo: r,
		todoRepo:    t,
	}
}

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

	return nil
}

type InitGenerateBoardSummary struct {
	Generator   domain.BoardSummaryGenerator  `resolve:""`
	SummaryRepo domain.BoardSummaryRepository `resolve:""`
	TodoRepo    domain.TodoRepository         `resolve:""`
}

func (igbs *InitGenerateBoardSummary) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[GenerateBoardSummary](NewGenerateBoardSummaryImpl(igbs.Generator, igbs.SummaryRepo, igbs.TodoRepo))
	return ctx, nil
}
