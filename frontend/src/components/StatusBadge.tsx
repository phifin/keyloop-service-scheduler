import type { AppointmentStatus } from '../types/api';

const statusClasses: Record<AppointmentStatus, string> = {
  CONFIRMED: 'bg-emerald-50 text-emerald-700 ring-emerald-200',
  CANCELLED: 'bg-slate-100 text-slate-700 ring-slate-300',
  COMPLETED: 'bg-sky-50 text-sky-700 ring-sky-200',
};

export default function StatusBadge({ status }: { status: AppointmentStatus }) {
  return (
    <span
      className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ring-1 ring-inset ${statusClasses[status]}`}
    >
      {status}
    </span>
  );
}
