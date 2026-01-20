import React from 'react';
import type { Todo, TodoStatus } from '../types';
import TodoItem from './TodoItem';

interface TodoListProps {
  todos: Todo[];
  loading: boolean;
  error: string | null;
  onUpdateTodo: (id: string, status: TodoStatus) => void;
  onUpdateTitle: (id: string, title: string) => void;
  statusFilter: TodoStatus | 'ALL';
  onStatusFilterChange: (status: TodoStatus | 'ALL') => void;
  currentPage?: number;
  previousPage?: number | null;
  nextPage?: number | null;
  onPreviousPage: () => void;
  onNextPage: () => void;
}

const TodoList: React.FC<TodoListProps> = ({
  todos,
  loading,
  error,
  onUpdateTodo,
  onUpdateTitle,
  statusFilter,
  onStatusFilterChange,
  currentPage,
  previousPage,
  nextPage,
  onPreviousPage,
  onNextPage,
}) => {
  return (
    <div className="todo-list">
      <div className="todo-list-header">
        <div>
          <h2>Todos</h2>
        </div>
      </div>

      {/* Error Message */}
      {error && (
        <div className="error">
          {error}
        </div>
      )}

      {/* Filters */}
      <div className="filter-bar">
        <div className="filter-group">
          <label>Status:</label>
          <div className="filter-buttons">
            {(['ALL', 'OPEN', 'DONE'] as const).map((status) => (
              <button
                key={status}
                className={`filter-button ${statusFilter === status ? 'active' : ''}`}
                onClick={() => onStatusFilterChange(status)}
              >
                {status}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Loading */}
      {loading && todos.length === 0 && (
        <div className="loading">Loading todos...</div>
      )}

      {/* Empty State */}
      {!loading && todos.length === 0 && !error && (
        <div className="empty-state">
          <p>No todos yet. Create one to get started!</p>
        </div>
      )}

      {/* Todos Grid */}
      {todos.length > 0 && (
        <>
          <div className="todos-grid">
            {todos.map((todo) => (
              <TodoItem
                key={todo.id}
                todo={todo}
                onUpdateTodo={onUpdateTodo}
                onUpdateTitle={onUpdateTitle}
              />
            ))}
          </div>

          {/* Pagination */}
          <div className="pagination">
            <div className="pagination-info">
              Page {currentPage || 1}
            </div>
            <div className="pagination-buttons">
              <button
                className="btn-secondary"
                onClick={onPreviousPage}
                disabled={previousPage === null}
              >
                ← Previous
              </button>
              <button
                className="btn-secondary"
                onClick={onNextPage}
                disabled={nextPage === null}
              >
                Next →
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
};

export default TodoList;