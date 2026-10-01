import { useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../lib/api";
import { ZONES, currentPosition } from "../lib/format";
import { useLoad, useRealtime, useSession } from "../lib/session";
import { Button, Card, Empty, ErrorText, JobCard, Select } from "../components/ui";

function ClientHome() {
  const { user } = useSession();
  const cats = useLoad(() => api.categories());
  const active = useLoad(() => api.myJobs("active"));
  const nearby = useLoad(
    () => (user?.latitude != null ? api.nearbyCount(user.latitude, user.longitude!, 5) : Promise.resolve(null)),
    [user?.latitude, user?.longitude],
  );
  useRealtime((e) => e.type === "job" && active.reload());

  return (
    <section className="space-y-4">
      <div className="rounded-2xl bg-gradient-to-br from-indigo-600 to-violet-700 p-6 text-white">
        <p className="text-sm opacity-80">Bonjour {user?.first_name} 👋</p>
        <h2 className="mt-1 text-2xl font-extrabold">Besoin de quelqu'un ? Postez le job.</h2>
        <p className="mt-1 text-sm opacity-90">
          {nearby.data ? `${nearby.data.count} workers disponibles à moins de ${nearby.data.radius_km} km` : "Des workers près de chez vous"}
        </p>
        <Link to="/post" className="mt-4 inline-block rounded-lg bg-white px-4 py-2 font-semibold text-indigo-700">
          POST A JOB
        </Link>
      </div>
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-5">
        {cats.data?.map((c) => (
          <Link key={c.id} to={`/post?category=${c.id}`}>
            <Card className="text-center hover:border-indigo-300">
              <div className="text-3xl">{c.icon}</div>
              <div className="mt-1 text-sm font-semibold">{c.name}</div>
            </Card>
          </Link>
        ))}
      </div>
      <h3 className="pt-2 font-bold">Mes jobs en cours</h3>
      {active.data?.length ? active.data.map((j) => <JobCard key={j.id} job={j} />) : <Empty>Aucun job en cours.</Empty>}
    </section>
  );
}

function WorkerHome() {
  const { user, setUser } = useSession();
  const feed = useLoad(() => api.availableJobs(), [user?.latitude, user?.worker?.skills.length]);
  const [error, setError] = useState<string | null>(null);
  const [zone, setZone] = useState(user?.zone || ZONES[0].name);
  useRealtime((e) => {
    if (e.type === "notification" && e.data.kind === "job_nearby") feed.reload();
    if (e.type === "job") feed.reload();
  });
  const w = user!.worker!;

  const toggle = async (available: boolean, useGps = false) => {
    setError(null);
    try {
      let pos: { latitude: number; longitude: number; zone?: string } | undefined;
      if (useGps) {
        const p = await currentPosition();
        pos = { latitude: p.lat, longitude: p.lng };
      } else if (user?.latitude == null || zone !== user.zone) {
        const z = ZONES.find((z) => z.name === zone)!;
        pos = { latitude: z.lat, longitude: z.lng, zone: z.name };
      }
      setUser(await api.setAvailability({ available, ...pos }));
    } catch (e) {
      setError((e as Error).message);
    }
  };

  return (
    <section className="space-y-4">
      <Card>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p className="font-bold">{w.available ? "🟢 Disponible" : "⚪ Hors ligne"}</p>
            <p className="text-sm text-slate-500">
              Zone : {user?.zone || "non définie"} · rayon {w.radius_km} km · {w.skills.length} compétence(s)
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Select value={zone} onChange={(e) => setZone(e.target.value)} className="w-auto">
              {ZONES.map((z) => (
                <option key={z.name}>{z.name}</option>
              ))}
            </Select>
            <Button variant="secondary" onClick={() => toggle(w.available, true)}>
              📍 Ma position
            </Button>
            <Button variant={w.available ? "secondary" : "primary"} onClick={() => toggle(!w.available)}>
              {w.available ? "Passer hors ligne" : "Je suis disponible"}
            </Button>
          </div>
        </div>
        {w.skills.length === 0 && (
          <p className="mt-3 text-sm text-amber-700">
            Ajoutez vos compétences dans <Link to="/profile" className="underline">votre profil</Link> pour recevoir des jobs.
          </p>
        )}
        <ErrorText error={error} />
      </Card>
      <h3 className="font-bold">Jobs disponibles près de vous</h3>
      <ErrorText error={feed.error} />
      {feed.data?.length ? feed.data.map((j) => <JobCard key={j.id} job={j} />) : !feed.error && <Empty>Aucun job à proximité pour le moment.</Empty>}
    </section>
  );
}

export default function Home() {
  const { user } = useSession();
  const [view, setViewState] = useState<"client" | "worker">(() => {
    try {
      const saved = localStorage.getItem("microjob.view");
      if (saved === "client" || saved === "worker") return saved;
    } catch {
      /* storage unavailable */
    }
    return user?.is_worker && !user.is_client ? "worker" : "client";
  });
  const setView = (v: "client" | "worker") => {
    setViewState(v);
    try {
      localStorage.setItem("microjob.view", v);
    } catch {
      /* storage unavailable */
    }
  };
  if (!user) return null;
  const both = user.is_worker && user.is_client;
  return (
    <div>
      {both && (
        <div className="mb-4 inline-flex rounded-lg bg-slate-100 p-1">
          {(["client", "worker"] as const).map((v) => (
            <button key={v} onClick={() => setView(v)} className={`rounded-md px-4 py-1.5 text-sm font-medium ${view === v ? "bg-white shadow-sm" : "text-slate-600"}`}>
              {v === "client" ? "🙋 Client" : "🛠 Worker"}
            </button>
          ))}
        </div>
      )}
      {(view === "worker" || !user.is_client) && user.is_worker && user.worker ? <WorkerHome /> : <ClientHome />}
    </div>
  );
}
