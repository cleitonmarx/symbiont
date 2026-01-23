// src/types/index.ts

export type TodoStatus = 'OPEN' | 'DONE';
export type EmailStatus = 'PENDING' | 'SENT' | 'FAILED';

export interface Todo {
  id: string;
  title: string;
  status: TodoStatus;
  due_date: string;
  email_status: EmailStatus;
  email_attempts: number;
  email_last_error: string | null;
  email_provider_id: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateTodoRequest {
  title: string;
  due_date: string;
}

export interface UpdateTodoRequest {
  title?: string;
  due_date?: string;
  status?: string;
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