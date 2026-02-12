import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { X } from 'lucide-react';
import { billsApi } from '../../api/bills';
import type { Bill } from '../../types';
import styles from './BillForm.module.css';

interface BillFormProps {
  bill: Bill | null;
  onClose: () => void;
}

export function BillForm({ bill, onClose }: BillFormProps) {
  const queryClient = useQueryClient();
  const isEditing = !!bill;

  const [form, setForm] = useState({
    name: bill?.name || '',
    default_amount: bill?.default_amount ?? '',
    due_day: bill?.due_day ?? '',
    recurrence: bill?.recurrence || 'monthly',
    is_autopay: bill?.is_autopay || false,
    category: bill?.category || '',
    notes: bill?.notes || '',
    // Credit card fields
    is_credit_card: !!bill?.credit_card,
    cc_card_label: bill?.credit_card?.card_label || '',
    cc_statement_day: bill?.credit_card?.statement_day ?? '',
    cc_due_day: bill?.credit_card?.due_day ?? '',
    cc_issuer: bill?.credit_card?.issuer || '',
  });

  const createMutation = useMutation({
    mutationFn: (data: Parameters<typeof billsApi.create>[0]) => billsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bills'] });
      onClose();
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<Bill> }) => billsApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bills'] });
      onClose();
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    const payload: Record<string, unknown> = {
      name: form.name,
      default_amount: form.default_amount ? Number(form.default_amount) : null,
      due_day: form.due_day ? Number(form.due_day) : null,
      recurrence: form.recurrence,
      is_autopay: form.is_autopay,
      category: form.category,
      notes: form.notes,
    };

    if (form.is_credit_card) {
      payload.credit_card = {
        card_label: form.cc_card_label,
        statement_day: Number(form.cc_statement_day),
        due_day: Number(form.cc_due_day),
        issuer: form.cc_issuer,
      };
    }

    if (isEditing) {
      updateMutation.mutate({ id: bill.id, data: payload as Partial<Bill> });
    } else {
      createMutation.mutate(payload as Parameters<typeof billsApi.create>[0]);
    }
  };

  const set = (field: string, value: unknown) =>
    setForm((prev) => ({ ...prev, [field]: value }));

  return (
    <div className={styles.overlay} onClick={onClose}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <div className={styles.modalHeader}>
          <h2>{isEditing ? 'Edit Bill' : 'Add Bill'}</h2>
          <button className={styles.closeBtn} onClick={onClose}>
            <X size={20} />
          </button>
        </div>

        <form onSubmit={handleSubmit} className={styles.form}>
          <div className={styles.field}>
            <label>Name *</label>
            <input
              type="text"
              value={form.name}
              onChange={(e) => set('name', e.target.value)}
              placeholder="e.g. Verizon"
              required
            />
          </div>

          <div className={styles.row}>
            <div className={styles.field}>
              <label>Default Amount</label>
              <input
                type="number"
                step="0.01"
                value={form.default_amount}
                onChange={(e) => set('default_amount', e.target.value)}
                placeholder="0.00"
              />
            </div>
            <div className={styles.field}>
              <label>Due Day</label>
              <input
                type="number"
                min="1"
                max="31"
                value={form.due_day}
                onChange={(e) => set('due_day', e.target.value)}
                placeholder="1-31"
              />
            </div>
          </div>

          <div className={styles.row}>
            <div className={styles.field}>
              <label>Recurrence</label>
              <select value={form.recurrence} onChange={(e) => set('recurrence', e.target.value)}>
                <option value="monthly">Monthly</option>
                <option value="biweekly">Biweekly</option>
                <option value="weekly">Weekly</option>
                <option value="quarterly">Quarterly</option>
                <option value="annual">Annual</option>
                <option value="irregular">Irregular</option>
              </select>
            </div>
            <div className={styles.field}>
              <label>Category</label>
              <select value={form.category} onChange={(e) => set('category', e.target.value)}>
                <option value="">None</option>
                <option value="housing">Housing</option>
                <option value="utilities">Utilities</option>
                <option value="transportation">Transportation</option>
                <option value="insurance">Insurance</option>
                <option value="debt">Debt</option>
                <option value="savings">Savings</option>
                <option value="subscriptions">Subscriptions</option>
                <option value="personal">Personal</option>
                <option value="other">Other</option>
              </select>
            </div>
          </div>

          <div className={styles.checkRow}>
            <label className={styles.checkbox}>
              <input
                type="checkbox"
                checked={form.is_autopay}
                onChange={(e) => set('is_autopay', e.target.checked)}
              />
              Autopay enabled
            </label>
            <label className={styles.checkbox}>
              <input
                type="checkbox"
                checked={form.is_credit_card}
                onChange={(e) => set('is_credit_card', e.target.checked)}
              />
              Credit card
            </label>
          </div>

          {form.is_credit_card && (
            <div className={styles.ccSection}>
              <div className={styles.row}>
                <div className={styles.field}>
                  <label>Issuer</label>
                  <input
                    type="text"
                    value={form.cc_issuer}
                    onChange={(e) => set('cc_issuer', e.target.value)}
                    placeholder="e.g. Chase"
                  />
                </div>
                <div className={styles.field}>
                  <label>Card Label</label>
                  <input
                    type="text"
                    value={form.cc_card_label}
                    onChange={(e) => set('cc_card_label', e.target.value)}
                    placeholder="e.g. QS ***8186"
                  />
                </div>
              </div>
              <div className={styles.row}>
                <div className={styles.field}>
                  <label>Statement Day</label>
                  <input
                    type="number"
                    min="1"
                    max="31"
                    value={form.cc_statement_day}
                    onChange={(e) => set('cc_statement_day', e.target.value)}
                  />
                </div>
                <div className={styles.field}>
                  <label>Due Day</label>
                  <input
                    type="number"
                    min="1"
                    max="31"
                    value={form.cc_due_day}
                    onChange={(e) => set('cc_due_day', e.target.value)}
                  />
                </div>
              </div>
            </div>
          )}

          <div className={styles.field}>
            <label>Notes</label>
            <textarea
              value={form.notes}
              onChange={(e) => set('notes', e.target.value)}
              rows={2}
              placeholder="Any notes..."
            />
          </div>

          <div className={styles.formActions}>
            <button type="button" className={styles.cancelBtn} onClick={onClose}>
              Cancel
            </button>
            <button type="submit" className={styles.submitBtn}>
              {isEditing ? 'Update' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
