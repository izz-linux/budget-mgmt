import { useQuery } from '@tanstack/react-query';
import { DollarSign, Receipt, TrendingUp, AlertCircle } from 'lucide-react';
import { gridApi } from '../../api/grid';
import styles from './Dashboard.module.css';

export function Dashboard() {
  const now = new Date();
  const from = new Date(now);
  from.setDate(1);
  const to = new Date(now);
  to.setMonth(to.getMonth() + 2);

  const { data, isLoading } = useQuery({
    queryKey: ['dashboard-grid', from.toISOString().split('T')[0], to.toISOString().split('T')[0]],
    queryFn: () => gridApi.get(from.toISOString().split('T')[0], to.toISOString().split('T')[0]),
  });

  if (isLoading) return <div className={styles.loading}>Loading dashboard...</div>;

  const periods = data?.periods || [];
  const assignments = data?.assignments || {};
  const bills = data?.bills || [];

  // Find upcoming bills (next 7 days)
  const today = new Date();
  const weekFromNow = new Date(today);
  weekFromNow.setDate(weekFromNow.getDate() + 7);

  const upcomingBills = bills
    .filter((bill) => {
      if (!bill.due_day) return false;
      const dueThisMonth = new Date(today.getFullYear(), today.getMonth(), bill.due_day);
      return dueThisMonth >= today && dueThisMonth <= weekFromNow;
    })
    .sort((a, b) => (a.due_day || 0) - (b.due_day || 0));

  // Stats
  const totalIncome = periods.reduce((sum, p) => sum + (p.expected_amount || 0), 0);
  const totalBills = periods.reduce((sum, p) => sum + p.total_bills, 0);
  const totalRemaining = totalIncome - totalBills;

  const paidCount = Object.values(assignments).filter((a) => a.status === 'paid').length;
  const pendingCount = Object.values(assignments).filter((a) => a.status === 'pending').length;

  const tightPeriods = periods.filter((p) => p.remaining < 0);

  return (
    <div className={styles.container}>
      <h1>Dashboard</h1>

      <div className={styles.cards}>
        <div className={styles.card}>
          <div className={styles.cardIcon} style={{ background: 'rgba(99, 102, 241, 0.15)', color: 'var(--color-primary)' }}>
            <DollarSign size={20} />
          </div>
          <div className={styles.cardContent}>
            <div className={styles.cardLabel}>Expected Income</div>
            <div className={styles.cardValue}>${totalIncome.toFixed(0)}</div>
          </div>
        </div>

        <div className={styles.card}>
          <div className={styles.cardIcon} style={{ background: 'rgba(239, 68, 68, 0.15)', color: 'var(--color-danger)' }}>
            <Receipt size={20} />
          </div>
          <div className={styles.cardContent}>
            <div className={styles.cardLabel}>Total Bills</div>
            <div className={styles.cardValue}>${totalBills.toFixed(0)}</div>
          </div>
        </div>

        <div className={styles.card}>
          <div className={styles.cardIcon} style={{ background: 'rgba(34, 197, 94, 0.15)', color: 'var(--color-success)' }}>
            <TrendingUp size={20} />
          </div>
          <div className={styles.cardContent}>
            <div className={styles.cardLabel}>Remaining</div>
            <div className={styles.cardValue} style={{ color: totalRemaining >= 0 ? 'var(--color-success)' : 'var(--color-danger)' }}>
              ${totalRemaining.toFixed(0)}
            </div>
          </div>
        </div>

        <div className={styles.card}>
          <div className={styles.cardIcon} style={{ background: 'rgba(245, 158, 11, 0.15)', color: 'var(--color-warning)' }}>
            <AlertCircle size={20} />
          </div>
          <div className={styles.cardContent}>
            <div className={styles.cardLabel}>Status</div>
            <div className={styles.cardValue}>{paidCount} paid / {pendingCount} pending</div>
          </div>
        </div>
      </div>

      <div className={styles.sections}>
        {/* Upcoming bills */}
        <div className={styles.section}>
          <h2>Due This Week</h2>
          {upcomingBills.length > 0 ? (
            <div className={styles.upcomingList}>
              {upcomingBills.map((bill) => (
                <div key={bill.id} className={styles.upcomingItem}>
                  <span className={styles.upcomingName}>{bill.name}</span>
                  <span className={styles.upcomingDue}>
                    {bill.due_day}th
                    {bill.is_autopay && <span className={styles.autoTag}>Auto</span>}
                  </span>
                  <span className={styles.upcomingAmount}>
                    {bill.default_amount ? `$${bill.default_amount.toFixed(0)}` : '-'}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <div className={styles.emptySection}>No bills due this week</div>
          )}
        </div>

        {/* Tight periods */}
        {tightPeriods.length > 0 && (
          <div className={styles.section}>
            <h2>Overbudget Periods</h2>
            <div className={styles.upcomingList}>
              {tightPeriods.map((period) => (
                <div key={period.id} className={`${styles.upcomingItem} ${styles.warningItem}`}>
                  <span>{new Date(period.pay_date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}</span>
                  <span className={styles.upcomingName}>{period.source_name}</span>
                  <span style={{ color: 'var(--color-danger)', fontWeight: 600 }}>
                    ${period.remaining.toFixed(0)}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Period overview */}
        <div className={styles.section}>
          <h2>Pay Period Overview</h2>
          <div className={styles.periodList}>
            {periods.map((period) => {
              const pct = period.expected_amount
                ? Math.min(100, (period.total_bills / period.expected_amount) * 100)
                : 0;
              return (
                <div key={period.id} className={styles.periodItem}>
                  <div className={styles.periodHeader}>
                    <span>
                      {new Date(period.pay_date).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}
                      {' '}
                      <span className={styles.periodSource}>{period.source_name}</span>
                    </span>
                    <span>${period.remaining.toFixed(0)} left</span>
                  </div>
                  <div className={styles.progressBar}>
                    <div
                      className={styles.progressFill}
                      style={{
                        width: `${pct}%`,
                        background: pct > 100 ? 'var(--color-danger)' : pct > 80 ? 'var(--color-warning)' : 'var(--color-primary)',
                      }}
                    />
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
