import { useState } from "react";
import { Link } from "react-router-dom";
import { api, type JobStatus } from "../../lib/api";
import { STATUS_LABEL, ago, fcfa } from "../../lib/format";
import { useLoad } from "../../lib/session";
import { Button, Card, ErrorText, Input, PageTitle, Select, StatusBadge } from "../../components/ui";

const CLOSED: JobStatus[] = ["COMPLETED", "PAID", "RATED", "CANCELLED"];

export default function Jobs() {
  const [status, setStatus] = useState("");
  const [q, setQ] = useState("");
  const jobs = useLoad(() => api.admin.jobs(status, q), [status, q]);
  const [error, setError] = useState<string | null>(null);

  const cancel = async (id: string) => {
    const reason = prompt("Raison de la modération ?");
    if (!reason) return;
    try {
      await api.admin.cancelJob(id, reason);
      jobs.reload();
    } catch (e) {
      setError((e as Error).message);
    }
  };

  return (
    <div>
      <PageTitle>Jobs</PageTitle>
      <div className="mb-3 flex gap-2">
        <Input placeholder="Titre ou zone…" value={q} onChange={(e) => setQ(e.target.value)} />
        <Select className="w-auto" value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="">Tous les statuts</option>
          {Object.entries(STATUS_LABEL).map(([k, v]) => (
            <option key={k} value={k}>
              {v}
            </option>
          ))}
        </Select>
      </div>
      <ErrorText error={error ?? jobs.error} />
      <Card className="overflow-x-auto p-0">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 text-left text-slate-500">
            <tr>
              <th className="p-2">Job</th>
              <th>Client</th>
              <th>Worker</th>
              <th>Prix</th>
              <th>Statut</th>
              <th>Créé</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {jobs.data?.map((j) => (
              <tr key={j.id} className="border-t">
                <td className="p-2">
                  <Link to={`/jobs/${j.id}`} className="font-medium hover:underline">
                    {j.category.icon} {j.title}
                  </Link>
                  <span className="block text-xs text-slate-500">📍 {j.zone}</span>
                </td>
                <td>{j.client.first_name}</td>
                <td>{j.worker?.first_name ?? "—"}</td>
                <td>{fcfa(j.agreed_price ?? j.budget)}</td>
                <td>
                  <StatusBadge status={j.status} />
                </td>
                <td className="text-xs text-slate-500">{ago(j.created_at)}</td>
                <td className="pr-2 text-right">
                  {!CLOSED.includes(j.status) && (
                    <Button variant="ghost" onClick={() => cancel(j.id)}>
                      Annuler
                    </Button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
    </div>
  );
}
