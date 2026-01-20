import React, { useState } from 'react';
import type { Todo, TodoStatus } from '../types';

interface TodoItemProps {
  todo: Todo;
  onUpdateTodo: (id: string, status: TodoStatus) => void;
  onUpdateTitle: (id: string, title: string) => void;
}

const TodoItem: React.FC<TodoItemProps> = ({ todo, onUpdateTodo, onUpdateTitle }) => {
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [editTitle, setEditTitle] = useState(todo.title);

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const markAsDone = () => {
    onUpdateTodo(todo.id, 'DONE');
  };

  const handleSaveEdit = () => {
    if (editTitle.trim() && editTitle !== todo.title) {
      onUpdateTitle(todo.id, editTitle);
    }
    setIsEditOpen(false);
  };

  const handleCancelEdit = () => {
    setEditTitle(todo.title);
    setIsEditOpen(false);
  };

  return (
    <>
      <div className="todo-card">
        <div className="todo-card-header">
          <h3 className="todo-card-title">{todo.title}</h3>
          <div className="todo-status-badges">
            <span className={`status-badge status-${todo.status.toLowerCase()}`}>
              {todo.status}
            </span>
          </div>
        </div>

        <div className="todo-card-body">
          <div className="todo-info-row">
            <span className="todo-label">Email Status:</span>
            <span className={`status-badge email-${todo.email_status.toLowerCase()}`}>
              {todo.email_status}
            </span>
          </div>

          {todo.email_attempts > 0 && (
            <div className="todo-info-row">
              <span className="todo-label">Attempts:</span>
              <span>{todo.email_attempts}</span>
            </div>
          )}

          {todo.email_last_error && (
            <div className="todo-info-row error-row">
              <span className="todo-label">Error:</span>
              <span className="error-text">{todo.email_last_error}</span>
            </div>
          )}

          {todo.email_provider_id && (
            <div className="todo-info-row">
              <span className="todo-label">Provider ID:</span>
              <span className="provider-id">{todo.email_provider_id}</span>
            </div>
          )}

          <div className="todo-dates">
            <div className="date-item">
              <span className="date-label">Created:</span>
              <span>{formatDate(todo.created_at)}</span>
            </div>
            <div className="date-item">
              <span className="date-label">Updated:</span>
              <span>{formatDate(todo.updated_at)}</span>
            </div>
          </div>
        </div>

        {todo.status === 'OPEN' && (
          <div className="todo-card-footer">
            <button
              className="btn-primary"
              onClick={markAsDone}
            >
              ✓ Mark Done
            </button>
            <button
              className="btn-secondary"
              onClick={() => setIsEditOpen(true)}
            >
              ✏️ Edit
            </button>
          </div>
        )}
      </div>

      {/* Edit Modal */}
      <div className={`modal-overlay ${isEditOpen ? 'active' : ''}`} onClick={handleCancelEdit}>
        <div className="modal-dialog" onClick={(e) => e.stopPropagation()}>
          <div className="modal-header">
            <h2>Edit Todo</h2>
          </div>

          <form onSubmit={(e) => { e.preventDefault(); handleSaveEdit(); }}>
            <div className="modal-content">
              <div className="form-group">
                <label htmlFor="edit-title">Todo Title</label>
                <input
                  id="edit-title"
                  type="text"
                  value={editTitle}
                  onChange={(e) => setEditTitle(e.target.value)}
                  placeholder="Enter todo title..."
                  autoFocus
                />
              </div>
            </div>

            <div className="modal-footer">
              <button 
                type="button" 
                className="btn-secondary"
                onClick={handleCancelEdit}
              >
                Cancel
              </button>
              <button 
                type="submit" 
                className="btn-primary"
                disabled={!editTitle.trim() || editTitle === todo.title}
              >
                Save
              </button>
            </div>
          </form>
        </div>
      </div>
    </>
  );
};

export default TodoItem;