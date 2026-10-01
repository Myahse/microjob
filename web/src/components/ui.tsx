import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode, SelectHTMLAttributes, TextareaHTMLAttributes } from "react";
import { Link } from "react-router-dom";
import { mediaUrl, type Job, type JobStatus } from "../lib/api";
import { STATUS_COLOR, STATUS_LABEL, fcfa, km, when } from "../lib/format";

type Variant = "primary" | "secondary" | "danger" | "ghost";
const VARIANTS: Record<Variant, string> = {
  primary: "bg-indigo-600 text-white hover:bg-indigo-700 disabled:bg-indigo-300",
  secondary: "bg-white text-slate-800 border border-slate-300 hover:bg-slate-50 disabled:opacity-50",
  danger: "bg-rose-600 text-white hover:bg-rose-700 disabled:bg-rose-300",
  ghost: "text-slate-600 hover:bg-slate-100",
};

export function Button({ variant = "primary", className = "", ...props }: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant }) {
  return (
    <button
      {...props}
      className={`inline-flex items-center justify-center gap-2 rounded-lg px-4 py-2 text-sm font-semibold transition ${VARIANTS[variant]} ${className}`}
    />
  );
}

/** Titled group for button sets; unlike Field it is not a <label>, so
 * clicking the title doesn't activate the first button inside. */
export function Group({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div role="group" aria-label={label}>
      <span className="mb-1 block text-sm font-medium text-slate-700">{label}</span>
      {children}
    </div>
  );
}

export function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1 block text-sm font-medium text-slate-700">{label}</span>
      {children}
      {hint && <span className="mt-1 block text-xs text-slate-500">{hint}</span>}
    </label>
  );
}

const inputCls =
  "w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-100";

export const Input = (p: InputHTMLAttributes<HTMLInputElement>) => <input {...p} className={`${inputCls} ${p.className ?? ""}`} />;
export const Textarea = (p: TextareaHTMLAttributes<HTMLTextAreaElement>) => <textarea {...p} className={`${inputCls} ${p.className ?? ""}`} />;
export const Select = (p: SelectHTMLAttributes<HTMLSelectElement>) => <select {...p} className={`${inputCls} ${p.className ?? ""}`} />;

export function Card({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <div className={`rounded-xl border border-slate-200 bg-white p-4 shadow-sm ${className}`}>{children}</div>;
}

export function StatusBadge({ status }: { status: JobStatus }) {
  return <span className={`rounded-full px-2.5 py-0.5 text-xs font-semibold ${STATUS_COLOR[status]}`}>{STATUS_LABEL[status]}</span>;
}

export function Stars({ value, count }: { value: number; count?: number }) {
  if (count === 0) return <span className="text-xs text-slate-400">Nouveau</span>;
  return (
    <span className="text-sm text-amber-600">
      ⭐ {value.toFixed(1)}
      {count !== undefined && <span className="text-slate-400"> ({count})</span>}
    </span>
  );
}

export function Avatar({ url, name, size = 40 }: { url?: string; name: string; size?: number }) {
  if (url) return <img src={mediaUrl(url)} alt={name} style={{ width: size, height: size }} className="rounded-full object-cover" />;
  return (
    <span style={{ width: size, height: size }} className="inline-flex items-center justify-center rounded-full bg-indigo-100 font-bold text-indigo-700">
      {name.charAt(0).toUpperCase()}
    </span>
  );
}

export function ErrorText({ error }: { error: string | null | undefined }) {
  if (!error) return null;
  return <p className="rounded-lg bg-rose-50 px-3 py-2 text-sm text-rose-700">{error}</p>;
}

export function Empty({ children }: { children: ReactNode }) {
  return <p className="rounded-xl border border-dashed border-slate-300 p-8 text-center text-sm text-slate-500">{children}</p>;
}

export function Tabs<T extends string>({ value, onChange, tabs }: { value: T; onChange: (v: T) => void; tabs: [T, string][] }) {
  return (
    <div className="mb-4 flex gap-1 rounded-lg bg-slate-100 p-1">
      {tabs.map(([k, label]) => (
        <button
          key={k}
          onClick={() => onChange(k)}
          className={`flex-1 rounded-md px-3 py-1.5 text-sm font-medium ${value === k ? "bg-white shadow-sm" : "text-slate-600"}`}
        >
          {label}
        </button>
      ))}
    </div>
  );
}

export function JobCard({ job }: { job: Job }) {
  const price = job.agreed_price ?? job.budget;
  return (
    <Link to={`/jobs/${job.id}`} className="block">
      <Card className="transition hover:border-indigo-300 hover:shadow">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-xs text-slate-500">
              {job.category.icon} {job.category.name}
              {job.skill_name ? ` · ${job.skill_name}` : ""}
            </p>
            <h3 className="truncate font-semibold text-slate-900">{job.title}</h3>
          </div>
          <StatusBadge status={job.status} />
        </div>
        <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-sm text-slate-600">
          <span>📍 {job.zone}</span>
          <span>💰 {job.pricing_mode === "offers" && !job.agreed_price ? "Proposez votre prix" : fcfa(price)}</span>
          {job.distance_km != null && <span>📏 {km(job.distance_km)}</span>}
          <span>⏱ {when(job.scheduled_at)}</span>
          {job.my_role === "client" && job.pricing_mode === "offers" && job.applications_count > 0 && (
            <span className="font-medium text-indigo-700">{job.applications_count} offre(s)</span>
          )}
        </div>
      </Card>
    </Link>
  );
}

export function PageTitle({ children, action }: { children: ReactNode; action?: ReactNode }) {
  return (
    <div className="mb-4 flex items-center justify-between gap-3">
      <h1 className="text-xl font-bold text-slate-900">{children}</h1>
      {action}
    </div>
  );
}
