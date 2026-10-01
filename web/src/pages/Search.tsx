import { useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { api } from "../lib/api";
import { fcfa, km } from "../lib/format";
import { useLoad, useSession } from "../lib/session";
import { Avatar, Button, Card, Empty, ErrorText, Input, PageTitle, Select, Stars } from "../components/ui";

export default function Search() {
  const { user } = useSession();
  const cats = useLoad(() => api.categories());
  const [q, setQ] = useState("");
  const [cat, setCat] = useState(0);
  const [query, setQuery] = useState({ q: "", cat: 0 });
  const results = useLoad(() => api.searchWorkers(query.q, user?.latitude, user?.longitude, query.cat), [query]);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    setQuery({ q, cat });
  };

  return (
    <div className="mx-auto max-w-3xl">
      <PageTitle>Trouver un prestataire</PageTitle>
      <form onSubmit={submit} className="mb-4 flex flex-wrap gap-2">
        <Input className="min-w-48 flex-1" placeholder="Je cherche un plombier, technicien informatique…" value={q} onChange={(e) => setQ(e.target.value)} />
        <Select className="w-auto" value={cat} onChange={(e) => setCat(Number(e.target.value))}>
          <option value={0}>Toutes catégories</option>
          {cats.data?.map((c) => (
            <option key={c.id} value={c.id}>
              {c.icon} {c.name}
            </option>
          ))}
        </Select>
        <Button>Rechercher</Button>
      </form>
      <ErrorText error={results.error} />
      <div className="space-y-2">
        {results.data?.length
          ? results.data.map((w) => (
              <Card key={w.id} className="flex items-center gap-3">
                <Avatar url={w.avatar_url} name={w.first_name} size={48} />
                <div className="min-w-0 flex-1">
                  <p className="font-semibold">
                    {w.first_name} {w.last_name} <span className="text-xs">{w.available ? "🟢" : "⚪"}</span>
                  </p>
                  <p className="text-sm text-slate-600">{w.headline}</p>
                  <div className="flex flex-wrap gap-x-3 text-xs text-slate-500">
                    <Stars value={w.rating} count={w.rating_count} />
                    {w.distance_km != null && <span>📍 {km(w.distance_km)}</span>}
                    {w.hourly_rate > 0 && <span>{fcfa(w.hourly_rate)}/h</span>}
                    <span>{w.completed_jobs} jobs</span>
                  </div>
                  <p className="truncate text-xs text-slate-400">{w.skills.join(" · ")}</p>
                </div>
                <Link to={`/users/${w.id}`} className="rounded-lg border border-indigo-600 px-3 py-1.5 text-sm font-semibold text-indigo-700">
                  CONTACT
                </Link>
              </Card>
            ))
          : !results.loading && <Empty>Aucun prestataire trouvé.</Empty>}
      </div>
    </div>
  );
}
