import { useState } from "react";
import { Link } from "react-router-dom";
import { api, type Report } from "../../lib/api";
import { REPORT_TYPES, ago, fcfa } from "../../lib/format";
import { useLoad } from "../../lib/session";
import { Button, Card, Empty, ErrorText, Input, PageTitle, StatusBadge, Tabs } from "../../components/ui";

function Resolve({ r, onDone }: { r: Report; onDone: (list: Report[]) => void }) {
  const [refund, setRefund] = useState("");
  const [note, setNote] = useState("");
  const [error, setError] = useState<string | null>(null);
  const resolve = async (resolution: string) => {
    setError(null);
    try {
      onDone(await api.admin.resolveReport(r.id, resolution, Number(refund) || 0, note));
    } catch (e) {
      setError((e as Error).message);
    }
  };
  const held = r.held_amount;
  return (
    <div className="mt-3 space-y-2 border-t pt-3">
      <p className="text-sm">{held != null ? `🔒 ${fcfa(held)} bloqués en séquestre` : "Aucun fonds bloqué (paiement déjà versé ou aucun paiement)."}</p>
      <Input placeholder="Note de décision" value={note} onChange={(e) => setNote(e.target.value)} />
      <div className="flex flex-wrap items-center gap-2">
        {held != null && (
          <>
            <Button variant="danger" onClick={() => resolve("refund_client")}>
              Rembourser le client
            </Button>
            <Button onClick={() => resolve("pay_worker")}>Payer le worker</Button>
            <Input className="w-32" type="number" placeholder="Remb. partiel" value={refund} onChange={(e) => setRefund(e.target.value)} />
            <Button variant="secondary" disabled={!refund} onClick={() => resolve("partial")}>
              Remboursement partiel
            </Button>
          </>
        )}
        <Button variant="ghost" onClick={() => resolve("dismiss")}>
          Classer sans suite
        </Button>
      </div>
      <ErrorText error={error} />
    </div>
  );
}

export default function Reports() {
  const [status, setStatus] = useState<"open" | "resolved" | "rejected">("open");
  const reports = useLoad(() => api.admin.reports(status), [status]);
  return (
    <div>
      <PageTitle>Litiges & signalements</PageTitle>
      <Tabs value={status} onChange={setStatus} tabs={[["open", "Ouverts"], ["resolved", "Résolus"], ["rejected", "Classés"]]} />
      <ErrorText error={reports.error} />
      <div className="space-y-2">
        {reports.data?.length ? (
          reports.data.map((r) => (
            <Card key={r.id}>
              <div className="flex flex-wrap items-start justify-between gap-2">
                <div>
                  <p className="font-semibold">
                    {REPORT_TYPES[r.type] ?? r.type} — <Link to={`/jobs/${r.job_id}`} className="text-indigo-700 hover:underline">{r.job_title}</Link>
                  </p>
                  <p className="text-xs text-slate-500">
                    Par {r.reporter_name} ({r.reporter_role}) · {ago(r.created_at)}
                  </p>
                </div>
                <StatusBadge status={r.job_status} />
              </div>
              {r.description && <p className="mt-2 text-sm text-slate-700">{r.description}</p>}
              {r.status === "open" ? (
                <Resolve r={r} onDone={() => reports.reload()} />
              ) : (
                <p className="mt-2 text-sm text-slate-600">
                  Décision : {r.resolution}
                  {r.refund_amount ? ` · ${fcfa(r.refund_amount)} remboursés` : ""}
                  {r.resolution_note && ` · ${r.resolution_note}`}
                </p>
              )}
            </Card>
          ))
        ) : (
          <Empty>Aucun signalement.</Empty>
        )}
      </div>
    </div>
  );
}
