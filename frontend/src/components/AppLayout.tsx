import { NavLink, Outlet } from 'react-router-dom';

export default function AppLayout() {
  const linkClass = ({ isActive }: { isActive: boolean }) =>
    [
      'rounded-md px-3 py-2 text-sm font-medium transition',
      isActive
        ? 'bg-slate-900 text-white'
        : 'text-slate-600 hover:bg-slate-100 hover:text-slate-950',
    ].join(' ');

  return (
    <div className="min-h-screen bg-slate-50 text-slate-950">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-6xl flex-col gap-3 px-4 py-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p className="text-xs font-semibold uppercase tracking-wider text-sky-700">Keyloop</p>
            <h1 className="text-xl font-semibold text-slate-950">
              Unified Service Scheduler
            </h1>
          </div>
          <nav className="flex gap-2">
            <NavLink to="/book" className={linkClass}>
              Book Appointment
            </NavLink>
            <NavLink to="/appointments" className={linkClass}>
              Appointments
            </NavLink>
          </nav>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-4 py-6">
        <Outlet />
      </main>
    </div>
  );
}
