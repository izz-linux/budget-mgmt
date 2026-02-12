import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AppShell } from './components/layout/AppShell';
import { Dashboard } from './components/dashboard/Dashboard';
import { BudgetGrid } from './components/budget-grid/BudgetGrid';
import { BillList } from './components/bills/BillList';
import { IncomeList } from './components/income/IncomeList';
import { ImportWizard } from './components/import/ImportWizard';
import { OptimizerView } from './components/optimizer/OptimizerView';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
});

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route element={<AppShell />}>
            <Route path="/" element={<Dashboard />} />
            <Route path="/budget" element={<BudgetGrid />} />
            <Route path="/bills" element={<BillList />} />
            <Route path="/income" element={<IncomeList />} />
            <Route path="/import" element={<ImportWizard />} />
            <Route path="/optimize" element={<OptimizerView />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
