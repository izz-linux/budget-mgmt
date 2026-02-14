import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Plus, Pencil, Trash2, CreditCard, Zap } from 'lucide-react';
import { billsApi } from '../../api/bills';
import { BillForm } from './BillForm';
import type { Bill } from '../../types';
import styles from './BillList.module.css';

export function BillList() {
  const queryClient = useQueryClient();
  const [editingBill, setEditingBill] = useState<Bill | null>(null);
  const [showForm, setShowForm] = useState(false);

  const { data: bills = [], isLoading } = useQuery({
    queryKey: ['bills'],
    queryFn: () => billsApi.list(true),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => billsApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['bills'] });
      queryClient.invalidateQueries({ queryKey: ['budget-grid'] });
    },
  });

  const handleEdit = (bill: Bill) => {
    setEditingBill(bill);
    setShowForm(true);
  };

  const handleClose = () => {
    setEditingBill(null);
    setShowForm(false);
  };

  if (isLoading) return <div className={styles.loading}>Loading bills...</div>;

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1>Bills & Expenses</h1>
        <button className={styles.addBtn} onClick={() => setShowForm(true)}>
          <Plus size={18} /> Add Bill
        </button>
      </div>

      <div className={styles.list}>
        {bills.map((bill) => (
          <div key={bill.id} className={styles.card}>
            <div className={styles.cardMain}>
              <div className={styles.cardTitle}>
                <span className={styles.name}>{bill.name}</span>
                <div className={styles.badges}>
                  {bill.is_autopay && (
                    <span className={styles.badgeAuto} title="Autopay">
                      <Zap size={12} /> Auto
                    </span>
                  )}
                  {bill.credit_card && (
                    <span className={styles.badgeCC} title="Credit Card">
                      <CreditCard size={12} />
                      {bill.credit_card.card_label || bill.credit_card.issuer}
                    </span>
                  )}
                  {bill.category && (
                    <span className={styles.badgeCat}>{bill.category}</span>
                  )}
                </div>
              </div>
              <div className={styles.details}>
                {bill.due_day && <span>Due: {ordinal(bill.due_day)}</span>}
                {bill.default_amount && <span>${bill.default_amount.toFixed(2)}</span>}
                <span className={styles.recurrence}>{bill.recurrence}</span>
                {bill.credit_card && (
                  <span className={styles.ccDates}>
                    Statement: {ordinal(bill.credit_card.statement_day)} / Due: {ordinal(bill.credit_card.due_day)}
                  </span>
                )}
              </div>
            </div>
            <div className={styles.actions}>
              <button className={styles.iconBtn} onClick={() => handleEdit(bill)} title="Edit">
                <Pencil size={16} />
              </button>
              <button
                className={`${styles.iconBtn} ${styles.deleteBtn}`}
                onClick={() => deleteMutation.mutate(bill.id)}
                title="Delete"
              >
                <Trash2 size={16} />
              </button>
            </div>
          </div>
        ))}
        {bills.length === 0 && (
          <div className={styles.empty}>
            No bills yet. Add your first bill to get started.
          </div>
        )}
      </div>

      {showForm && (
        <BillForm
          bill={editingBill}
          onClose={handleClose}
        />
      )}
    </div>
  );
}

function ordinal(n: number): string {
  const s = ['th', 'st', 'nd', 'rd'];
  const v = n % 100;
  return n + (s[(v - 20) % 10] || s[v] || s[0]);
}
