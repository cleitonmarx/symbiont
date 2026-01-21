package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain"
	domain_mocks "github.com/cleitonmarx/symbiont/examples/todomailer/internal/domain/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetBoardSummaryImpl_Query(t *testing.T) {
	fixedUUID := func() uuid.UUID {
		return uuid.MustParse("223e4567-e89b-12d3-a456-426614174000")
	}
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	boardSummary := domain.BoardSummary{
		ID: fixedUUID(),
		Content: domain.BoardSummaryContent{
			Counts: domain.TodoStatusCounts{
				Open: 3,
				Done: 5,
			},
			NextUp: []domain.NextUpTodoItem{
				{
					Title:  "Review project proposal",
					Reason: "Due tomorrow",
				},
				{
					Title:  "Submit report",
					Reason: "Overdue by 2 days",
				},
			},
			Overdue: []string{
				"Submit report",
				"Update documentation",
			},
			NearDeadline: []string{
				"Review project proposal",
			},
			Summary: "You have 2 overdue tasks and 1 task due tomorrow.",
		},
		Model:         "mistral",
		GeneratedAt:   fixedTime,
		SourceVersion: 1,
	}

	tests := map[string]struct {
		setExpectations func(summaryRepo *domain_mocks.MockBoardSummaryRepository)
		expectedSummary domain.BoardSummary
		expectedErr     error
	}{
		"success": {
			setExpectations: func(summaryRepo *domain_mocks.MockBoardSummaryRepository) {
				summaryRepo.EXPECT().GetLatestSummary(
					mock.Anything,
				).Return(boardSummary, nil)
			},
			expectedSummary: boardSummary,
			expectedErr:     nil,
		},
		"repository-error": {
			setExpectations: func(summaryRepo *domain_mocks.MockBoardSummaryRepository) {
				summaryRepo.EXPECT().GetLatestSummary(
					mock.Anything,
				).Return(domain.BoardSummary{}, errors.New("database error"))
			},
			expectedSummary: domain.BoardSummary{},
			expectedErr:     errors.New("database error"),
		},
		"no-summary-found": {
			setExpectations: func(summaryRepo *domain_mocks.MockBoardSummaryRepository) {
				summaryRepo.EXPECT().GetLatestSummary(
					mock.Anything,
				).Return(domain.BoardSummary{}, nil)
			},
			expectedSummary: domain.BoardSummary{},
			expectedErr:     nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			summaryRepo := domain_mocks.NewMockBoardSummaryRepository(t)

			if tt.setExpectations != nil {
				tt.setExpectations(summaryRepo)
			}

			gbs := NewGetBoardSummaryImpl(summaryRepo)

			got, gotErr := gbs.Query(context.Background())
			assert.Equal(t, tt.expectedErr, gotErr)
			if tt.expectedErr == nil {
				assert.Equal(t, tt.expectedSummary.ID, got.ID)
				assert.Equal(t, tt.expectedSummary.Content.Counts.Open, got.Content.Counts.Open)
				assert.Equal(t, tt.expectedSummary.Content.Counts.Done, got.Content.Counts.Done)
				assert.Equal(t, tt.expectedSummary.Content.Summary, got.Content.Summary)
			}
		})
	}
}

func TestInitGetBoardSummary_Initialize(t *testing.T) {
	summaryRepo := domain_mocks.NewMockBoardSummaryRepository(t)

	igbs := &InitGetBoardSummary{
		SummaryRepo: summaryRepo,
	}

	ctx, err := igbs.Initialize(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)
}
