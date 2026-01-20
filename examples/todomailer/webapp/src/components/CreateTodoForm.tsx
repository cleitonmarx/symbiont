import React, { useState } from 'react';

interface CreateTodoFormProps {
  onCreateTodo: (title: string) => void;
}

const CreateTodoForm: React.FC<CreateTodoFormProps> = ({ onCreateTodo }) => {
  const [isOpen, setIsOpen] = useState(false);
  const [title, setTitle] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (title.trim()) {
      onCreateTodo(title.trim());
      setTitle('');
      setIsOpen(false);
    }
  };

  const handleCancel = () => {
    setTitle('');
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
                disabled={!title.trim()}
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