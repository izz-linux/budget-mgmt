import { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ChevronLeft, ChevronRight, Calendar, RefreshCw, Trash2 } from 'lucide-react';
import { gridApi } from '../../api/grid';
import { assignmentsApi } from '../../api/assignments';
import { periodsApi } from '../../api/periods';
import { useBudgetStore } from '../../stores/budgetStore';
import { useUIStore } from '../../stores/uiStore';
import type { Bill, PayPeriod, BillAssignment } from '../../types';
import { ordinal } from '../../utils/ordinal';
import styles from './BudgetGrid.module.css';

const STATUS_COLORS: Record<string, string> = {
  paid: 'var(--color-success)',
  deferred: 'var(--color-warning)',
  uncertain: 'var(--color-info)',
  pending: 'var(--color-text-muted)',
  skipped: 'var(--color-text-muted)',
};

const STATUS_LABELS: Record<string, string> = {
  paid: 'Paid',
  deferred: 'Deferred',
  uncertain: '??',
  pending: '-',
  skipped: 'Skip',
};

export function BudgetGrid() {
  const queryClient = useQueryClient();
  const { dateRange, setDateRange } = useBudgetStore();
  const { isMobile, selectedPeriodIndex, setSelectedPeriodIndex } = useUIStore();
  const gridRef = useRef<HTMLDivElement>(null);
  const [editingCell, setEditingCell] = useState<string | null>(null);
  const [editValue, setEditValue] = useState('');
  const [editingPeriod, setEditingPeriod] = useState<number | null>(null);
  const [periodEditValue, setPeriodEditValue] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['budget-grid', dateRange.from, dateRange.to],
    queryFn: () => gridApi.get(dateRange.from, dateRange.to),
  });

  const updateAssignment = useMutation({
    mutationFn: ({ id, ...updates }: { id: number } & Partial<BillAssignment>) =>
      assignmentsApi.update(id, updates),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['budget-grid'], exact: false }),
  });

  const createAssignment = useMutation({
    mutationFn: (data: Partial<BillAssignment>) => assignmentsApi.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['budget-grid'], exact: false }),
  });

  const statusCycle = useMutation({
    mutationFn: ({ id, status }: { id: number; status: string }) =>
      assignmentsApi.updateStatus(id, status),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['budget-grid'], exact: false }),
  });

  const deleteAssignment = useMutation({
    mutationFn: (id: number) => assignmentsApi.delete(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['budget-grid'], exact: false }),
  });

  const updatePeriod = useMutation({
    mutationFn: ({ id, expected_amount }: { id: number; expected_amount: number }) =>
      periodsApi.update(id, { expected_amount }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['budget-grid'], exact: false }),
  });

  const generatePeriods = useMutation({
    mutationFn: async () => {
      await periodsApi.generate(dateRange.from, dateRange.to);
      await assignmentsApi.autoAssign(dateRange.from, dateRange.to);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['budget-grid'], exact: false }),
  });

  const bills = data?.bills || [];
  const periods = data?.periods || [];
  const assignments = data?.assignments || {};

  // Auto-assign bills with due dates whenever grid loads with periods
  const autoAssignRanRef = useRef('');
  const rangeKey = useMemo(() => `${dateRange.from}-${dateRange.to}`, [dateRange.from, dateRange.to]);
  useEffect(() => {
    if (periods.length > 0 && autoAssignRanRef.current !== rangeKey) {
      autoAssignRanRef.current = rangeKey;
      assignmentsApi.autoAssign(dateRange.from, dateRange.to).then(() => {
        queryClient.invalidateQueries({ queryKey: ['budget-grid'], exact: false });
      }).catch(() => { /* best-effort */ });
    }
  }, [periods.length, rangeKey, dateRange.from, dateRange.to, queryClient]);

  const shiftRange = useCallback((months: number) => {
    const from = new Date(dateRange.from);
    const to = new Date(dateRange.to);
    from.setMonth(from.getMonth() + months);
    to.setMonth(to.getMonth() + months);
    setDateRange({
      from: from.toISOString().split('T')[0],
      to: to.toISOString().split('T')[0],
    });
  }, [dateRange, setDateRange]);

  // Handle touch swipe for mobile
  const touchStart = useRef(0);
  const handleTouchStart = (e: React.TouchEvent) => {
    touchStart.current = e.touches[0].clientX;
  };
  const handleTouchEnd = (e: React.TouchEvent) => {
    const diff = touchStart.current - e.changedTouches[0].clientX;
    if (Math.abs(diff) > 50) {
      if (diff > 0 && selectedPeriodIndex < periods.length - 1) {
        setSelectedPeriodIndex(selectedPeriodIndex + 1);
      } else if (diff < 0 && selectedPeriodIndex > 0) {
        setSelectedPeriodIndex(selectedPeriodIndex - 1);
      }
    }
  };

  // Reset period index when periods change
  useEffect(() => {
    if (selectedPeriodIndex >= periods.length) {
      setSelectedPeriodIndex(Math.max(0, periods.length - 1));
    }
  }, [periods.length, selectedPeriodIndex, setSelectedPeriodIndex]);

  const getAssignment = (billId: number, periodId: number) =>
    assignments[`${billId}-${periodId}`];

  const handleCellClick = (bill: Bill, period: PayPeriod) => {
    const key = `${bill.id}-${period.id}`;
    const existing = assignments[key];
    if (existing) {
      setEditingCell(key);
      setEditValue(String(existing.planned_amount ?? ''));
    } else {
      setEditingCell(key);
      setEditValue(String(bill.default_amount ?? ''));
    }
  };

  const handleCellSave = (billId: number, periodId: number) => {
    const key = `${billId}-${periodId}`;
    const existing = assignments[key];
    const amount = editValue ? Number(editValue) : null;

    if (existing) {
      updateAssignment.mutate({ id: existing.id, planned_amount: amount });
    } else if (amount) {
      createAssignment.mutate({
        bill_id: billId,
        pay_period_id: periodId,
        planned_amount: amount,
        status: 'pending',
      });
    }
    setEditingCell(null);
  };

  const handleStatusToggle = (assignment: BillAssignment) => {
    const cycle = ['pending', 'paid', 'deferred', 'uncertain'];
    const nextIdx = (cycle.indexOf(assignment.status) + 1) % cycle.length;
    statusCycle.mutate({ id: assignment.id, status: cycle[nextIdx] });
  };

  const handlePeriodAmountClick = (period: PayPeriod) => {
    setEditingPeriod(period.id);
    setPeriodEditValue(String(period.expected_amount ?? ''));
  };

  const handlePeriodAmountSave = (periodId: number) => {
    const amount = periodEditValue ? Number(periodEditValue) : null;
    if (amount != null) {
      updatePeriod.mutate({ id: periodId, expected_amount: amount });
    }
    setEditingPeriod(null);
  };

  const handleDeleteAssignment = (assignment: BillAssignment) => {
    deleteAssignment.mutate(assignment.id);
  };

  const formatDate = (dateStr: string) => {
    const d = new Date(dateStr);
    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  };

  const formatAmount = (amount: number | null | undefined) => {
    if (amount == null) return '';
    return `$${amount.toFixed(0)}`;
  };

  if (isLoading) return <div className={styles.loading}>Loading budget...</div>;

  // Mobile: show single period
  if (isMobile) {
    const period = periods[selectedPeriodIndex];
    if (!period) {
      return (
        <div className={styles.empty}>
          No pay periods found.
          <button
            className={styles.generateBtn}
            onClick={() => generatePeriods.mutate()}
            disabled={generatePeriods.isPending}
          >
            <RefreshCw size={16} /> {generatePeriods.isPending ? 'Generating...' : 'Generate Periods'}
          </button>
        </div>
      );
    }

    return (
      <div className={styles.mobileContainer}
        onTouchStart={handleTouchStart}
        onTouchEnd={handleTouchEnd}
      >
        <div className={styles.mobileHeader}>
          <button className={styles.navBtn} onClick={() => setSelectedPeriodIndex(Math.max(0, selectedPeriodIndex - 1))}
            disabled={selectedPeriodIndex === 0}>
            <ChevronLeft size={20} />
          </button>
          <div className={styles.periodInfo}>
            <div className={styles.periodDate}>{formatDate(period.pay_date)}</div>
            <div className={styles.periodSource}>{period.source_name}</div>
            <div
              className={styles.periodAmount}
              onClick={() => handlePeriodAmountClick(period)}
              style={{ cursor: 'pointer' }}
            >
              {formatAmount(period.expected_amount)} income
            </div>
          </div>
          <button className={styles.navBtn} onClick={() => setSelectedPeriodIndex(Math.min(periods.length - 1, selectedPeriodIndex + 1))}
            disabled={selectedPeriodIndex === periods.length - 1}>
            <ChevronRight size={20} />
          </button>
        </div>

        <div className={styles.periodDots}>
          {periods.map((_, i) => (
            <button
              key={i}
              className={`${styles.dot} ${i === selectedPeriodIndex ? styles.dotActive : ''}`}
              onClick={() => setSelectedPeriodIndex(i)}
            />
          ))}
        </div>

        <div className={styles.mobileBills}>
          {bills.map((bill) => {
            const assignment = getAssignment(bill.id, period.id);
            return (
              <div key={bill.id} className={styles.mobileBillRow}>
                <div className={styles.mobileBillName}>
                  <span>{bill.name}</span>
                  {bill.due_day && <span className={styles.dueTag}>Due {ordinal(bill.due_day)}</span>}
                </div>
                <div className={styles.mobileBillRight}>
                  {assignment ? (
                    <>
                      <span className={styles.amount}>{formatAmount(assignment.planned_amount)}</span>
                      <button
                        className={styles.statusBtn}
                        style={{ color: STATUS_COLORS[assignment.status] }}
                        onClick={() => handleStatusToggle(assignment)}
                      >
                        {STATUS_LABELS[assignment.status]}
                      </button>
                      <button
                        className={styles.removeBtn}
                        onClick={() => handleDeleteAssignment(assignment)}
                        title="Remove"
                      >
                        <Trash2 size={14} />
                      </button>
                    </>
                  ) : (
                    <button
                      className={styles.assignBtn}
                      onClick={() => handleCellClick(bill, period)}
                    >
                      +
                    </button>
                  )}
                </div>
              </div>
            );
          })}
        </div>

        <div className={styles.mobileSummary}>
          <div className={styles.summaryRow}>
            <span>Total Bills</span>
            <span>{formatAmount(period.total_bills)}</span>
          </div>
          <div className={`${styles.summaryRow} ${styles.summaryRemaining}`}>
            <span>Remaining</span>
            <span style={{ color: period.remaining >= 0 ? 'var(--color-success)' : 'var(--color-danger)' }}>
              {formatAmount(period.remaining)}
            </span>
          </div>
        </div>

        {editingPeriod && (() => {
          const p = periods.find(pp => pp.id === editingPeriod);
          if (!p) return null;
          return (
            <div className={styles.editOverlay} onClick={() => setEditingPeriod(null)}>
              <div className={styles.editModal} onClick={(e) => e.stopPropagation()}>
                <label>Income Amount for {formatDate(p.pay_date)}</label>
                <input
                  type="number"
                  step="0.01"
                  value={periodEditValue}
                  onChange={(e) => setPeriodEditValue(e.target.value)}
                  autoFocus
                  onKeyDown={(e) => { if (e.key === 'Enter') handlePeriodAmountSave(p.id); }}
                />
                <div className={styles.editActions}>
                  <button onClick={() => setEditingPeriod(null)}>Cancel</button>
                  <button className={styles.saveBtn} onClick={() => handlePeriodAmountSave(p.id)}>Save</button>
                </div>
              </div>
            </div>
          );
        })()}

        {editingCell && (() => {
          const [billId, periodId] = editingCell.split('-').map(Number);
          return (
            <div className={styles.editOverlay} onClick={() => setEditingCell(null)}>
              <div className={styles.editModal} onClick={(e) => e.stopPropagation()}>
                <label>Amount</label>
                <input
                  type="number"
                  step="0.01"
                  value={editValue}
                  onChange={(e) => setEditValue(e.target.value)}
                  autoFocus
                  onKeyDown={(e) => { if (e.key === 'Enter') handleCellSave(billId, periodId); }}
                />
                <div className={styles.editActions}>
                  <button onClick={() => setEditingCell(null)}>Cancel</button>
                  <button className={styles.saveBtn} onClick={() => handleCellSave(billId, periodId)}>Save</button>
                </div>
              </div>
            </div>
          );
        })()}
      </div>
    );
  }

  // Desktop: full grid view
  return (
    <div className={styles.container}>
      <div className={styles.toolbar}>
        <h1>Budget Grid</h1>
        <div className={styles.dateNav}>
          <button className={styles.navBtn} onClick={() => shiftRange(-1)}>
            <ChevronLeft size={18} />
          </button>
          <span className={styles.dateLabel}>
            <Calendar size={14} />
            {new Date(dateRange.from).toLocaleDateString('en-US', { month: 'short', year: 'numeric' })}
            {' - '}
            {new Date(dateRange.to).toLocaleDateString('en-US', { month: 'short', year: 'numeric' })}
          </span>
          <button className={styles.navBtn} onClick={() => shiftRange(1)}>
            <ChevronRight size={18} />
          </button>
        </div>
      </div>

      {periods.length === 0 ? (
        <div className={styles.empty}>
          No pay periods in this range.
          <button
            className={styles.generateBtn}
            onClick={() => generatePeriods.mutate()}
            disabled={generatePeriods.isPending}
          >
            <RefreshCw size={16} /> {generatePeriods.isPending ? 'Generating...' : 'Generate Periods'}
          </button>
        </div>
      ) : (
        <div className={styles.gridWrapper} ref={gridRef}>
          <table className={styles.grid}>
            <thead>
              <tr>
                <th className={styles.stickyCol}>Bill</th>
                {periods.map((period) => (
                  <th key={period.id} className={styles.periodHeader}>
                    <div className={styles.headerDate}>{formatDate(period.pay_date)}</div>
                    <div className={styles.headerSource}>{period.source_name}</div>
                    {editingPeriod === period.id ? (
                      <input
                        className={styles.periodInput}
                        type="number"
                        step="0.01"
                        value={periodEditValue}
                        onChange={(e) => setPeriodEditValue(e.target.value)}
                        onBlur={() => handlePeriodAmountSave(period.id)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') handlePeriodAmountSave(period.id);
                          if (e.key === 'Escape') setEditingPeriod(null);
                        }}
                        autoFocus
                      />
                    ) : (
                      <div
                        className={styles.headerAmount}
                        onClick={() => handlePeriodAmountClick(period)}
                        title="Click to adjust income for this period"
                        style={{ cursor: 'pointer' }}
                      >
                        {formatAmount(period.expected_amount)}
                      </div>
                    )}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {bills.map((bill) => (
                <tr key={bill.id}>
                  <td className={styles.stickyCol}>
                    <div className={styles.billLabel}>
                      <span className={styles.billName}>{bill.name}</span>
                      <span className={styles.billMeta}>
                        {bill.due_day && ordinal(bill.due_day)}
                        {bill.is_autopay && ' Auto'}
                      </span>
                    </div>
                  </td>
                  {periods.map((period) => {
                    const key = `${bill.id}-${period.id}`;
                    const assignment = assignments[key];
                    const isEditing = editingCell === key;

                    return (
                      <td key={period.id} className={styles.cell}>
                        {isEditing ? (
                          <input
                            className={styles.cellInput}
                            type="number"
                            step="0.01"
                            value={editValue}
                            onChange={(e) => setEditValue(e.target.value)}
                            onBlur={() => handleCellSave(bill.id, period.id)}
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') handleCellSave(bill.id, period.id);
                              if (e.key === 'Escape') setEditingCell(null);
                            }}
                            autoFocus
                          />
                        ) : assignment ? (
                          <div className={styles.cellContent} onClick={() => handleCellClick(bill, period)}>
                            <span className={styles.cellAmount}>{formatAmount(assignment.planned_amount)}</span>
                            <div className={styles.cellActions}>
                              <button
                                className={styles.cellStatus}
                                style={{ color: STATUS_COLORS[assignment.status] }}
                                onClick={(e) => { e.stopPropagation(); handleStatusToggle(assignment); }}
                                title={`Click to change status (${assignment.status})`}
                              >
                                {STATUS_LABELS[assignment.status]}
                              </button>
                              <button
                                className={styles.cellRemove}
                                onClick={(e) => { e.stopPropagation(); handleDeleteAssignment(assignment); }}
                                title="Remove assignment"
                              >
                                <Trash2 size={10} />
                              </button>
                            </div>
                          </div>
                        ) : (
                          <div
                            className={styles.cellEmpty}
                            onClick={() => handleCellClick(bill, period)}
                          />
                        )}
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr className={styles.totalRow}>
                <td className={styles.stickyCol}>Total</td>
                {periods.map((period) => (
                  <td key={period.id} className={styles.totalCell}>
                    {formatAmount(period.total_bills)}
                  </td>
                ))}
              </tr>
              <tr className={styles.remainingRow}>
                <td className={styles.stickyCol}>Remaining</td>
                {periods.map((period) => (
                  <td
                    key={period.id}
                    className={styles.remainingCell}
                    style={{ color: period.remaining >= 0 ? 'var(--color-success)' : 'var(--color-danger)' }}
                  >
                    {formatAmount(period.remaining)}
                  </td>
                ))}
              </tr>
            </tfoot>
          </table>
        </div>
      )}
    </div>
  );
}
