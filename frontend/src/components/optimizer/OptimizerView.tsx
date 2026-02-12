import { useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { Lightbulb, TrendingUp, Calendar, ArrowRight } from 'lucide-react';
import { useBudgetStore } from '../../stores/budgetStore';
import styles from './OptimizerView.module.css';

interface Suggestion {
  bill_name: string;
  from_period: string;
  to_period: string;
  amount: number;
  reason: string;
}

interface SurplusMonth {
  month: string;
  source: string;
  extra_checks: number;
  surplus_amount: number;
}

interface OptimizationResult {
  suggestions: Suggestion[];
  current_min_balance: number;
  optimized_min_balance: number;
  improvement: number;
}

interface SurplusResult {
  surplus_months: SurplusMonth[];
  annual_surplus: number;
}

export function OptimizerView() {
  const { dateRange } = useBudgetStore();
  const [tab, setTab] = useState<'assign' | 'surplus'>('assign');

  const optimizeMutation = useMutation({
    mutationFn: async () => {
      const res = await fetch('/api/v1/optimizer/suggest', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ from: dateRange.from, to: dateRange.to, strategy: 'maximize_minimum_balance' }),
      });
      if (!res.ok) throw new Error('Optimization failed');
      const json = await res.json();
      return json.data as OptimizationResult;
    },
  });

  const { data: surplusData, isLoading: surplusLoading } = useQuery({
    queryKey: ['surplus', dateRange.from],
    queryFn: async () => {
      const year = new Date(dateRange.from).getFullYear();
      const res = await fetch(`/api/v1/optimizer/surplus?from=${year}-01-01&to=${year}-12-31`);
      if (!res.ok) throw new Error('Failed to load surplus data');
      const json = await res.json();
      return json.data as SurplusResult;
    },
  });

  return (
    <div className={styles.container}>
      <h1>Savings Optimizer</h1>
      <p className={styles.subtitle}>
        Optimize bill assignments across paychecks and find surplus opportunities.
      </p>

      <div className={styles.tabs}>
        <button
          className={`${styles.tab} ${tab === 'assign' ? styles.tabActive : ''}`}
          onClick={() => setTab('assign')}
        >
          <Lightbulb size={16} /> Bill Assignment
        </button>
        <button
          className={`${styles.tab} ${tab === 'surplus' ? styles.tabActive : ''}`}
          onClick={() => setTab('surplus')}
        >
          <Calendar size={16} /> Surplus Detection
        </button>
      </div>

      {tab === 'assign' && (
        <div className={styles.section}>
          <p className={styles.sectionDesc}>
            Analyzes your bill due dates and paycheck schedule to suggest the optimal assignment
            of bills to paychecks, maximizing your minimum balance across all pay periods.
          </p>
          <button
            className={styles.runBtn}
            onClick={() => optimizeMutation.mutate()}
            disabled={optimizeMutation.isPending}
          >
            {optimizeMutation.isPending ? 'Optimizing...' : 'Run Optimizer'}
          </button>

          {optimizeMutation.data && (
            <div className={styles.results}>
              <div className={styles.comparison}>
                <div className={styles.compareCard}>
                  <div className={styles.compareLabel}>Current Min Balance</div>
                  <div className={styles.compareValue}>${optimizeMutation.data.current_min_balance.toFixed(0)}</div>
                </div>
                <ArrowRight size={24} className={styles.arrow} />
                <div className={`${styles.compareCard} ${styles.compareImproved}`}>
                  <div className={styles.compareLabel}>Optimized Min Balance</div>
                  <div className={styles.compareValue}>${optimizeMutation.data.optimized_min_balance.toFixed(0)}</div>
                </div>
              </div>

              {optimizeMutation.data.improvement > 0 && (
                <div className={styles.improvement}>
                  <TrendingUp size={16} />
                  ${optimizeMutation.data.improvement.toFixed(0)} improvement in minimum balance
                </div>
              )}

              {optimizeMutation.data.suggestions.length > 0 ? (
                <div className={styles.suggestionList}>
                  <h3>Suggested Changes</h3>
                  {optimizeMutation.data.suggestions.map((s, i) => (
                    <div key={i} className={styles.suggestion}>
                      <div className={styles.suggestionMain}>
                        <span className={styles.suggestionBill}>{s.bill_name}</span>
                        <span className={styles.suggestionAmount}>${s.amount.toFixed(0)}</span>
                      </div>
                      <div className={styles.suggestionMove}>
                        <span>{s.from_period}</span>
                        <ArrowRight size={14} />
                        <span>{s.to_period}</span>
                      </div>
                      <div className={styles.suggestionReason}>{s.reason}</div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className={styles.noChanges}>
                  Your bill assignments are already optimal for this period.
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {tab === 'surplus' && (
        <div className={styles.section}>
          <p className={styles.sectionDesc}>
            Identifies months where you receive extra paychecks (3-paycheck months for biweekly,
            5-paycheck months for weekly) and calculates potential savings from those surpluses.
          </p>

          {surplusLoading ? (
            <div className={styles.loading}>Loading surplus data...</div>
          ) : surplusData ? (
            <>
              {surplusData.annual_surplus > 0 && (
                <div className={styles.annualSurplus}>
                  <TrendingUp size={20} />
                  <div>
                    <div className={styles.surplusLabel}>Potential Annual Savings from Surpluses</div>
                    <div className={styles.surplusValue}>${surplusData.annual_surplus.toFixed(0)}</div>
                  </div>
                </div>
              )}

              {surplusData.surplus_months.length > 0 ? (
                <div className={styles.surplusList}>
                  {surplusData.surplus_months.map((m, i) => (
                    <div key={i} className={styles.surplusCard}>
                      <div className={styles.surplusMonth}>{m.month}</div>
                      <div className={styles.surplusDetails}>
                        <span>{m.source}: {m.extra_checks} extra check(s)</span>
                        <span className={styles.surplusAmount}>+${m.surplus_amount.toFixed(0)}</span>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className={styles.noChanges}>No surplus months found for this year.</div>
              )}
            </>
          ) : null}
        </div>
      )}
    </div>
  );
}
