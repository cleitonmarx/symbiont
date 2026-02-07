import type { TodoStatus } from '../../types';
import type { TodoSort } from '../../services/todosApi';

interface TodoControlsBarProps {
  statusFilter: TodoStatus | 'ALL';
  onStatusFilterChange: (status: TodoStatus | 'ALL') => void;
  sortBy: TodoSort;
  onSortByChange: (sort: TodoSort) => void;
  pageSize: number;
  onPageSizeChange: (size: number) => void;
  searchQuery: string;
  onSearchQueryChange: (query: string) => void;
}

export const TodoControlsBar = ({
  statusFilter,
  onStatusFilterChange,
  sortBy,
  onSortByChange,
  pageSize,
  onPageSizeChange,
  searchQuery,
  onSearchQueryChange,
}: TodoControlsBarProps) => {
  return (
    <section className="ui-controls" aria-label="Todo filters and search">
      <div className="ui-controls-group">
        <span className="ui-controls-label">Status</span>
        <div className="ui-segmented">
          {(['ALL', 'OPEN', 'DONE'] as const).map((status) => (
            <button
              key={status}
              type="button"
              className={`ui-segmented-item ${statusFilter === status ? 'active' : ''}`}
              onClick={() => onStatusFilterChange(status)}
            >
              {status}
            </button>
          ))}
        </div>
      </div>

      <div className="ui-controls-group ui-controls-sort">
        <label className="ui-controls-label" htmlFor="todo-sort-select">
          Sort
        </label>
        <select
          id="todo-sort-select"
          className="ui-select"
          value={sortBy}
          onChange={(event) => onSortByChange(event.target.value as TodoSort)}
        >
          <option value="createdAtAsc">Created At</option>
          <option value="createdAtDesc">Created At Desc</option>
          <option value="dueDateAsc">Due Date Asc</option>
          <option value="dueDateDesc">Due Date Desc</option>
          {searchQuery ? (
            <>
              <option value="similarityAsc">Similarity Asc</option>
              <option value="similarityDesc">Similarity Desc</option>
            </>
          ) : null}
        </select>
      </div>

      <div className="ui-controls-group ui-controls-size">
        <label className="ui-controls-label" htmlFor="todo-page-size-select">
          Page Size
        </label>
        <select
          id="todo-page-size-select"
          className="ui-select"
          value={pageSize}
          onChange={(event) => onPageSizeChange(Number(event.target.value))}
        >
          <option value={25}>25</option>
          <option value={50}>50</option>
          <option value={100}>100</option>
        </select>
      </div>

      <div className="ui-controls-group ui-controls-search">
        <label className="ui-controls-label" htmlFor="todo-search-input">
          Search
        </label>
        <input
          id="todo-search-input"
          className="ui-input"
          type="text"
          value={searchQuery}
          placeholder="Search todos"
          onChange={(event) => onSearchQueryChange(event.target.value)}
        />
      </div>
    </section>
  );
};
