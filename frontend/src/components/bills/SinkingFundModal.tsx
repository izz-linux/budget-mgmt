import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { X, AlertTriangle, CheckCircle } from 'lucide-react';
import { periodsApi } from '../../api/periods';
import { billsApi } from '../../api/bills';
import type { SinkingFundPlan } from '../../types';
import { formatShortDate } from '../../utils/date';
import styles from './SinkingFundModal.module.css';

interface SinkingFundModalProps {
  billId: number;
  billName: string;
  defaultNumPeriods: number | null;
  onClose: () => void;
}

export function SinkingFundModal({
  billId,
  billName,
  defaultNumPeriods,
  onClose,
}: SinkingFundModalProps) {
  const queryClient = useQueryClient();

  const [targetPeriodId, setTargetPeriodId] = useState<number | null>(null);
  const [numPeriods, setNumPeriods] = useState(defaultNumPeriods ?? 6);
  const [plan, setPlan] = useState<SinkingFundPlan | null>(null);
  const [planError, setPlanError] = useState('');

  // Fetch ~12 months of future periods for the dropdown
  const today = new Date().toISOString().split('T')[0];
  const oneYearOut = new Date(Date.now() + 365 * 24 * 60 * 60 * 1000)
    .toISOString()
    .split('T')[0];

  const { data: periods = [] } = useQuery({
    queryKey: ['pay-periods', today, oneYearOut],
    queryFn: () => periodsApi.list(today, oneYearOut),
  });

  const planMutation = useMutation({
    mutationFn: () => billsApi.sinkingFundPlan(billId, targetPeriodId!, numPeriods),
    onSuccess: (data) => {
      setPlan(data);
      setPlanError('');
    },
    onError: (err: Error) => {
      setPlanError(err.message);
      setPlan(null);
    },
  });

  const applyMutation = useMutation({
    mutationFn: () => billsApi.sinkingFundApply(billId, targetPeriodId!, numPeriods),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['budget-grid'], exact: false });
      onClose();
    },
  });

  const clearMutation = useMutation({
    mutationFn: () => billsApi.sinkingFundClear(billId, targetPeriodId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['budget-grid'], exact: false });
      setPlan(null);
    },
  });

  const handleGeneratePlan = () => {
    if (!targetPeriodId || numPeriods <= 0) return;
    planMutation.mutate();
  };

  const formatAmount = (n: number) => `$${n.toFixed(2)}`;

  return (
    <div className={styles.overlay} onClick={onClose}>
      <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
        <div className={styles.header}>
          <h2>Sinking Fund — {billName}</h2>
          <button className={styles.closeBtn} onClick={onClose}>
            <X size={20} />
          </button>
        </div>

        <div className={styles.body}>
          <div className={styles.controls}>
            <div className={styles.field}>
              <label>Target pay period (when the bill is due)</label>
              <select
                value={targetPeriodId ?? ''}
                onChange={(e) => {
                  setTargetPeriodId(Number(e.target.value) || null);
                  setPlan(null);
                }}
              >
                <option value="">Select a period...</option>
                {periods.map((p) => (
                  <option key={p.id} value={p.id}>
                    {formatShortDate(p.pay_date)}
                    {p.source_name ? ` — ${p.source_name}` : ''}
                  </option>
                ))}
              </select>
            </div>

            <div className={styles.field}>
              <label>Spread over N periods</label>
              <input
                type="number"
                min="1"
                max="24"
                value={numPeriods}
                onChange={(e) => {
                  setNumPeriods(Number(e.target.value));
                  setPlan(null);
                }}
              />
            </div>

            <button
              className={styles.planBtn}
              onClick={handleGeneratePlan}
              disabled={!targetPeriodId || numPeriods <= 0 || planMutation.isPending}
            >
              {planMutation.isPending ? 'Calculating...' : 'Preview Plan'}
            </button>
          </div>

          {planError && (
            <div className={styles.errorBanner}>
              <AlertTriangle size={16} /> {planError}
            </div>
          )}

          {plan && (
            <div className={styles.planSection}>
              <div className={styles.planSummary}>
                <span>Total needed: <strong>{formatAmount(plan.total_needed)}</strong></span>
                <span>Can fund: <strong>{formatAmount(plan.total_funded)}</strong></span>
              </div>

              {plan.shortfall > 0 && (
                <div className={styles.shortfallBanner}>
                  <AlertTriangle size={16} />
                  Only {formatAmount(plan.total_funded)} of {formatAmount(plan.total_needed)} can be
                  funded — {formatAmount(plan.shortfall)} shortfall due to $50 buffer constraint.
                  You can still apply a partial plan.
                </div>
              )}

              {plan.shortfall === 0 && (
                <div className={styles.fullyCoveredBanner}>
                  <CheckCircle size={16} />
                  Fully funded across {plan.installments.filter((i) => i.amount > 0).length} periods.
                </div>
              )}

              <table className={styles.table}>
                <thead>
                  <tr>
                    <th>Period</th>
                    <th>Surplus Available</th>
                    <th>Installment Reserved</th>
                  </tr>
                </thead>
                <tbody>
                  {plan.installments.map((inst) => (
                    <tr key={inst.period_id} className={inst.amount === 0 ? styles.zeroRow : ''}>
                      <td>{formatShortDate(inst.pay_date)}</td>
                      <td>{formatAmount(inst.surplus)}</td>
                      <td className={inst.amount === 0 ? styles.zeroAmount : styles.amount}>
                        {inst.amount === 0 ? '—' : formatAmount(inst.amount)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>

              <div className={styles.actions}>
                {targetPeriodId && (
                  <button
                    className={styles.clearBtn}
                    onClick={() => clearMutation.mutate()}
                    disabled={clearMutation.isPending}
                  >
                    {clearMutation.isPending ? 'Clearing...' : 'Clear Existing Installments'}
                  </button>
                )}
                <button className={styles.cancelBtn} onClick={onClose}>
                  Cancel
                </button>
                <button
                  className={styles.applyBtn}
                  onClick={() => applyMutation.mutate()}
                  disabled={applyMutation.isPending || plan.installments.every((i) => i.amount === 0)}
                >
                  {applyMutation.isPending ? 'Applying...' : 'Apply Sinking Fund'}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
