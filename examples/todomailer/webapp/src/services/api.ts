import axios from 'axios';
import type { Todo, CreateTodoRequest, UpdateTodoRequest, ListTodosResponse } from '../types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add response interceptor to handle errors
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    console.error('API Error:', error);
    
    if (error.response) {
      const errorData = error.response.data?.error;
      const message = errorData?.message || error.response.statusText || 'An error occurred';
      const status = error.response.status;
      throw new Error(`[${status}] ${message}`);
    } else if (error.request) {
      throw new Error('No response from server');
    } else {
      throw new Error(error.message);
    }
  }
);

export const getTodos = async (
  status?: string,
  page: number = 1,
  pagesize: number = 50
): Promise<ListTodosResponse> => {
  const params: Record<string, any> = {
    page,
    pagesize,
  };
  
  if (status) {
    params.status = status;
  }

  const response = await apiClient.get<ListTodosResponse>('/api/v1/todos', { params });
  return response.data;
};

export const createTodo = async (request: CreateTodoRequest): Promise<Todo> => {
  const response = await apiClient.post<Todo>('/api/v1/todos', request);
  return response.data;
};

export const updateTodo = async (id: string, request: UpdateTodoRequest): Promise<Todo> => {
  const response = await apiClient.patch<Todo>(`/api/v1/todos/${id}`, request);
  return response.data;
};