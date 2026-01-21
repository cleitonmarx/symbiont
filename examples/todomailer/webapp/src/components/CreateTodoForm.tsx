import React, { useState } from 'react';

interface CreateTodoFormProps {
  onCreateTodo: (title: string, due_date: string) => void;
}

const CreateTodoForm: React.FC<CreateTodoFormProps> = ({ onCreateTodo }) => {
  const [isOpen, setIsOpen] = useState(false);
  const [title, setTitle] = useState('');
  const [due_date, setDueDate] = useState('');

  const getMinDate = () => {
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    yesterday.setHours(0, 0, 0, 0);
    const year = yesterday.getFullYear();
    const month = String(yesterday.getMonth() + 1).padStart(2, '0');
    const day = String(yesterday.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (title.trim() && due_date) {
      onCreateTodo(title.trim(), due_date);
      setTitle('');
      setDueDate('');
      setIsOpen(false);
    }
  };

  const handleCancel = () => {
    setTitle('');
    setDueDate('');
    setIsOpen(false);
  };

  return (
    <>
      <button 
        className="toolbar-button" 
        onClick={() => setIsOpen(true)}
        title="Create new todo"
      >
        ➕
      </button>

      <div className={`modal-overlay ${isOpen ? 'active' : ''}`} onClick={handleCancel}>
        <div className="modal-dialog" onClick={(e) => e.stopPropagation()}>
          <div className="modal-header">
            <h2>Create New Todo</h2>
          </div>

          <form onSubmit={handleSubmit}>
            <div className="modal-content">
              <div className="form-group">
                <label htmlFor="todo-title">Todo Title</label>
                <input
                  id="todo-title"
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="Enter todo title..."
                  autoFocus
                />
              </div>
              <div className="form-group">
                <label htmlFor="todo-due-date">Due Date</label>
                <input
                  id="todo-due-date"
                  type="date"
                  value={due_date}
                  onChange={(e) => setDueDate(e.target.value)}
                  min={getMinDate()}
                />
              </div>
            </div>

            <div className="modal-footer">
              <button 
                type="button" 
                className="btn-secondary"
                onClick={handleCancel}
              >
                Cancel
              </button>
              <button 
                type="submit" 
                className="btn-primary"
                disabled={!title.trim() || !due_date}
              >
                Create
              </button>
            </div>
          </form>
        </div>
      </div>
    </>
  );
};

export default CreateTodoForm;