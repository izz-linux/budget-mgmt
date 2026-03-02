import { api } from './client';
import type { Bill, BillAssignment, SinkingFundPlan } from '../types';

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

  sinkingFundPlan: (id: number, targetPeriodId: number, numPeriods: number) =>
    api.post<SinkingFundPlan>(`/bills/${id}/sinking-fund/plan`, {
      target_period_id: targetPeriodId,
      num_periods: numPeriods,
    }),

  sinkingFundApply: (id: number, targetPeriodId: number, numPeriods: number) =>
    api.post<BillAssignment[]>(`/bills/${id}/sinking-fund/apply`, {
      target_period_id: targetPeriodId,
      num_periods: numPeriods,
    }),

  sinkingFundClear: (id: number, targetPeriodId: number) =>
    api.delete(`/bills/${id}/sinking-fund?target_period_id=${targetPeriodId}`),
};
