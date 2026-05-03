export default function LoadingState({ label = 'Loading...' }: { label?: string }) {
  return (
    <div className="rounded-md border border-slate-200 bg-white px-4 py-6 text-sm text-slate-600">
      {label}
    </div>
  );
}
