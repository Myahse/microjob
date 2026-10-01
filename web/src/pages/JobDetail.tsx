import { useEffect, useRef, useState } from "react";
import { Link, useLocation, useParams } from "react-router-dom";
import { api, mediaUrl, type Application, type Job, type JobStatus } from "../lib/api";
import { REPORT_TYPES, STATUS_LABEL, TIMELINE, ago, fcfa, km, when } from "../lib/format";
import { useLoad, useRealtime } from "../lib/session";
import Chat from "../components/Chat";
import { Avatar, Button, Card, ErrorText, Field, Input, Select, Stars, StatusBadge, Textarea } from "../components/ui";

const OPEN: JobStatus[] = ["POSTED", "MATCHING"];
const ACTIVE: JobStatus[] = ["ASSIGNED", "WORKER_ON_THE_WAY", "ARRIVED", "STARTED", "AWAITING_CONFIRMATION"];
const FINISHED: JobStatus[] = ["COMPLETED", "PAID", "RATED"];
const CANCELLABLE: JobStatus[] = ["ASSIGNED", "WORKER_ON_THE_WAY", "ARRIVED"];

function Timeline({ job }: { job: Job }) {
  if (job.status === "CANCELLED" || job.status === "DISPUTED") return null;
  const idx = TIMELINE.indexOf(job.status === "POSTED" ? "MATCHING" : job.status === "COMPLETED" ? "PAID" : job.status);
  return (
    <ol className="flex flex-wrap gap-y-2 text-xs">
      {TIMELINE.map((s, i) => (
        <li key={s} className="flex items-center">
          <span className={`rounded-full px-2 py-1 font-medium ${i < idx ? "bg-emerald-100 text-emerald-700" : i === idx ? "bg-indigo-600 text-white" : "bg-slate-100 text-slate-400"}`}>
            {STATUS_LABEL[s]}
          </span>
          {i < TIMELINE.length - 1 && <span className="mx-1 text-slate-300">→</span>}
        </li>
      ))}
    </ol>
  );
}

function Applications({ job, onChosen }: { job: Job; onChosen: (j: Job) => void }) {
  const apps = useLoad(() => api.applications(job.id), [job.id, job.applications_count]);
  const [error, setError] = useState<string | null>(null);
  const choose = async (a: Application) => {
    if (!confirm(`Choisir ${a.worker?.first_name} pour ${fcfa(a.price)} ? Le montant sera retenu en séquestre.`)) return;
    try {
      onChosen(await api.choose(job.id, a.id));
    } catch (e) {
      setError((e as Error).message);
    }
  };
  const pending = apps.data?.filter((a) => a.status === "pending") ?? [];
  return (
    <Card>
      <h3 className="mb-2 font-bold">Workers intéressés ({pending.length})</h3>
      <ErrorText error={error ?? apps.error} />
      {pending.length === 0 && <p className="text-sm text-slate-500">Les offres apparaîtront ici en temps réel.</p>}
      <div className="space-y-2">
        {pending.map((a) => (
          <div key={a.id} className="flex items-center gap-3 rounded-lg border border-slate-200 p-3">
            <Avatar url={a.worker?.avatar_url} name={a.worker?.first_name ?? "?"} />
            <div className="min-w-0 flex-1">
              <Link to={`/users/${a.worker?.id}`} className="font-semibold hover:underline">
                {a.worker?.first_name} {a.worker?.last_name}
              </Link>
              <div className="flex flex-wrap gap-x-3 text-xs text-slate-500">
                <Stars value={a.worker?.rating ?? 0} count={a.worker?.rating_count} />
                <span>{a.worker?.completed_jobs} jobs</span>
                {a.distance_km != null && <span>📏 {km(a.distance_km)}</span>}
              </div>
              {a.message && <p className="mt-1 text-sm text-slate-700">« {a.message} »</p>}
            </div>
            <div className="text-right">
              <p className="font-bold">{fcfa(a.price)}</p>
              <Button className="mt-1" onClick={() => choose(a)}>
                Choisir
              </Button>
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}

function ReviewForm({ job, onDone }: { job: Job; onDone: (j: Job) => void }) {
  const isClient = job.my_role === "client";
  const criteria: [string, string][] = isClient
    ? [["punctuality", "Ponctualité"], ["communication", "Communication"], ["quality", "Qualité"]]
    : [["respect", "Respect"], ["communication", "Communication"], ["payment", "Paiement"]];
  const [rating, setRating] = useState(5);
  const [sub, setSub] = useState<Record<string, number>>({});
  const [comment, setComment] = useState("");
  const [error, setError] = useState<string | null>(null);
  const starRow = (v: number, set: (n: number) => void) => (
    <span>
      {[1, 2, 3, 4, 5].map((n) => (
        <button type="button" key={n} onClick={() => set(n)} className={`text-xl ${n <= v ? "" : "opacity-25"}`}>
          ⭐
        </button>
      ))}
    </span>
  );
  const submit = async () => {
    try {
      onDone(await api.review(job.id, { rating, comment, ...sub }));
    } catch (e) {
      setError((e as Error).message);
    }
  };
  return (
    <Card className="space-y-2">
      <h3 className="font-bold">Évaluer {isClient ? job.worker?.first_name : job.client.first_name}</h3>
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium">Note globale</span>
        {starRow(rating, setRating)}
      </div>
      {criteria.map(([k, label]) => (
        <div key={k} className="flex items-center justify-between text-sm">
          <span className="text-slate-600">{label}</span>
          {starRow(sub[k] ?? 0, (n) => setSub({ ...sub, [k]: n }))}
        </div>
      ))}
      <Textarea rows={2} placeholder="Un commentaire ?" value={comment} onChange={(e) => setComment(e.target.value)} />
      <ErrorText error={error} />
      <Button onClick={submit}>Envoyer l'évaluation</Button>
    </Card>
  );
}

/** Shares the worker's live position with the client while travelling. */
function useLocationSharing(jobId: string, active: boolean) {
  const last = useRef(0);
  useEffect(() => {
    if (!active || !navigator.geolocation) return;
    const id = navigator.geolocation.watchPosition(
      (p) => {
        if (Date.now() - last.current < 15000) return;
        last.current = Date.now();
        api.shareLocation(jobId, p.coords.latitude, p.coords.longitude).catch(() => {});
      },
      () => {},
      { enableHighAccuracy: true },
    );
    return () => navigator.geolocation.clearWatch(id);
  }, [jobId, active]);
}

export default function JobDetail() {
  const { id = "" } = useParams();
  const location = useLocation();
  const job = useLoad(() => api.job(id), [id]);
  const [error, setError] = useState<string | null>(null);
  const [offer, setOffer] = useState("");
  const [offerMsg, setOfferMsg] = useState("");
  const [reportType, setReportType] = useState("not_done");
  const [reportText, setReportText] = useState("");
  const [showReport, setShowReport] = useState(false);
  const [sharing, setSharing] = useState(false);
  useRealtime((e) => e.type === "job" && e.data.job_id === id && job.reload(), [id]);
  const j = job.data;
  useLocationSharing(id, sharing && !!j && j.my_role === "worker" && CANCELLABLE.includes(j.status));

  if (job.error) return <ErrorText error={job.error} />;
  if (!j) return <p className="text-slate-500">Chargement…</p>;

  const run = async (fn: () => Promise<Job | void>) => {
    setError(null);
    try {
      const res = await fn();
      if (res) job.setData(res);
      else job.reload();
    } catch (e) {
      setError((e as Error).message);
    }
  };
  const cancel = () => {
    const reason = prompt("Raison de l'annulation ?");
    if (reason) run(() => api.cancel(j.id, reason));
  };
  const price = j.agreed_price ?? j.budget;
  const isOpen = OPEN.includes(j.status);
  const notified = (location.state as { notified?: number } | null)?.notified;

  return (
    <div className="grid gap-4 lg:grid-cols-[1fr_24rem]">
      <div className="space-y-4">
        <Card>
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-sm text-slate-500">
                {j.category.icon} {j.category.name}
                {j.skill_name ? ` · ${j.skill_name}` : ""}
              </p>
              <h1 className="text-xl font-bold">{j.title}</h1>
            </div>
            <StatusBadge status={j.status} />
          </div>
          <div className="mt-3 grid grid-cols-2 gap-2 text-sm sm:grid-cols-4">
            <span>📍 {j.zone}</span>
            <span>💰 {j.pricing_mode === "offers" && !j.agreed_price ? "Sur offres" : fcfa(price)}</span>
            <span>🕕 {when(j.scheduled_at)}</span>
            <span>⏳ {j.duration_minutes} min</span>
            {j.distance_km != null && <span>📏 {km(j.distance_km)}</span>}
          </div>
          {j.address && <p className="mt-2 text-sm text-slate-700">🏠 {j.address || "Adresse non précisée"}</p>}
          {j.description && <p className="mt-3 whitespace-pre-wrap text-slate-700">{j.description}</p>}
          {j.photos.length > 0 && (
            <div className="mt-3 flex flex-wrap gap-2">
              {j.photos.map((p) => (
                <a key={p} href={mediaUrl(p)} target="_blank" rel="noreferrer">
                  <img src={mediaUrl(p)} className="h-24 w-24 rounded-lg object-cover" alt="" />
                </a>
              ))}
            </div>
          )}
          {j.my_role !== "candidate" && (
            <div className="mt-4">
              <Timeline job={j} />
            </div>
          )}
        </Card>

        {j.my_role === "client" && isOpen && (
          <Card className="border-amber-200 bg-amber-50">
            <p className="font-semibold">🔎 Recherche de workers à proximité…</p>
            <p className="text-sm text-slate-600">
              {notified !== undefined ? `${notified} worker(s) compatible(s) notifié(s). ` : ""}
              {j.pricing_mode === "fixed" ? "Le premier qui accepte est attribué." : "Choisissez parmi les offres reçues."}
            </p>
          </Card>
        )}
        {j.my_role === "client" && isOpen && j.pricing_mode === "offers" && <Applications job={j} onChosen={(x) => job.setData(x)} />}

        {/* Counterpart */}
        {j.worker && j.my_role !== "worker" && (
          <Card className="flex items-center gap-3">
            <Avatar url={j.worker.avatar_url} name={j.worker.first_name} size={48} />
            <div className="flex-1">
              <Link to={`/users/${j.worker.id}`} className="font-semibold hover:underline">
                {j.worker.first_name} {j.worker.last_name}
              </Link>
              <div className="flex gap-3 text-sm text-slate-500">
                <Stars value={j.worker.rating} count={j.worker.rating_count} />
                <span>{j.worker.completed_jobs} jobs</span>
              </div>
            </div>
            {j.assignment?.worker_distance_km != null && CANCELLABLE.includes(j.status) && (
              <p className="text-right text-sm font-semibold text-indigo-700">
                {j.worker.first_name} est à {km(j.assignment.worker_distance_km)}
                {j.assignment.location_updated_at && <span className="block text-xs font-normal text-slate-400">{ago(j.assignment.location_updated_at)}</span>}
              </p>
            )}
          </Card>
        )}
        {j.my_role === "worker" && (
          <Card className="flex items-center gap-3">
            <Avatar url={j.client.avatar_url} name={j.client.first_name} size={48} />
            <div>
              <p className="font-semibold">
                Client : {j.client.first_name} {j.client.last_name}
              </p>
              <Stars value={j.client.rating} count={j.client.rating_count} />
            </div>
          </Card>
        )}

        {j.payment && (j.my_role === "client" || j.my_role === "worker" || j.my_role === "admin") && (
          <Card className="text-sm">
            <h3 className="mb-1 font-bold">Paiement</h3>
            <div className="grid grid-cols-2 gap-1">
              <span className="text-slate-500">Montant</span>
              <span>{fcfa(j.payment.amount)}</span>
              <span className="text-slate-500">Commission ({Math.round(j.payment.commission_rate * 100)} %)</span>
              <span>{fcfa(j.payment.platform_fee)}</span>
              <span className="text-slate-500">Worker reçoit</span>
              <span className="font-semibold">{fcfa(j.payment.worker_amount)}</span>
              {j.payment.refunded_amount > 0 && (
                <>
                  <span className="text-slate-500">Remboursé</span>
                  <span>{fcfa(j.payment.refunded_amount)}</span>
                </>
              )}
              <span className="text-slate-500">Statut</span>
              <span>
                {
                  { held: "🔒 Retenu (séquestre)", released: "✅ Versé", refunded: "↩️ Remboursé", partially_refunded: "↩️ Partiellement remboursé" }[
                    j.payment.status
                  ]
                }
              </span>
            </div>
          </Card>
        )}

        {j.history && j.history.length > 0 && (
          <Card>
            <h3 className="mb-2 font-bold">Historique</h3>
            <ul className="space-y-1 text-sm">
              {j.history.map((h, i) => (
                <li key={i} className="flex justify-between gap-2">
                  <span>
                    <span className="font-medium">{STATUS_LABEL[h.status as JobStatus] ?? h.status}</span>
                    {h.note && <span className="text-slate-500"> — {h.note}</span>}
                  </span>
                  <span className="shrink-0 text-xs text-slate-400">{ago(h.created_at)}</span>
                </li>
              ))}
            </ul>
          </Card>
        )}
      </div>

      {/* Action column */}
      <div className="space-y-4">
        <ErrorText error={error} />

        {j.my_role === "candidate" && isOpen && j.pricing_mode === "fixed" && (
          <Card className="space-y-2">
            <p className="text-2xl font-extrabold">{fcfa(j.budget)}</p>
            <Button className="w-full py-3" onClick={() => run(() => api.accept(j.id))}>
              ACCEPT JOB
            </Button>
            <Button variant="secondary" className="w-full" onClick={() => run(() => api.decline(j.id))}>
              DECLINE
            </Button>
          </Card>
        )}
        {j.my_role === "candidate" && isOpen && j.pricing_mode === "offers" && (
          <Card className="space-y-2">
            <h3 className="font-bold">Proposez votre prix</h3>
            {j.my_application?.status === "pending" && (
              <p className="rounded-lg bg-indigo-50 p-2 text-sm text-indigo-800">Offre envoyée : {fcfa(j.my_application.price)}. Vous pouvez la modifier.</p>
            )}
            {j.budget > 0 && <p className="text-sm text-slate-500">Budget indicatif : {fcfa(j.budget)}</p>}
            <Field label="Votre prix (FCFA)">
              <Input type="number" min={0} step={500} value={offer} onChange={(e) => setOffer(e.target.value)} />
            </Field>
            <Input placeholder="Message (optionnel)" value={offerMsg} onChange={(e) => setOfferMsg(e.target.value)} />
            <Button className="w-full" disabled={!offer} onClick={() => run(() => api.apply(j.id, Number(offer), offerMsg))}>
              Envoyer l'offre
            </Button>
            <Button variant="secondary" className="w-full" onClick={() => run(() => api.decline(j.id))}>
              Pas intéressé
            </Button>
          </Card>
        )}
        {j.my_role === "candidate" && !isOpen && <Card className="text-sm text-slate-600">Ce job a été attribué à un autre worker.</Card>}

        {j.my_role === "worker" && ACTIVE.includes(j.status) && (
          <Card className="space-y-2">
            <h3 className="font-bold">Mission</h3>
            {j.status === "ASSIGNED" && (
              <>
                <Button className="w-full" onClick={() => run(() => api.advance(j.id, "WORKER_ON_THE_WAY"))}>
                  🚗 Je suis en route
                </Button>
                <Button variant="secondary" className="w-full" onClick={() => run(() => api.advance(j.id, "STARTED"))}>
                  Commencer directement (job à distance)
                </Button>
              </>
            )}
            {j.status === "WORKER_ON_THE_WAY" && (
              <Button className="w-full" onClick={() => run(() => api.advance(j.id, "ARRIVED"))}>
                📍 ARRIVED
              </Button>
            )}
            {(j.status === "ASSIGNED" || j.status === "WORKER_ON_THE_WAY") && (
              <label className="flex items-center gap-2 text-sm">
                <input type="checkbox" checked={sharing} onChange={(e) => setSharing(e.target.checked)} />
                Partager ma position avec le client
              </label>
            )}
            {j.status === "ARRIVED" && (
              <Button className="w-full" onClick={() => run(() => api.advance(j.id, "STARTED"))}>
                🛠 START JOB
              </Button>
            )}
            {j.status === "STARTED" && (
              <Button className="w-full" onClick={() => run(() => api.advance(j.id, "AWAITING_CONFIRMATION"))}>
                🏁 COMPLETE JOB
              </Button>
            )}
            {j.status === "AWAITING_CONFIRMATION" && <p className="text-sm text-slate-600">En attente de la confirmation du client…</p>}
            {j.latitude != null && (
              <a
                className="block text-center text-sm text-indigo-600 underline"
                target="_blank"
                rel="noreferrer"
                href={`https://www.google.com/maps/dir/?api=1&destination=${j.latitude},${j.longitude}`}
              >
                Itinéraire vers le client
              </a>
            )}
          </Card>
        )}

        {j.my_role === "client" && j.status === "AWAITING_CONFIRMATION" && (
          <Card className="space-y-2 border-emerald-200 bg-emerald-50">
            <p className="font-semibold">Le worker indique que le job est terminé.</p>
            <Button className="w-full py-3" onClick={() => run(() => api.confirm(j.id))}>
              CONFIRM COMPLETION
            </Button>
            <p className="text-xs text-slate-600">La confirmation libère le paiement au worker.</p>
          </Card>
        )}

        {FINISHED.includes(j.status) && !j.reviewed_by_me && (j.my_role === "client" || j.my_role === "worker") && (
          <ReviewForm job={j} onDone={(x) => job.setData(x)} />
        )}
        {FINISHED.includes(j.status) && j.reviewed_by_me && <Card className="text-sm text-emerald-700">✅ Merci pour votre évaluation.</Card>}

        {j.has_conversation && <Chat jobId={j.id} closed={j.status === "CANCELLED"} />}

        {(j.my_role === "client" || j.my_role === "worker") && (
          <div className="flex flex-wrap gap-2">
            {((j.my_role === "client" && (isOpen || CANCELLABLE.includes(j.status))) || (j.my_role === "worker" && CANCELLABLE.includes(j.status))) && (
              <Button variant="secondary" onClick={cancel}>
                Annuler le job
              </Button>
            )}
            {(ACTIVE.includes(j.status) || FINISHED.includes(j.status)) && !j.open_report && (
              <Button variant="ghost" onClick={() => setShowReport(!showReport)}>
                ⚠️ Signaler un problème
              </Button>
            )}
          </div>
        )}
        {j.open_report && <Card className="text-sm text-rose-700">⚖️ Un signalement est en cours d'examen par l'équipe MicroJob.</Card>}
        {showReport && (
          <Card className="space-y-2">
            <h3 className="font-bold">Report a problem</h3>
            <Select value={reportType} onChange={(e) => setReportType(e.target.value)}>
              {Object.entries(REPORT_TYPES).map(([k, v]) => (
                <option key={k} value={k}>
                  {v}
                </option>
              ))}
            </Select>
            <Textarea rows={3} placeholder="Décrivez le problème" value={reportText} onChange={(e) => setReportText(e.target.value)} />
            <Button
              variant="danger"
              onClick={() =>
                run(async () => {
                  const res = await api.report(j.id, reportType, reportText);
                  setShowReport(false);
                  return res;
                })
              }
            >
              Envoyer le signalement
            </Button>
          </Card>
        )}
      </div>
    </div>
  );
}
