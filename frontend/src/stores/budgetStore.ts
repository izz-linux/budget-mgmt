import { create } from 'zustand';

interface DateRange {
  from: string;
  to: string;
}

interface BudgetState {
  dateRange: DateRange;
  setDateRange: (range: DateRange) => void;
}

function getDefaultRange(): DateRange {
  const now = new Date();
  const from = new Date(now);
  from.setDate(1); // start of month
  const to = new Date(now);
  to.setMonth(to.getMonth() + 3);
  return {
    from: from.toISOString().split('T')[0],
    to: to.toISOString().split('T')[0],
  };
}

export const useBudgetStore = create<BudgetState>((set) => ({
  dateRange: getDefaultRange(),
  setDateRange: (range) => set({ dateRange: range }),
}));
