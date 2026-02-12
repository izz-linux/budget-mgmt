import { api } from './client';
import type { Bill } from '../types';

export const billsApi = {
  list: (active?: boolean) =>
    api.get<Bill[]>(`/bills${active ? '?active=true' : ''}`),

  get: (id: number) =>
    api.get<Bill>(`/bills/${id}`),

  create: (data: Partial<Bill> & { credit_card?: Partial<Bill['credit_card']> }) =>
    api.post<Bill>('/bills', data),

  update: (id: number, data: Partial<Bill>) =>
    api.put<Bill>(`/bills/${id}`, data),

  delete: (id: number) =>
    api.delete(`/bills/${id}`),

  reorder: (orders: { id: number; sort_order: number }[]) =>
    api.patch<void>('/bills/reorder', { orders }),
};
