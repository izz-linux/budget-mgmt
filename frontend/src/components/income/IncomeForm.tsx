import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { X } from 'lucide-react';
import { incomeApi } from '../../api/income';
import { periodsApi } from '../../api/periods';
import { assignmentsApi } from '../../api/assignments';
import { useBudgetStore } from '../../stores/budgetStore';
import type { IncomeSource } from '../../types';
import styles from '../bills/BillForm.module.css'; // reuse bill form styles

interface IncomeFormProps {
  source: IncomeSource | null;
  onClose: () => void;
}

export function IncomeForm({ source, onClose }: IncomeFormProps) {
  const queryClient = useQueryClient();
  const { dateRange } = useBudgetStore();
  const isEditing = !!source;

  const detail = source?.schedule_detail as Record<string, unknown> | undefined;

  const [form, setForm] = useState({
    name: source?.name || '',
    pay_schedule: source?.pay_schedule || 'biweekly',
    default_amount: source?.default_amount ?? '',
    weekday: (detail?.weekday as number) ?? 5, // Friday
    anchor_date: (detail?.anchor_date as string) || '',
    semi_day1: ((detail?.days as number[]) || [1, 16])[0],
    semi_day2: ((detail?.days as number[]) || [1, 16])[1],
    one_time_date: (detail?.date as string) || new Date().toISOString().split('T')[0],
    start_date: new Date().toISOString().split('T')[0],
  });

  const createMutation = useMutation({
    mutationFn: (data: Partial<IncomeSource>) => incomeApi.create(data),
    onSuccess: async (created) => {
      queryClient.invalidateQueries({ queryKey: ['income-sources'] });
      // Auto-generate pay periods
      let from: string;
      let to: string;
      if (form.pay_schedule === 'one_time') {
        // For one-time, ensure the date falls within the generation range
        from = form.one_time_date;
        to = form.one_time_date;
      } else {
        from = form.start_date || dateRange.from;
        to = dateRange.to;
      }
      try {
        await periodsApi.generate(from, to, [created.id]);
        await assignmentsApi.autoAssign(dateRange.from, dateRange.to);
        queryClient.invalidateQueries({ queryKey: ['budget-grid'] });
      } catch {
        // Period generation is best-effort; user can retry from budget grid
      }
      onClose();
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<IncomeSource> }) => incomeApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['income-sources'] });
      onClose();
    },
  });

  const buildScheduleDetail = () => {
    switch (form.pay_schedule) {
      case 'weekly':
        return { weekday: Number(form.weekday) };
      case 'biweekly':
        return { weekday: Number(form.weekday), anchor_date: form.anchor_date };
      case 'semimonthly':
        return { days: [Number(form.semi_day1), Number(form.semi_day2)] };
      case 'one_time':
        return { date: form.one_time_date };
      default:
        return {};
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const payload = {
      name: form.name,
      pay_schedule: form.pay_schedule,
      schedule_detail: buildScheduleDetail(),
      default_amount: form.default_amount ? Number(form.default_amount) : null,
    };

    if (isEditing) {
      updateMutation.mutate({ id: source.id, data: payload });
    } else {
      createMutation.mutate(payload);
    }
  };

  const set = (field: string, value: unknown) =>
    setForm((prev) => ({ ...prev, [field]: value }));

  const weekdays = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];

  return (
    <div className={styles.overlay} onClick={onClose}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <div className={styles.modalHeader}>
          <h2>{isEditing ? 'Edit Income Source' : 'Add Income Source'}</h2>
          <button className={styles.closeBtn} onClick={onClose}><X size={20} /></button>
        </div>

        <form onSubmit={handleSubmit} className={styles.form}>
          <div className={styles.field}>
            <label>Name *</label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => set('name', e.target.value)}
              placeholder="e.g. Bayside"
              required
            />
          </div>

          <div className={styles.row}>
            <div className={styles.field}>
              <label>Pay Schedule</label>
              <select value={form.pay_schedule} onChange={(e) => set('pay_schedule', e.target.value)}>
                <option value="weekly">Weekly</option>
                <option value="biweekly">Biweekly</option>
                <option value="semimonthly">Twice a month</option>
                <option value="one_time">One-time (bonus, etc.)</option>
              </select>
            </div>
            <div className={styles.field}>
              <label>Amount per Pay</label>
              <input
                type="number"
                step="0.01"
                value={form.default_amount}
                onChange={(e) => set('default_amount', e.target.value)}
                placeholder="0.00"
              />
            </div>
          </div>

          {(form.pay_schedule === 'weekly' || form.pay_schedule === 'biweekly') && (
            <div className={styles.field}>
              <label>Pay Day</label>
              <select value={form.weekday} onChange={(e) => set('weekday', e.target.value)}>
                {weekdays.map((day, i) => (
                  <option key={i} value={i}>{day}</option>
                ))}
              </select>
            </div>
          )}

          {form.pay_schedule === 'biweekly' && (
            <div className={styles.field}>
              <label>Anchor Date (a known pay date)</label>
              <input
                type="date"
                value={form.anchor_date}
                onChange={(e) => set('anchor_date', e.target.value)}
              />
            </div>
          )}

          {form.pay_schedule === 'semimonthly' && (
            <div className={styles.row}>
              <div className={styles.field}>
                <label>First Pay Day</label>
                <input
                  type="number"
                  min="1"
                  max="31"
                  value={form.semi_day1}
                  onChange={(e) => set('semi_day1', e.target.value)}
                />
              </div>
              <div className={styles.field}>
                <label>Second Pay Day</label>
                <input
                  type="number"
                  min="1"
                  max="31"
                  value={form.semi_day2}
                  onChange={(e) => set('semi_day2', e.target.value)}
                />
              </div>
            </div>
          )}

          {form.pay_schedule === 'one_time' && (
            <div className={styles.field}>
              <label>Date</label>
              <input
                type="date"
                value={form.one_time_date}
                onChange={(e) => set('one_time_date', e.target.value)}
                required
              />
            </div>
          )}

          {!isEditing && form.pay_schedule !== 'one_time' && (
            <div className={styles.field}>
              <label>Start Date (generate periods from)</label>
              <input
                type="date"
                value={form.start_date}
                onChange={(e) => set('start_date', e.target.value)}
              />
            </div>
          )}

          <div className={styles.formActions}>
            <button type="button" className={styles.cancelBtn} onClick={onClose}>Cancel</button>
            <button type="submit" className={styles.submitBtn}>{isEditing ? 'Update' : 'Create'}</button>
          </div>
        </form>
      </div>
    </div>
  );
}
