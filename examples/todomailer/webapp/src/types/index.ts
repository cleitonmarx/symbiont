// src/types/index.ts

export type TodoStatus = 'OPEN' | 'DONE';

export interface Todo {
  id: string;
  title: string;
  status: TodoStatus;
  due_date: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTodoRequest {
  title: string;
  due_date: string;
}

export interface UpdateTodoRequest {
  title?: string;
  status?: TodoStatus;
  due_date?: string;
}

export interface ListTodosResponse {
  items: Todo[];
  page: number;
  previous_page: number | null;
  next_page: number | null;
}

export interface ErrorResponse {
  error: {
    code: 'BAD_REQUEST' | 'NOT_FOUND' | 'INTERNAL_ERROR';
    message: string;
  };
}