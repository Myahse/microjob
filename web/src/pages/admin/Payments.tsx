import { useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../../lib/api";
import { ago, fcfa } from "../../lib/format";
import { useLoad } from "../../lib/session";
import { Button, Card, ErrorText, PageTitle, Tabs } from "../../components/ui";

const PAY_STATUS: Record<string, string> = {
  held: "🔒 Séquestre",
  released: "✅ Versé",
  refunded: "↩️ Remboursé",
  partially_refunded: "↩️ Partiel",
};

export default function Payments() {
  const [tab, setTab] = useState<"payments" | "withdrawals">("payments");
  const payments = useLoad(() => api.admin.payments());
  const withdrawals = useLoad(() => api.admin.withdrawals());
  const [error, setError] = useState<string | null>(null);

  const process = async (id: string, status: "paid" | "rejected") => {
    setError(null);
    try {
      withdrawals.setData(await api.admin.processWithdrawal(id, status));
    } catch (e) {
      setError((e as Error).message);
    }
  };

  return (
    <div>
      <PageTitle>Paiements</PageTitle>
      <Tabs value={tab} onChange={setTab} tabs={[["payments", "Transactions"], ["withdrawals", "Retraits"]]} />
      <ErrorText error={error ?? payments.error ?? withdrawals.error} />
      {tab === "payments" ? (
        <Card className="overflow-x-auto p-0">
          <table className="w-full text-sm">
            <thead className="bg-slate-50 text-left text-slate-500">
              <tr>
                <th className="p-2">Job</th>
                <th>Client → Worker</th>
                <th>Montant</th>
                <th>Commission</th>
                <th>Worker</th>
                <th>Statut</th>
                <th>Date</th>
              </tr>
            </thead>
            <tbody>
              {payments.data?.map((p) => (
                <tr key={p.id} className="border-t">
                  <td className="p-2">
                    <Link to={`/jobs/${p.job_id}`} className="hover:underline">
                      {p.job_title}
                    </Link>
                  </td>
                  <td>
                    {p.client_name} → {p.worker_name}
                  </td>
                  <td>{fcfa(p.amount)}</td>
                  <td>{fcfa(p.platform_fee)}</td>
                  <td>{fcfa(p.worker_amount)}</td>
                  <td>
                    {PAY_STATUS[p.status]}
                    {p.refunded_amount > 0 && <span className="block text-xs text-slate-500">−{fcfa(p.refunded_amount)}</span>}
                  </td>
                  <td className="text-xs text-slate-500">{ago(p.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      ) : (
        <div className="space-y-2">
          {withdrawals.data?.map((w) => (
            <Card key={w.id} className="flex flex-wrap items-center gap-3">
              <div className="flex-1">
                <p className="font-semibold">
                  {w.user_name} — {fcfa(w.amount)}
                </p>
                <p className="text-xs text-slate-500">
                  {w.method === "bank" ? "Banque" : "Mobile Money"} · {w.destination} · {ago(w.created_at)}
                </p>
              </div>
              {w.status === "pending" ? (
                <>
                  <Button onClick={() => process(w.id, "paid")}>Marquer payé</Button>
                  <Button variant="secondary" onClick={() => process(w.id, "rejected")}>
                    Refuser
                  </Button>
                </>
              ) : (
                <span className="text-sm text-slate-500">{w.status === "paid" ? "✅ Payé" : "❌ Refusé"}</span>
              )}
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
