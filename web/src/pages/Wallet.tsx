import { useState, type FormEvent } from "react";
import { Link } from "react-router-dom";
import { api } from "../lib/api";
import { fcfa } from "../lib/format";
import { useLoad, useRealtime } from "../lib/session";
import { Button, Card, ErrorText, Field, Input, PageTitle, Select } from "../components/ui";

export default function WalletPage() {
  const wallet = useLoad(() => api.wallet());
  const [amount, setAmount] = useState("");
  const [method, setMethod] = useState("mobile_money");
  const [destination, setDestination] = useState("");
  const [error, setError] = useState<string | null>(null);
  useRealtime((e) => e.type === "notification" && e.data.kind === "payment" && wallet.reload());
  const w = wallet.data;

  const withdraw = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    try {
      wallet.setData(await api.withdraw(Number(amount), method, destination));
      setAmount("");
    } catch (err) {
      setError((err as Error).message);
    }
  };

  return (
    <div className="mx-auto max-w-3xl space-y-4">
      <PageTitle>Revenus</PageTitle>
      <ErrorText error={wallet.error} />
      {w && (
        <>
          <div className="grid gap-3 sm:grid-cols-3">
            <Card className="bg-indigo-600 text-white">
              <p className="text-sm opacity-80">Solde disponible</p>
              <p className="text-2xl font-extrabold">{fcfa(w.balance)}</p>
            </Card>
            <Card>
              <p className="text-sm text-slate-500">Ce mois-ci</p>
              <p className="text-xl font-bold">{fcfa(w.earned_this_month)}</p>
            </Card>
            <Card>
              <p className="text-sm text-slate-500">Total gagné</p>
              <p className="text-xl font-bold">{fcfa(w.total_earned)}</p>
            </Card>
          </div>

          <Card>
            <h3 className="mb-2 font-bold">Demander un retrait</h3>
            <form onSubmit={withdraw} className="grid gap-3 sm:grid-cols-4 sm:items-end">
              <Field label="Montant">
                <Input type="number" min={500} step={500} value={amount} onChange={(e) => setAmount(e.target.value)} required />
              </Field>
              <Field label="Vers">
                <Select value={method} onChange={(e) => setMethod(e.target.value)}>
                  <option value="mobile_money">Mobile Money</option>
                  <option value="bank">Compte bancaire</option>
                </Select>
              </Field>
              <Field label={method === "bank" ? "IBAN / RIB" : "Numéro"}>
                <Input value={destination} onChange={(e) => setDestination(e.target.value)} required />
              </Field>
              <Button>Retirer</Button>
            </form>
            <ErrorText error={error} />
            {w.pending_withdrawals > 0 && <p className="mt-2 text-sm text-slate-500">{fcfa(w.pending_withdrawals)} en cours de traitement.</p>}
          </Card>

          <Card>
            <h3 className="mb-2 font-bold">Historique</h3>
            {w.transactions.length === 0 && <p className="text-sm text-slate-500">Aucune transaction.</p>}
            <ul className="divide-y">
              {w.transactions.map((t) => (
                <li key={t.id} className="flex justify-between py-2 text-sm">
                  <span>
                    {t.job_id ? (
                      <Link to={`/jobs/${t.job_id}`} className="hover:underline">
                        {t.description}
                      </Link>
                    ) : (
                      t.description
                    )}
                    <span className="block text-xs text-slate-400">{new Date(t.created_at).toLocaleString("fr-FR")}</span>
                  </span>
                  <span className={`font-semibold ${t.amount >= 0 ? "text-emerald-600" : "text-rose-600"}`}>
                    {t.amount >= 0 ? "+" : ""}
                    {fcfa(t.amount)}
                  </span>
                </li>
              ))}
            </ul>
          </Card>
        </>
      )}
    </div>
  );
}
