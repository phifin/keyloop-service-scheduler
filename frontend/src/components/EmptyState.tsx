export default function EmptyState({ title }: { title: string }) {
  return (
    <div className="rounded-md border border-dashed border-slate-300 bg-slate-50 px-4 py-10 text-center text-sm text-slate-600">
      {title}
    </div>
  );
}
