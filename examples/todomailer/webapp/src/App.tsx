import React from 'react';
import CreateTodoForm from './components/CreateTodoForm';
import TodoList from './components/TodoList';
import { useTodos } from './hooks/useTodos';

const App: React.FC = () => {
  const { 
    todos, 
    loading, 
    error,
    createTodo, 
    updateTodo,
    updateTitle,
    statusFilter,
    setStatusFilter,
    page,
    previousPage,
    nextPage,
    goToPage,
  } = useTodos();

  return (
    <div className="app">
      <header className="app-header">
        <h1>Todo Mailer</h1>
      </header>
      <div className="app-main">
        <div className="sidebar-toolbar">
          <CreateTodoForm onCreateTodo={createTodo} />
        </div>
        <div className="content-area">
          {error && (
            <div className="error" style={{ marginBottom: '1.5rem' }}>
              <strong>Error:</strong> {error}
            </div>
          )}
          <TodoList 
            todos={todos} 
            loading={loading}
            error={null}
            onUpdateTodo={updateTodo}
            onUpdateTitle={updateTitle}
            statusFilter={statusFilter}
            onStatusFilterChange={setStatusFilter}
            currentPage={page}
            previousPage={previousPage}
            nextPage={nextPage}
            onPreviousPage={() => previousPage !== null && goToPage(previousPage)}
            onNextPage={() => nextPage !== null && goToPage(nextPage)}
          />
        </div>
      </div>
    </div>
  );
};

export default App;