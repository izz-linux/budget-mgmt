import { api } from './client';
import type { PayPeriod } from '../types';

export const periodsApi = {
  list: (from: string, to: string) =>
    api.get<PayPeriod[]>(`/pay-periods?from=${from}&to=${to}`),

  generate: (from: string, to: string, sourceIds?: number[]) =>
    api.post<PayPeriod[]>('/pay-periods/generate', {
      from, to, source_ids: sourceIds || [],
    }),

  update: (id: number, data: Partial<PayPeriod>) =>
    api.put<PayPeriod>(`/pay-periods/${id}`, data),
};
