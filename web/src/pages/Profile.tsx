import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { api, type User } from "../lib/api";
import { ZONES } from "../lib/format";
import { useLoad, useSession } from "../lib/session";
import { Avatar, Button, Card, ErrorText, Field, Group, Input, PageTitle, Select, Stars, Textarea } from "../components/ui";

export default function Profile() {
  const { user, setUser } = useSession();
  const cats = useLoad(() => api.categories());
  const [info, setInfo] = useState({ first_name: "", last_name: "", bio: "", zone: "" });
  const [worker, setWorker] = useState({ headline: "", hourly_rate: "0", radius_km: "10" });
  const [skills, setSkills] = useState<number[]>([]);
  const [msg, setMsg] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!user) return;
    setInfo({ first_name: user.first_name, last_name: user.last_name, bio: user.bio, zone: user.zone });
    if (user.worker) {
      setWorker({ headline: user.worker.headline, hourly_rate: String(user.worker.hourly_rate), radius_km: String(user.worker.radius_km) });
      setSkills(user.worker.skills.map((s) => s.id));
    }
  }, [user]);
  if (!user) return null;

  const save = async (fn: () => Promise<User>) => {
    setError(null);
    setMsg(null);
    try {
      setUser(await fn());
      setMsg("Enregistré ✅");
    } catch (e) {
      setError((e as Error).message);
    }
  };

  const avatar = async (files: FileList | null) => {
    if (!files?.[0]) return;
    save(async () => {
      const { url } = await api.upload(files[0]);
      return api.updateMe({ avatar_url: url });
    });
  };

  const saveInfo = () => {
    const z = ZONES.find((x) => x.name === info.zone);
    save(() => api.updateMe({ ...info, ...(z && z.name !== user.zone ? { latitude: z.lat, longitude: z.lng } : {}) }));
  };

  const saveWorker = () =>
    save(() =>
      api.updateWorker({
        headline: worker.headline,
        hourly_rate: Number(worker.hourly_rate) || 0,
        radius_km: Number(worker.radius_km) || 10,
        skill_ids: skills,
      }),
    );

  const toggleSkill = (id: number) => setSkills(skills.includes(id) ? skills.filter((s) => s !== id) : [...skills, id]);

  return (
    <div className="mx-auto max-w-3xl space-y-4">
      <PageTitle action={<Link to={`/users/${user.id}`} className="text-sm text-indigo-600">Voir mon profil public →</Link>}>Mon profil</PageTitle>
      <Card className="flex items-center gap-4">
        <label className="cursor-pointer" title="Changer la photo">
          <Avatar url={user.avatar_url} name={user.first_name} size={72} />
          <input type="file" accept="image/*" className="hidden" onChange={(e) => avatar(e.target.files)} />
        </label>
        <div>
          <p className="text-lg font-bold">
            {user.first_name} {user.last_name} <span className="text-sm font-normal text-slate-500">@{user.username}</span>
          </p>
          <p className="text-sm text-slate-500">
            {user.email} · {user.phone}
          </p>
          <div className="mt-1 flex flex-wrap gap-2 text-xs">
            {user.identity_verified && <span className="rounded-full bg-emerald-100 px-2 py-0.5 text-emerald-800">✅ Identité vérifiée</span>}
            {user.phone_verified && <span className="rounded-full bg-slate-100 px-2 py-0.5">📱 Téléphone vérifié</span>}
            {user.worker?.badges.map((b) => (
              <span key={b} className="rounded-full bg-indigo-100 px-2 py-0.5 text-indigo-800">
                {b}
              </span>
            ))}
          </div>
        </div>
      </Card>

      <ErrorText error={error} />
      {msg && <p className="text-sm text-emerald-700">{msg}</p>}

      <Card className="space-y-3">
        <h3 className="font-bold">Informations</h3>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label="Prénom">
            <Input value={info.first_name} onChange={(e) => setInfo({ ...info, first_name: e.target.value })} />
          </Field>
          <Field label="Nom">
            <Input value={info.last_name} onChange={(e) => setInfo({ ...info, last_name: e.target.value })} />
          </Field>
        </div>
        <Field label="Quartier">
          <Select value={info.zone} onChange={(e) => setInfo({ ...info, zone: e.target.value })}>
            <option value="">—</option>
            {ZONES.map((z) => (
              <option key={z.name}>{z.name}</option>
            ))}
          </Select>
        </Field>
        <Field label="Bio">
          <Textarea rows={2} value={info.bio} onChange={(e) => setInfo({ ...info, bio: e.target.value })} />
        </Field>
        <Button onClick={saveInfo}>Enregistrer</Button>
      </Card>

      <Card className="space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="font-bold">{user.is_worker ? "Profil worker" : "Je veux proposer mes services"}</h3>
          {user.worker && (
            <span className="text-sm">
              <Stars value={user.worker.rating} count={user.worker.rating_count} /> · {user.worker.completed_jobs} jobs · {user.worker.completion_rate} %
            </span>
          )}
        </div>
        <Field label="Titre professionnel">
          <Input placeholder="💻 Développeur / Technicien" value={worker.headline} onChange={(e) => setWorker({ ...worker, headline: e.target.value })} />
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Tarif horaire (FCFA)">
            <Input type="number" min={0} step={500} value={worker.hourly_rate} onChange={(e) => setWorker({ ...worker, hourly_rate: e.target.value })} />
          </Field>
          <Field label="Rayon d'intervention (km)">
            <Input type="number" min={1} max={50} value={worker.radius_km} onChange={(e) => setWorker({ ...worker, radius_km: e.target.value })} />
          </Field>
        </div>
        <Group label="Compétences">
          <div className="space-y-2">
            {cats.data?.map((c) => (
              <div key={c.id}>
                <p className="text-sm font-semibold">
                  {c.icon} {c.name}
                </p>
                <div className="mt-1 flex flex-wrap gap-1.5">
                  {c.skills.map((s) => {
                    const on = skills.includes(s.id);
                    const verified = user.worker?.skills.find((x) => x.id === s.id)?.verified;
                    return (
                      <button
                        type="button"
                        key={s.id}
                        onClick={() => toggleSkill(s.id)}
                        className={`rounded-full border px-3 py-1 text-sm ${on ? "border-indigo-600 bg-indigo-600 text-white" : "border-slate-300"}`}
                      >
                        {s.name}
                        {verified ? " ✓" : ""}
                      </button>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>
        </Group>
        <Button onClick={saveWorker}>{user.is_worker ? "Enregistrer" : "Activer mon profil worker"}</Button>
      </Card>
    </div>
  );
}
