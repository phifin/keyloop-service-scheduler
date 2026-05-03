import type { ReactNode } from 'react';

type AlertTone = 'error' | 'success' | 'info';

const toneClasses: Record<AlertTone, string> = {
  error: 'border-red-200 bg-red-50 text-red-800',
  success: 'border-emerald-200 bg-emerald-50 text-emerald-800',
  info: 'border-sky-200 bg-sky-50 text-sky-800',
};

export default function Alert({
  tone = 'info',
  children,
}: {
  tone?: AlertTone;
  children: ReactNode;
}) {
  return <div className={`rounded-md border px-4 py-3 text-sm ${toneClasses[tone]}`}>{children}</div>;
}
