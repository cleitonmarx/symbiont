import React from 'react';
import CreateTodoForm from './components/CreateTodoForm';
import TodoList from './components/TodoList';
import { useTodos } from './hooks/useTodos';

const App: React.FC = () => {
  const { 
    todos, 
    boardSummary,
    loading, 
    error,
    createTodo, 
    updateTodo,
    deleteTodo,
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
        <div className="header-content">
          <img src="/symbiont-icon.png" alt="Todo App Logo" className="header-logo" />
          <h1>Todo App</h1>
        </div>
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
          {loading ? (
            <div className="loading">Loading...</div>
          ) : (
            <TodoList 
              todos={todos} 
              boardSummary={boardSummary}
              onUpdate={updateTodo}
              onDelete={deleteTodo}
              statusFilter={statusFilter}
              onStatusFilterChange={setStatusFilter}
              currentPage={page}
              previousPage={previousPage}
              nextPage={nextPage}
              onPreviousPage={() => previousPage !== null && goToPage(previousPage)}
              onNextPage={() => nextPage !== null && goToPage(nextPage)}
              loading={loading}
              error={null}
            />
          )}
        </div>
      </div>
    </div>
  );
};

export default App;