import { api } from './client';
import type { BudgetGridData } from '../types';

export const gridApi = {
  get: (from: string, to: string) =>
    api.get<BudgetGridData>(`/budget-grid?from=${from}&to=${to}`),
};
