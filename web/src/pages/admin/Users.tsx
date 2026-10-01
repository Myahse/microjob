import { useState } from "react";
import { Link } from "react-router-dom";
import { api, type User } from "../../lib/api";
import { useLoad } from "../../lib/session";
import { Avatar, Button, Card, ErrorText, Input, PageTitle, Select, Stars } from "../../components/ui";

export default function Users() {
  const [q, setQ] = useState("");
  const [role, setRole] = useState("");
  const users = useLoad(() => api.admin.users(q, role), [q, role]);
  const [error, setError] = useState<string | null>(null);
  const [open, setOpen] = useState<string | null>(null);

  const update = async (u: User, body: Record<string, unknown>) => {
    setError(null);
    try {
      const next = await api.admin.updateUser(u.id, body);
      users.setData((list) => list?.map((x) => (x.id === u.id ? next : x)) ?? null);
    } catch (e) {
      setError((e as Error).message);
    }
  };
  const verifySkill = async (u: User, skillId: number, verified: boolean) => {
    try {
      const next = await api.admin.verifySkill(u.id, skillId, verified);
      users.setData((list) => list?.map((x) => (x.id === u.id ? next : x)) ?? null);
    } catch (e) {
      setError((e as Error).message);
    }
  };

  return (
    <div>
      <PageTitle>Utilisateurs</PageTitle>
      <div className="mb-3 flex gap-2">
        <Input placeholder="Nom, email, téléphone…" value={q} onChange={(e) => setQ(e.target.value)} />
        <Select className="w-auto" value={role} onChange={(e) => setRole(e.target.value)}>
          <option value="">Tous</option>
          <option value="worker">Workers</option>
          <option value="client">Clients</option>
          <option value="admin">Admins</option>
          <option value="suspended">Suspendus / bannis</option>
        </Select>
      </div>
      <ErrorText error={error ?? users.error} />
      <div className="space-y-2">
        {users.data?.map((u) => (
          <Card key={u.id}>
            <div className="flex flex-wrap items-center gap-3">
              <Avatar url={u.avatar_url} name={u.first_name} />
              <div className="min-w-0 flex-1">
                <Link to={`/users/${u.id}`} className="font-semibold hover:underline">
                  {u.first_name} {u.last_name}
                </Link>{" "}
                <span className="text-sm text-slate-500">@{u.username}</span>
                <p className="text-xs text-slate-500">
                  {u.email} · {u.phone} · {[u.is_client && "client", u.is_worker && "worker", u.is_admin && "admin"].filter(Boolean).join(", ")}
                </p>
                {u.worker && (
                  <p className="text-xs">
                    <Stars value={u.worker.rating} count={u.worker.rating_count} /> · {u.worker.completed_jobs} jobs · {u.worker.completion_rate} %
                  </p>
                )}
              </div>
              <span
                className={`rounded-full px-2 py-0.5 text-xs font-semibold ${
                  u.status === "active" ? "bg-emerald-100 text-emerald-800" : "bg-rose-100 text-rose-800"
                }`}
              >
                {u.status}
              </span>
              <div className="flex flex-wrap gap-1">
                <Button variant="secondary" onClick={() => update(u, { identity_verified: !u.identity_verified })}>
                  {u.identity_verified ? "Retirer vérif." : "✅ Vérifier identité"}
                </Button>
                {u.status === "active" ? (
                  <>
                    <Button variant="secondary" onClick={() => update(u, { status: "suspended" })}>
                      Suspendre
                    </Button>
                    <Button variant="danger" onClick={() => confirm(`Bannir ${u.first_name} ?`) && update(u, { status: "banned" })}>
                      Bannir
                    </Button>
                  </>
                ) : (
                  <Button onClick={() => update(u, { status: "active" })}>Réactiver</Button>
                )}
                {u.worker && (
                  <Button variant="ghost" onClick={() => setOpen(open === u.id ? null : u.id)}>
                    Compétences
                  </Button>
                )}
              </div>
            </div>
            {open === u.id && u.worker && (
              <div className="mt-3 flex flex-wrap gap-1.5 border-t pt-3">
                {u.worker.skills.length === 0 && <span className="text-sm text-slate-500">Aucune compétence.</span>}
                {u.worker.skills.map((s) => (
                  <button
                    key={s.id}
                    onClick={() => verifySkill(u, s.id, !s.verified)}
                    className={`rounded-full border px-3 py-1 text-sm ${s.verified ? "border-emerald-600 bg-emerald-50 text-emerald-800" : "border-slate-300"}`}
                    title="Cliquer pour (dé)valider le justificatif"
                  >
                    {s.verified ? "✓ " : ""}
                    {s.name}
                  </button>
                ))}
              </div>
            )}
          </Card>
        ))}
      </div>
    </div>
  );
}
