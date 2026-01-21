package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	domain_mocks "github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGenerateBoardSummaryImpl_Execute(t *testing.T) {
	fixedUUID := func() uuid.UUID {
		return uuid.MustParse("223e4567-e89b-12d3-a456-426614174000")
	}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	todos := []domain.Todo{
		{
			ID:        fixedUUID(),
			Title:     "Open task 1",
			Status:    domain.TodoStatus_OPEN,
			DueDate:   fixedTime.AddDate(0, 0, 5),
			CreatedAt: fixedTime,
			UpdatedAt: fixedTime,
		},
		{
			ID:        uuid.MustParse("323e4567-e89b-12d3-a456-426614174000"),
			Title:     "Done task 1",
			Status:    domain.TodoStatus_DONE,
			DueDate:   fixedTime.AddDate(0, 0, -1),
			CreatedAt: fixedTime,
			UpdatedAt: fixedTime,
		},
	}

	boardSummary := domain.BoardSummary{
		ID: fixedUUID(),
		Content: domain.BoardSummaryContent{
			Counts: domain.TodoStatusCounts{
				Open: 1,
				Done: 1,
			},
			NextUp: []domain.NextUpTodoItem{
				{
					Title:  "Open task 1",
					Reason: "Due in 5 days",
				},
			},
			Overdue:      []string{},
			NearDeadline: []string{"Done task 1"},
			Summary:      "You have 1 open todo and 1 completed todo.",
		},
		Model:         "mistral",
		GeneratedAt:   fixedTime,
		SourceVersion: 1,
	}

	tests := map[string]struct {
		setExpectations func(generator *domain_mocks.MockBoardSummaryGenerator, summaryRepo *domain_mocks.MockBoardSummaryRepository, todoRepo *domain_mocks.MockTodoRepository)
		expectedErr     error
	}{
		"success": {
			setExpectations: func(generator *domain_mocks.MockBoardSummaryGenerator, summaryRepo *domain_mocks.MockBoardSummaryRepository, todoRepo *domain_mocks.MockTodoRepository) {
				todoRepo.EXPECT().ListTodos(
					mock.Anything,
					1,
					1000,
				).Return(todos, false, nil)

				generator.EXPECT().GenerateBoardSummary(
					mock.Anything,
					todos,
				).Return(boardSummary, nil)

				summaryRepo.EXPECT().StoreSummary(
					mock.Anything,
					mock.MatchedBy(func(s domain.BoardSummary) bool {
						return s.Model == boardSummary.Model &&
							s.Content.Counts.Open == boardSummary.Content.Counts.Open &&
							s.Content.Counts.Done == boardSummary.Content.Counts.Done
					}),
				).Return(nil)
			},
			expectedErr: nil,
		},
		"list-todos-error": {
			setExpectations: func(generator *domain_mocks.MockBoardSummaryGenerator, summaryRepo *domain_mocks.MockBoardSummaryRepository, todoRepo *domain_mocks.MockTodoRepository) {
				todoRepo.EXPECT().ListTodos(
					mock.Anything,
					1,
					1000,
				).Return(nil, false, errors.New("database error"))
			},
			expectedErr: errors.New("database error"),
		},
		"generator-error": {
			setExpectations: func(generator *domain_mocks.MockBoardSummaryGenerator, summaryRepo *domain_mocks.MockBoardSummaryRepository, todoRepo *domain_mocks.MockTodoRepository) {
				todoRepo.EXPECT().ListTodos(
					mock.Anything,
					1,
					1000,
				).Return(todos, false, nil)

				generator.EXPECT().GenerateBoardSummary(
					mock.Anything,
					todos,
				).Return(domain.BoardSummary{}, errors.New("LLM error"))
			},
			expectedErr: errors.New("LLM error"),
		},
		"store-summary-error": {
			setExpectations: func(generator *domain_mocks.MockBoardSummaryGenerator, summaryRepo *domain_mocks.MockBoardSummaryRepository, todoRepo *domain_mocks.MockTodoRepository) {
				todoRepo.EXPECT().ListTodos(
					mock.Anything,
					1,
					1000,
				).Return(todos, false, nil)

				generator.EXPECT().GenerateBoardSummary(
					mock.Anything,
					todos,
				).Return(boardSummary, nil)

				summaryRepo.EXPECT().StoreSummary(
					mock.Anything,
					mock.Anything,
				).Return(errors.New("database error"))
			},
			expectedErr: errors.New("database error"),
		},
		"empty-todos": {
			setExpectations: func(generator *domain_mocks.MockBoardSummaryGenerator, summaryRepo *domain_mocks.MockBoardSummaryRepository, todoRepo *domain_mocks.MockTodoRepository) {
				todoRepo.EXPECT().ListTodos(
					mock.Anything,
					1,
					1000,
				).Return([]domain.Todo{}, false, nil)

				emptySummary := domain.BoardSummary{
					Content: domain.BoardSummaryContent{
						Counts: domain.TodoStatusCounts{
							Open: 0,
							Done: 0,
						},
						Summary: "No todos to summarize.",
					},
				}

				generator.EXPECT().GenerateBoardSummary(
					mock.Anything,
					[]domain.Todo{},
				).Return(emptySummary, nil)

				summaryRepo.EXPECT().StoreSummary(
					mock.Anything,
					mock.Anything,
				).Return(nil)
			},
			expectedErr: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			generator := domain_mocks.NewMockBoardSummaryGenerator(t)
			summaryRepo := domain_mocks.NewMockBoardSummaryRepository(t)
			todoRepo := domain_mocks.NewMockTodoRepository(t)

			if tt.setExpectations != nil {
				tt.setExpectations(generator, summaryRepo, todoRepo)
			}

			gbs := NewGenerateBoardSummaryImpl(generator, summaryRepo, todoRepo)

			err := gbs.Execute(context.Background())
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestInitGenerateBoardSummary_Initialize(t *testing.T) {
	igbs := InitGenerateBoardSummary{}

	ctx, err := igbs.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredGbs, err := depend.Resolve[GenerateBoardSummary]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredGbs)
}
