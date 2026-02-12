import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Pencil, Trash2 } from 'lucide-react';
import { incomeApi } from '../../api/income';
import { IncomeForm } from './IncomeForm';
import type { IncomeSource } from '../../types';
import styles from './IncomeList.module.css';

const scheduleLabels: Record<string, string> = {
  weekly: 'Weekly',
  biweekly: 'Every 2 weeks',
  semimonthly: 'Twice a month',
};

export function IncomeList() {
  const queryClient = useQueryClient();
  const [editingSource, setEditingSource] = useState<IncomeSource | null>(null);
  const [showForm, setShowForm] = useState(false);

  const { data: sources = [], isLoading } = useQuery({
    queryKey: ['income-sources'],
    queryFn: () => incomeApi.list(),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => incomeApi.delete(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['income-sources'] }),
  });

  const handleClose = () => {
    setEditingSource(null);
    setShowForm(false);
  };

  if (isLoading) return <div className={styles.loading}>Loading income sources...</div>;

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1>Income Sources</h1>
        <button className={styles.addBtn} onClick={() => setShowForm(true)}>
          <Plus size={18} /> Add Source
        </button>
      </div>

      <div className={styles.list}>
        {sources.map((source) => (
          <div key={source.id} className={styles.card}>
            <div className={styles.cardMain}>
              <div className={styles.name}>{source.name}</div>
              <div className={styles.details}>
                <span className={styles.schedule}>{scheduleLabels[source.pay_schedule] || source.pay_schedule}</span>
                {source.default_amount && (
                  <span>${source.default_amount.toFixed(2)} per pay</span>
                )}
              </div>
            </div>
            <div className={styles.actions}>
              <button
                className={styles.iconBtn}
                onClick={() => { setEditingSource(source); setShowForm(true); }}
              >
                <Pencil size={16} />
              </button>
              <button
                className={`${styles.iconBtn} ${styles.deleteBtn}`}
                onClick={() => deleteMutation.mutate(source.id)}
              >
                <Trash2 size={16} />
              </button>
            </div>
          </div>
        ))}
        {sources.length === 0 && (
          <div className={styles.empty}>
            No income sources yet. Add your pay schedules to get started.
          </div>
        )}
      </div>

      {showForm && <IncomeForm source={editingSource} onClose={handleClose} />}
    </div>
  );
}
