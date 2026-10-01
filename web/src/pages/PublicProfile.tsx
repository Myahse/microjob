import { useParams } from "react-router-dom";
import { api } from "../lib/api";
import { ago, fcfa } from "../lib/format";
import { useLoad } from "../lib/session";
import { Avatar, Card, ErrorText, Stars } from "../components/ui";

export default function PublicProfile() {
  const { id = "" } = useParams();
  const p = useLoad(() => api.profile(id), [id]);
  if (p.error) return <ErrorText error={p.error} />;
  if (!p.data) return <p className="text-slate-500">Chargement…</p>;
  const { user: u, reviews } = p.data;
  const w = u.worker;
  return (
    <div className="mx-auto max-w-2xl space-y-4">
      <Card className="text-center">
        <div className="flex justify-center">
          <Avatar url={u.avatar_url} name={u.first_name} size={88} />
        </div>
        <h1 className="mt-2 text-xl font-bold">
          {u.first_name} {u.last_name}
        </h1>
        {w?.headline && <p className="text-slate-600">{w.headline}</p>}
        {w && (
          <div className="mt-2 flex justify-center gap-4 text-sm">
            <Stars value={w.rating} count={w.rating_count} />
            <span>{w.completed_jobs} jobs réalisés</span>
            <span>{w.completion_rate} % de complétion</span>
          </div>
        )}
        {u.zone && <p className="mt-1 text-sm text-slate-500">📍 {u.zone}</p>}
        {w && <p className="mt-1 text-sm">{w.available ? "🟢 Disponible" : "⚪ Indisponible"}{w.hourly_rate > 0 && ` · ${fcfa(w.hourly_rate)}/h`}</p>}
        <div className="mt-3 flex flex-wrap justify-center gap-2 text-xs">
          {(w?.badges ?? (u.identity_verified ? ["Identité vérifiée"] : [])).map((b) => (
            <span key={b} className="rounded-full bg-indigo-100 px-2 py-0.5 text-indigo-800">
              {b === "Top Worker" ? "🏆 " : "✅ "}
              {b}
            </span>
          ))}
        </div>
        {u.bio && <p className="mt-3 text-slate-700">{u.bio}</p>}
      </Card>
      {w && w.skills.length > 0 && (
        <Card>
          <h3 className="mb-2 font-bold">Compétences</h3>
          <div className="flex flex-wrap gap-1.5">
            {w.skills.map((s) => (
              <span key={s.id} className="rounded-full bg-slate-100 px-3 py-1 text-sm">
                {s.name}
                {s.verified && " ✓"}
              </span>
            ))}
          </div>
        </Card>
      )}
      <Card>
        <h3 className="mb-2 font-bold">Avis ({reviews.length})</h3>
        {reviews.length === 0 && <p className="text-sm text-slate-500">Pas encore d'avis.</p>}
        <ul className="space-y-3">
          {reviews.map((r) => (
            <li key={r.id} className="text-sm">
              <div className="flex justify-between">
                <span>
                  {"⭐".repeat(r.rating)} <span className="font-medium">{r.reviewer_name}</span>
                  <span className="text-slate-400"> · {r.job_title}</span>
                </span>
                <span className="text-xs text-slate-400">{ago(r.created_at)}</span>
              </div>
              {r.comment && <p className="text-slate-700">{r.comment}</p>}
            </li>
          ))}
        </ul>
      </Card>
    </div>
  );
}
