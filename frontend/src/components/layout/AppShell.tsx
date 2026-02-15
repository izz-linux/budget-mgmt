import { useEffect } from 'react';
import { Outlet, NavLink, useNavigate } from 'react-router-dom';
import { LayoutDashboard, Grid3X3, Receipt, DollarSign, Upload, Lightbulb, Menu, X, LogOut } from 'lucide-react';
import { useUIStore } from '../../stores/uiStore';
import { useAuthStore } from '../../stores/authStore';
import styles from './AppShell.module.css';

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/budget', icon: Grid3X3, label: 'Budget' },
  { to: '/bills', icon: Receipt, label: 'Bills' },
  { to: '/income', icon: DollarSign, label: 'Income' },
  { to: '/import', icon: Upload, label: 'Import' },
  { to: '/optimize', icon: Lightbulb, label: 'Optimize' },
];

export function AppShell() {
  const { sidebarOpen, toggleSidebar, isMobile, setIsMobile } = useUIStore();
  const { authRequired, logout } = useAuthStore();
  const navigate = useNavigate();

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  useEffect(() => {
    const handleResize = () => setIsMobile(window.innerWidth < 768);
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, [setIsMobile]);

  return (
    <div className={styles.shell}>
      {/* Desktop sidebar */}
      {!isMobile && (
        <nav className={styles.sidebar}>
          <div className={styles.logo}>Budget</div>
          {navItems.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/'}
              className={({ isActive }) =>
                `${styles.navItem} ${isActive ? styles.navItemActive : ''}`
              }
            >
              <Icon size={18} />
              <span>{label}</span>
            </NavLink>
          ))}
          {authRequired && (
            <button className={`${styles.navItem} ${styles.logoutBtn}`} onClick={handleLogout}>
              <LogOut size={18} />
              <span>Logout</span>
            </button>
          )}
        </nav>
      )}

      {/* Mobile header */}
      {isMobile && (
        <header className={styles.mobileHeader}>
          <button className={styles.menuBtn} onClick={toggleSidebar}>
            {sidebarOpen ? <X size={20} /> : <Menu size={20} />}
          </button>
          <span className={styles.mobileTitle}>Budget</span>
        </header>
      )}

      {/* Mobile slide-over nav */}
      {isMobile && sidebarOpen && (
        <>
          <div className={styles.overlay} onClick={toggleSidebar} />
          <nav className={styles.mobileNav}>
            {navItems.map(({ to, icon: Icon, label }) => (
              <NavLink
                key={to}
                to={to}
                end={to === '/'}
                className={({ isActive }) =>
                  `${styles.navItem} ${isActive ? styles.navItemActive : ''}`
                }
                onClick={toggleSidebar}
              >
                <Icon size={18} />
                <span>{label}</span>
              </NavLink>
            ))}
            {authRequired && (
              <button className={`${styles.navItem} ${styles.logoutBtn}`} onClick={() => { toggleSidebar(); handleLogout(); }}>
                <LogOut size={18} />
                <span>Logout</span>
              </button>
            )}
          </nav>
        </>
      )}

      <main className={styles.main}>
        <Outlet />
      </main>

      {/* Mobile bottom tabs */}
      {isMobile && (
        <nav className={styles.bottomNav}>
          {navItems.slice(0, 4).map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/'}
              className={({ isActive }) =>
                `${styles.bottomNavItem} ${isActive ? styles.bottomNavActive : ''}`
              }
            >
              <Icon size={20} />
              <span>{label}</span>
            </NavLink>
          ))}
        </nav>
      )}
    </div>
  );
}
