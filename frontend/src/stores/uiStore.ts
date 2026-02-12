import { create } from 'zustand';

interface UIState {
  sidebarOpen: boolean;
  selectedPeriodIndex: number;
  isMobile: boolean;
  toggleSidebar: () => void;
  setSelectedPeriodIndex: (index: number) => void;
  setIsMobile: (mobile: boolean) => void;
}

export const useUIStore = create<UIState>((set) => ({
  sidebarOpen: false,
  selectedPeriodIndex: 0,
  isMobile: window.innerWidth < 768,
  toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),
  setSelectedPeriodIndex: (index) => set({ selectedPeriodIndex: index }),
  setIsMobile: (mobile) => set({ isMobile: mobile }),
}));
