import { useEffect, useState } from "react";
import { api } from "../../lib/api";
import { fcfa } from "../../lib/format";
import { useLoad } from "../../lib/session";
import { Button, Card, ErrorText, Field, Input, PageTitle } from "../../components/ui";

function Tile({ label, value, sub }: { label: string; value: string | number; sub?: string }) {
  return (
    <Card>
      <p className="text-xs font-medium uppercase tracking-wide text-slate-500">{label}</p>
      <p className="mt-1 text-2xl font-extrabold text-slate-900">{value}</p>
      {sub && <p className="text-xs text-slate-500">{sub}</p>}
    </Card>
  );
}

function Settings() {
  const s = useLoad(() => api.admin.settings());
  const [form, setForm] = useState<Record<string, string>>({});
  const [msg, setMsg] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    if (s.data) setForm(s.data);
  }, [s.data]);
  const save = async () => {
    setError(null);
    try {
      s.setData(await api.admin.updateSettings(form));
      setMsg("Paramètres enregistrés");
    } catch (e) {
      setError((e as Error).message);
    }
  };
  return (
    <Card className="space-y-3">
      <h3 className="font-bold">Paramètres de la plateforme</h3>
      <div className="grid gap-3 sm:grid-cols-3">
        <Field label="Commission par défaut" hint="0.10 = 10 % (surchargeable par catégorie)">
          <Input value={form.default_commission_rate ?? ""} onChange={(e) => setForm({ ...form, default_commission_rate: e.target.value })} />
        </Field>
        <Field label="Workers notifiés par job">
          <Input value={form.matching_max_notified ?? ""} onChange={(e) => setForm({ ...form, matching_max_notified: e.target.value })} />
        </Field>
        <Field label="Rayon worker par défaut (km)">
          <Input value={form.default_worker_radius_km ?? ""} onChange={(e) => setForm({ ...form, default_worker_radius_km: e.target.value })} />
        </Field>
      </div>
      <ErrorText error={error} />
      {msg && <p className="text-sm text-emerald-700">{msg}</p>}
      <Button onClick={save}>Enregistrer</Button>
    </Card>
  );
}

export default function Dashboard() {
  const stats = useLoad(() => api.admin.stats());
  const s = stats.data;
  const maxPosted = Math.max(1, ...(s?.daily.map((d) => d.posted) ?? [1]));
  return (
    <div className="space-y-4">
      <PageTitle>Statistiques</PageTitle>
      <ErrorText error={stats.error} />
      {s && (
        <>
          <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
            <Tile label="GMV" value={fcfa(s.gmv)} />
            <Tile label="Commission" value={fcfa(s.commission)} />
            <Tile label="Jobs publiés" value={s.jobs_posted} sub={`${s.acceptance_rate} % acceptés`} />
            <Tile label="Jobs complétés" value={s.jobs_completed} sub={`${s.completion_rate} % de complétion`} />
            <Tile label="Workers actifs (30 j)" value={s.active_workers} sub={`${s.available_workers} disponibles maintenant`} />
            <Tile label="Clients actifs (30 j)" value={s.active_clients} sub={`${s.users} comptes`} />
            <Tile
              label="Temps moyen d'attribution"
              value={s.avg_minutes_to_assignment == null ? "—" : `${Math.round(s.avg_minutes_to_assignment)} min`}
            />
            <Tile label="À traiter" value={s.open_reports + s.pending_withdrawals} sub={`${s.open_reports} litiges · ${s.pending_withdrawals} retraits`} />
          </div>
          <Card>
            <h3 className="mb-3 font-bold">Jobs par jour (14 jours)</h3>
            <div className="flex h-40 items-end gap-1.5">
              {s.daily.map((d) => (
                <div key={d.day} className="group flex flex-1 flex-col items-center justify-end" title={`${d.day} : ${d.posted} publiés, ${d.completed} complétés`}>
                  <span className="mb-1 text-[10px] text-slate-500">{d.posted || ""}</span>
                  <div className="w-full rounded-t bg-indigo-500" style={{ height: `${(d.posted / maxPosted) * 100}%`, minHeight: d.posted ? 4 : 1 }} />
                  <span className="mt-1 text-[10px] text-slate-400">{d.day.slice(8)}</span>
                </div>
              ))}
            </div>
          </Card>
          <Card>
            <h3 className="mb-2 font-bold">Par catégorie</h3>
            <table className="w-full text-sm">
              <thead className="text-left text-slate-500">
                <tr>
                  <th className="py-1">Catégorie</th>
                  <th>Jobs</th>
                  <th>Complétés</th>
                </tr>
              </thead>
              <tbody>
                {s.by_category.map((c) => (
                  <tr key={c.name} className="border-t">
                    <td className="py-1.5">
                      {c.icon} {c.name}
                    </td>
                    <td>{c.jobs}</td>
                    <td>{c.done}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Card>
        </>
      )}
      <Settings />
    </div>
  );
}
