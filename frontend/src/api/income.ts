import { api } from './client';
import type { IncomeSource } from '../types';

export const incomeApi = {
  list: (active?: boolean) =>
    api.get<IncomeSource[]>(`/income-sources${active ? '?active=true' : ''}`),

  get: (id: number) =>
    api.get<IncomeSource>(`/income-sources/${id}`),

  create: (data: Partial<IncomeSource>) =>
    api.post<IncomeSource>('/income-sources', data),

  update: (id: number, data: Partial<IncomeSource>) =>
    api.put<IncomeSource>(`/income-sources/${id}`, data),

  delete: (id: number) =>
    api.delete(`/income-sources/${id}`),
};
