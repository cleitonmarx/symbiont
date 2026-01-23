import React, { useState } from 'react';
import type { Todo, TodoStatus } from '../types';

export interface TodoItemProps {
  todo: Todo;
  onComplete: (id: string, status: TodoStatus) => void;
  onUpdateTitle: (id: string, title: string, due_date: string) => void;
}

const TodoItem: React.FC<TodoItemProps> = ({ todo, onComplete, onUpdateTitle }) => {
  const [isEditOpen, setIsEditOpen] = useState(false);
  const [editTitle, setEditTitle] = useState(todo.title);
  const [editDueDate, setEditDueDate] = useState(todo.due_date);

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

  const formatDueDate = (dateString: string) => {
    const [year, month, day] = dateString.split('-').map(Number);
    const date = new Date(year, month - 1, day);
    return date.toLocaleDateString('en-US', { 
      month: 'short',
      day: 'numeric',
      year: 'numeric'
    });
  };

  const getMinDate = () => {
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    yesterday.setHours(0, 0, 0, 0);
    const year = yesterday.getFullYear();
    const month = String(yesterday.getMonth() + 1).padStart(2, '0');
    const day = String(yesterday.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  };

  const markAsDone = () => {
    onComplete(todo.id, 'DONE');
  };

  const handleSaveEdit = () => {
    if (editTitle.trim() && editDueDate && (editTitle !== todo.title || editDueDate !== todo.due_date)) {
      onUpdateTitle(todo.id, editTitle.trim(), editDueDate);
    }
    setIsEditOpen(false);
  };

  const handleCancelEdit = () => {
    setEditTitle(todo.title);
    setEditDueDate(todo.due_date);
    setIsEditOpen(false);
  };

  const getDueDateColor = () => {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    
    const [year, month, day] = todo.due_date.split('-').map(Number);
    const dueDate = new Date(year, month - 1, day);
    dueDate.setHours(0, 0, 0, 0);

    if (todo.status === 'DONE') return '#000000'; // black - task is done
    if (dueDate < today) return '#E74C3C'; // red - overdue
    if (dueDate.getTime() === today.getTime()) return '#FFB84D'; // yellow - due today
    return '#4A9DD4'; // blue - due in future
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
          <div className="due-date-row">
            <span className="due-date-label">Due:</span>
            <span className="due-date-value" style={{ color: getDueDateColor() }}>
              {formatDueDate(todo.due_date)}
            </span>
          </div>

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
                  required
                />
              </div>
              <div className="form-group">
                <label htmlFor="edit-due-date">Due Date</label>
                <input
                  id="edit-due-date"
                  type="date"
                  value={editDueDate}
                  onChange={(e) => setEditDueDate(e.target.value)}
                  min={getMinDate()}
                  required
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
                disabled={!editTitle.trim() || !editDueDate || (editTitle === todo.title && editDueDate === todo.due_date)}
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