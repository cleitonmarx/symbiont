package usecases

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/google/uuid"
)

// LLMTool represents a tool that can be used in chat interactions.
type LLMTool interface {
	Tool() domain.LLMTool
	Call(ctx context.Context, call domain.LLMStreamEventFunctionCall) domain.LLMChatMessage
}

// LLMToolRegistry defines the interface for calling LLM tools.
type LLMToolRegistry interface {
	Call(ctx context.Context, call domain.LLMStreamEventFunctionCall) domain.LLMChatMessage
	// List returns all registered LLM tools.
	List() []domain.LLMTool
}

// LLMToolManager manages a collection of LLM tools.
type LLMToolManager struct {
	tools map[string]LLMTool
}

// NewLLMToolManager creates a new LLMToolManager with the provided tools.
func NewLLMToolManager(tools ...LLMTool) LLMToolManager {
	toolMap := make(map[string]LLMTool)
	for _, tool := range tools {
		toolMap[tool.Tool().Function.Name] = tool
	}
	return LLMToolManager{
		tools: toolMap,
	}
}

// List returns all registered LLM tools.
func (ctr LLMToolManager) List() []domain.LLMTool {
	toolList := make([]domain.LLMTool, 0, len(ctr.tools))
	for _, tool := range ctr.tools {
		toolList = append(toolList, tool.Tool())
	}
	return toolList
}

// Call invokes the appropriate tool based on the function call.
func (ctr LLMToolManager) Call(ctx context.Context, call domain.LLMStreamEventFunctionCall) domain.LLMChatMessage {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()
	tool, exists := ctr.tools[call.Function]
	if !exists {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: "Error: unknown tool " + call.Function,
		}
	}
	return tool.Call(spanCtx, call)
}

// NewTodoFetcherTool creates a new instance of TodoFetcherTool.
func NewTodoFetcherTool(repo domain.TodoRepository, llmCli domain.LLMClient, llmEmbeddingModel string) TodoFetcherTool {
	return TodoFetcherTool{
		repo:              repo,
		llmCli:            llmCli,
		llmEmbeddingModel: llmEmbeddingModel,
	}
}

// TodoFetcherTool is an LLM tool for fetching todos.
type TodoFetcherTool struct {
	repo              domain.TodoRepository
	llmCli            domain.LLMClient
	llmEmbeddingModel string
}

// Tool returns the LLMTool definition for the TodoFetcherTool.
func (lft TodoFetcherTool) Tool() domain.LLMTool {
	return domain.LLMTool{
		Type: "function",
		Function: domain.LLMToolFunction{
			Name:        "fetch_todos",
			Description: "Finds todos using semantic search, pagination, and filtering. Use clear, relevant keywords for best results. All parameters must be integers except 'search_term'.",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
					"page": {
						Type:        "integer",
						Description: "Page number to retrieve (starting from 1, integer).",
						Required:    true,
					},
					"page_size": {
						Type:        "integer",
						Description: "Number of todos per page (max 30, integer).",
						Required:    true,
					},
					"search_term": {
						Type:        "string",
						Description: "Keyword or phrase for semantic search (e.g., 'April tasks', 'overdue', 'shopping').",
						Required:    true,
					},
				},
			},
		},
	}
}

// Call executes the TodoFetcherTool with the provided function call.
func (lft TodoFetcherTool) Call(ctx context.Context, call domain.LLMStreamEventFunctionCall) domain.LLMChatMessage {
	params := struct {
		Page       int    `json:"page"`
		PageSize   int    `json:"page_size"`
		SearchTerm string `json:"search_term"`
	}{
		Page:     1,  // default page
		PageSize: 10, // default page size
	}

	err := json.Unmarshal([]byte(call.Arguments), &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: "Sorry, I couldn't understand the parameters for fetching todos. Please check 'page', 'page_size', and 'search_term'. ERROR: " + err.Error(),
		}
	}

	embedding, err := lft.llmCli.Embed(ctx, lft.llmEmbeddingModel, params.SearchTerm)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: "Sorry, I couldn't process your search term. Please try a different keyword or phrase. ERROR: " + err.Error(),
		}
	}

	todos, hasMore, err := lft.repo.ListTodos(ctx, params.Page, params.PageSize, domain.WithEmbedding(embedding))
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: "Sorry, I couldn't retrieve your todos. Please try again later. ERROR: " + err.Error(),
		}
	}

	if len(todos) == 0 {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: "No todos matched your search.",
		}
	}

	todosInput := make([]string, 0, len(todos))
	for _, t := range todos {
		todosInput = append(todosInput, t.ToLLMInput())
	}

	var nextPage *int
	if hasMore {
		nxt := params.Page + 1
		nextPage = &nxt
	}

	output := map[string]any{
		"todos":     todosInput,
		"next_page": nextPage,
	}
	content, err := json.Marshal(output)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: "Error: failed to marshal tool response: " + err.Error(),
		}
	}

	return domain.LLMChatMessage{
		Role:    domain.ChatRole_Tool,
		Content: string(content),
	}
}

// TodoCreatorTool is an LLM tool for creating todos.
type TodoCreatorTool struct {
	uow     domain.UnitOfWork
	creator TodoCreator
}

// NewTodoCreatorTool creates a new instance of TodoCreatorTool.
func NewTodoCreatorTool(uow domain.UnitOfWork, creator TodoCreator) TodoCreatorTool {
	return TodoCreatorTool{
		uow:     uow,
		creator: creator,
	}
}

// Tool returns the LLMTool definition for the TodoCreatorTool.
func (tct TodoCreatorTool) Tool() domain.LLMTool {
	return domain.LLMTool{
		Type: "function",
		Function: domain.LLMToolFunction{
			Name:        "create_todo",
			Description: "Creates a new todo item with a title and due date. The due date must be a unix timestamp (integer, e.g., 1769904000). Use clear, descriptive titles.",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
					"title": {
						Type:        "string",
						Description: "Title of the todo (required).",
						Required:    true,
					},
					"due_date": {
						Type:        "integer",
						Description: "Due date as a unix timestamp (integer, e.g., 1769904000, required).",
						Required:    true,
					},
				},
			},
		},
	}
}

// Call executes the TodoCreatorTool with the provided function call.
func (tct TodoCreatorTool) Call(ctx context.Context, call domain.LLMStreamEventFunctionCall) domain.LLMChatMessage {
	params := struct {
		Title   string `json:"title"`
		DueDate int64  `json:"due_date"`
	}{}

	err := json.Unmarshal([]byte(call.Arguments), &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: "Sorry, I couldn't understand the parameters for creating a todo. Please provide a title and a valid due date.",
		}
	}
	var dueDate time.Time
	if params.DueDate > 0 {
		dueDate = time.Unix(params.DueDate, 0).UTC()
	}

	var todo domain.Todo
	err = tct.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		td, err := tct.creator.Create(ctx, uow, params.Title, dueDate)
		if err != nil {
			return err
		}
		todo = td
		return nil
	})
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: "Sorry, I couldn't create the todo. Please try again or check your input. ERROR: " + err.Error(),
		}
	}

	return domain.LLMChatMessage{
		Role:    domain.ChatRole_Tool,
		Content: "Your todo was created successfully! Created todo: " + todo.ToLLMInput(),
	}
}

// TodoUpdaterTool is an LLM tool for updating todos.
type TodoUpdaterTool struct {
	uow     domain.UnitOfWork
	updater TodoUpdater
}

// NewTodoUpdaterTool creates a new instance of TodoUpdaterTool.
func NewTodoUpdaterTool(uow domain.UnitOfWork, updater TodoUpdater) TodoUpdaterTool {
	return TodoUpdaterTool{
		uow:     uow,
		updater: updater,
	}
}

// Tool returns the LLMTool definition for the TodoUpdaterTool.
func (tut TodoUpdaterTool) Tool() domain.LLMTool {
	return domain.LLMTool{
		Type: "function",
		Function: domain.LLMToolFunction{
			Name:        "update_todo",
			Description: "Updates an existing todo item. You can change the title, status, or due date. The due date must be a unix timestamp (integer, e.g., 1769904000).",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
					"id": {
						Type:        "string",
						Description: "ID of the todo to update (UUID string, required).",
						Required:    true,
					},
					"title": {
						Type:        "string",
						Description: "New title (optional).",
						Required:    false,
					},
					"status": {
						Type:        "string",
						Description: "New status (OPEN or DONE, optional).",
						Required:    false,
					},
					"due_date": {
						Type:        "integer",
						Description: "New due date as a unix timestamp (integer, e.g., 1769904000, optional).",
						Required:    false,
					},
				},
			},
		},
	}
}

// Call executes the TodoUpdaterTool with the provided function call.
func (tut TodoUpdaterTool) Call(ctx context.Context, call domain.LLMStreamEventFunctionCall) domain.LLMChatMessage {
	params := struct {
		ID      string  `json:"id"`
		Title   *string `json:"title"`
		Status  *string `json:"status"`
		DueDate *int64  `json:"due_date"`
	}{}

	err := json.Unmarshal([]byte(call.Arguments), &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: "Sorry, I couldn't understand the parameters for updating a todo. Please check your input. ERROR: " + err.Error(),
		}
	}

	todoID, err := uuid.Parse(params.ID)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: "Sorry, the todo ID format is invalid. Please use a valid UUID.",
		}
	}

	var dueDate *time.Time
	if params.DueDate != nil {
		parsedDueDate := time.Unix(*params.DueDate, 0).UTC()
		dueDate = &parsedDueDate
	}

	var todo domain.Todo
	err = tut.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		td, err := tut.updater.Update(ctx, uow, todoID, params.Title, (*domain.TodoStatus)(params.Status), dueDate)
		if err != nil {
			return err
		}
		todo = td
		return nil
	})

	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: "Sorry, I couldn't update the todo. Please try again or check your input. ERROR: " + err.Error(),
		}
	}

	return domain.LLMChatMessage{
		Role:    domain.ChatRole_Tool,
		Content: "Your todo was updated successfully! Updated todo: " + todo.ToLLMInput(),
	}
}

type TodoDeleterTool struct {
	uow     domain.UnitOfWork
	deleter TodoDeleter
}

// NewTodoDeleterTool creates a new instance of TodoDeleterTool.
func NewTodoDeleterTool(uow domain.UnitOfWork, deleter TodoDeleter) TodoDeleterTool {
	return TodoDeleterTool{
		uow:     uow,
		deleter: deleter,
	}
}

// Tool returns the LLMTool definition for the TodoDeleterTool.
func (tdt TodoDeleterTool) Tool() domain.LLMTool {
	return domain.LLMTool{
		Type: "function",
		Function: domain.LLMToolFunction{
			Name:        "delete_todo",
			Description: "Deletes an existing todo item by its ID.",
			Parameters: domain.LLMToolFunctionParameters{
				Type: "object",
				Properties: map[string]domain.LLMToolFunctionParameterDetail{
					"id": {
						Type:        "string",
						Description: "ID of the todo to delete (UUID string, required).",
						Required:    true,
					},
				},
			},
		},
	}
}

// Call executes the TodoDeleterTool with the provided function call.
func (tdt TodoDeleterTool) Call(ctx context.Context, call domain.LLMStreamEventFunctionCall) domain.LLMChatMessage {
	params := struct {
		ID string `json:"id"`
	}{}

	err := json.Unmarshal([]byte(call.Arguments), &params)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: "Sorry, I couldn't understand the parameters for deleting a todo. Please check your input. ERROR: " + err.Error(),
		}
	}

	todoID, err := uuid.Parse(params.ID)
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: "Sorry, the todo ID format is invalid. Please use a valid UUID.",
		}
	}

	err = tdt.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		return tdt.deleter.Delete(ctx, uow, todoID)
	})
	if err != nil {
		return domain.LLMChatMessage{
			Role:    domain.ChatRole_Tool,
			Content: "Sorry, I couldn't delete the todo. Please try again or check your input. ERROR: " + err.Error(),
		}
	}

	return domain.LLMChatMessage{
		Role:    domain.ChatRole_Tool,
		Content: "The todo was deleted successfully!",
	}
}
