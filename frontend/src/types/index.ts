export interface Bill {
  id: number;
  name: string;
  default_amount: number | null;
  due_day: number | null;
  recurrence: string;
  recurrence_detail?: Record<string, unknown>;
  is_autopay: boolean;
  category: string;
  notes: string;
  is_active: boolean;
  sort_order: number;
  created_at: string;
  updated_at: string;
  credit_card?: CreditCard;
}

export interface CreditCard {
  id: number;
  bill_id: number;
  card_label: string;
  statement_day: number;
  due_day: number;
  issuer: string;
  created_at: string;
}

export interface IncomeSource {
  id: number;
  name: string;
  pay_schedule: 'weekly' | 'biweekly' | 'semimonthly';
  schedule_detail: Record<string, unknown>;
  default_amount: number | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface PayPeriod {
  id: number;
  income_source_id: number;
  pay_date: string;
  expected_amount: number | null;
  actual_amount: number | null;
  notes: string;
  created_at: string;
  source_name?: string;
  total_bills: number;
  remaining: number;
}

export interface BillAssignment {
  id: number;
  bill_id: number;
  pay_period_id: number;
  planned_amount: number | null;
  forecast_amount: number | null;
  actual_amount: number | null;
  status: 'pending' | 'paid' | 'deferred' | 'uncertain' | 'skipped';
  deferred_to_id: number | null;
  is_extra: boolean;
  extra_name: string;
  notes: string;
  created_at: string;
  updated_at: string;
  bill_name?: string;
}

export interface BudgetGridData {
  bills: Bill[];
  periods: PayPeriod[];
  assignments: Record<string, BillAssignment>; // key: "billId-periodId"
}

export interface APIResponse<T> {
  data: T;
  meta?: { timestamp: string };
}

export interface APIError {
  error: {
    code: string;
    message: string;
    details?: unknown;
  };
}
