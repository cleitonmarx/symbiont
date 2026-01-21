import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getTodos, createTodo, updateTodo } from '../services/api';
import type { Todo, CreateTodoRequest, UpdateTodoRequest, TodoStatus } from '../types';
import { useState, useEffect } from 'react';

export const useTodos = () => {
  const queryClient = useQueryClient();
  const [statusFilter, setStatusFilterState] = useState<TodoStatus | 'ALL'>('ALL');
  const [currentPage, setCurrentPage] = useState<number>(1);
  const [mutationError, setMutationError] = useState<string | null>(null);

  // Reset page to 1 whenever status filter changes
  useEffect(() => {
    setCurrentPage(1);
  }, [statusFilter]);

  const { 
    data: response, 
    isLoading: loading, 
    error, 
    refetch 
  } = useQuery({
    queryKey: ['todos', statusFilter, currentPage],
    queryFn: () => getTodos(
      statusFilter === 'ALL' ? undefined : statusFilter,
      currentPage,
      6 // You can adjust this value as needed
    ),
    refetchInterval: 5000,
    retry: 1,
  });

  const todos = response?.items || [];
  const page = response?.page;
  const previousPage = response?.previous_page;
  const nextPage = response?.next_page;

  const errorMessage = error 
    ? error instanceof Error 
      ? error.message 
      : String(error)
    : mutationError;

  const createMutation = useMutation({
    mutationFn: (title: string) => createTodo({ title }),
    onSuccess: () => {
      setMutationError(null);
      setCurrentPage(1);
      queryClient.invalidateQueries({ queryKey: ['todos'] });
    },
    onError: (err: Error) => {
      setMutationError(err.message);
    },
  });

  const updateStatusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: TodoStatus }) => 
      updateTodo(id, { status }),
    onSuccess: () => {
      setMutationError(null);
      queryClient.invalidateQueries({ queryKey: ['todos'] });
    },
    onError: (err: Error) => {
      setMutationError(err.message);
    },
  });

  const updateTitleMutation = useMutation({
    mutationFn: ({ id, title }: { id: string; title: string }) => 
      updateTodo(id, { title }),
    onSuccess: () => {
      setMutationError(null);
      queryClient.invalidateQueries({ queryKey: ['todos'] });
    },
    onError: (err: Error) => {
      setMutationError(err.message);
    },
  });

  return {
    todos,
    loading,
    error: errorMessage,
    createTodo: (title: string) => createMutation.mutate(title),
    updateTodo: (id: string, status: TodoStatus) => 
      updateStatusMutation.mutate({ id, status }),
    updateTitle: (id: string, title: string) => 
      updateTitleMutation.mutate({ id, title }),
    refetch,
    statusFilter,
    setStatusFilter: setStatusFilterState,
    page,
    previousPage,
    nextPage,
    goToPage: setCurrentPage,
  };
};