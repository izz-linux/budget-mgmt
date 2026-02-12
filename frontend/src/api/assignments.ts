import { api } from './client';
import type { BillAssignment } from '../types';

export const assignmentsApi = {
  list: (params?: { period_id?: number; bill_id?: number; status?: string }) => {
    const query = new URLSearchParams();
    if (params?.period_id) query.set('period_id', String(params.period_id));
    if (params?.bill_id) query.set('bill_id', String(params.bill_id));
    if (params?.status) query.set('status', params.status);
    return api.get<BillAssignment[]>(`/assignments?${query}`);
  },

  create: (data: Partial<BillAssignment>) =>
    api.post<BillAssignment>('/assignments', data),

  update: (id: number, data: Partial<BillAssignment>) =>
    api.put<BillAssignment>(`/assignments/${id}`, data),

  updateStatus: (id: number, status: string, deferredToId?: number) =>
    api.patch<BillAssignment>(`/assignments/${id}/status`, {
      status,
      deferred_to_id: deferredToId,
    }),

  delete: (id: number) =>
    api.delete(`/assignments/${id}`),
};
