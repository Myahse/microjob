import { useState, type FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { api, mediaUrl } from "../lib/api";
import { ZONES, currentPosition, fcfa } from "../lib/format";
import { useLoad, useSession } from "../lib/session";
import { Button, Card, ErrorText, Field, Group, Input, PageTitle, Select, Textarea } from "../components/ui";

export default function PostJob() {
  const { user } = useSession();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const cats = useLoad(() => api.categories());
  const [categoryId, setCategoryId] = useState(Number(params.get("category")) || 0);
  const [skillId, setSkillId] = useState(0);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [zone, setZone] = useState(ZONES.find((z) => z.name === user?.zone)?.name ?? ZONES[0].name);
  const [coords, setCoords] = useState<{ lat: number; lng: number } | null>(null);
  const [address, setAddress] = useState("");
  const [date, setDate] = useState("");
  const [time, setTime] = useState("");
  const [mode, setMode] = useState<"fixed" | "offers">("fixed");
  const [budget, setBudget] = useState("");
  const [duration, setDuration] = useState("60");
  const [photos, setPhotos] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const category = cats.data?.find((c) => c.id === categoryId);

  const gps = async () => {
    try {
      setCoords(await currentPosition());
    } catch (e) {
      setError((e as Error).message);
    }
  };

  const addPhoto = async (files: FileList | null) => {
    if (!files?.[0]) return;
    try {
      const { url } = await api.upload(files[0]);
      setPhotos([...photos, url]);
    } catch (e) {
      setError((e as Error).message);
    }
  };

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    if (!categoryId) return setError("Choisissez une catégorie");
    const z = ZONES.find((x) => x.name === zone)!;
    const at = date ? new Date(`${date}T${time || "09:00"}`).toISOString() : null;
    setBusy(true);
    try {
      const res = await api.createJob({
        category_id: categoryId,
        skill_id: skillId || null,
        title,
        description,
        latitude: coords?.lat ?? z.lat,
        longitude: coords?.lng ?? z.lng,
        address,
        zone,
        pricing_mode: mode,
        budget: Number(budget) || 0,
        scheduled_at: at,
        duration_minutes: Number(duration) || 60,
        photos,
      });
      navigate(`/jobs/${res.job.id}`, { state: { notified: res.notified_workers } });
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="mx-auto max-w-2xl">
      <PageTitle>Poster un job</PageTitle>
      <form onSubmit={submit} className="space-y-4">
        <Card className="space-y-3">
          <Group label="Catégorie">
            <div className="grid grid-cols-3 gap-2 sm:grid-cols-5">
              {cats.data?.map((c) => (
                <button
                  type="button"
                  key={c.id}
                  onClick={() => {
                    setCategoryId(c.id);
                    setSkillId(0);
                  }}
                  className={`rounded-lg border-2 p-2 text-sm ${categoryId === c.id ? "border-indigo-600 bg-indigo-50" : "border-slate-200"}`}
                >
                  <div className="text-2xl">{c.icon}</div>
                  {c.name}
                </button>
              ))}
            </div>
          </Group>
          {category && (
            <Field label="Type de service">
              <Select value={skillId} onChange={(e) => setSkillId(Number(e.target.value))}>
                <option value={0}>Autre / non précisé</option>
                {category.skills.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name}
                  </option>
                ))}
              </Select>
            </Field>
          )}
          <Field label="Titre">
            <Input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Installer Windows sur mon PC" required minLength={3} />
          </Field>
          <Field label="Description">
            <Textarea rows={3} value={description} onChange={(e) => setDescription(e.target.value)} placeholder="J'ai besoin d'une installation complète de Windows 11." />
          </Field>
        </Card>

        <Card className="space-y-3">
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="📍 Quartier" hint="Visible par les workers">
              <Select value={zone} onChange={(e) => setZone(e.target.value)}>
                {ZONES.map((z) => (
                  <option key={z.name}>{z.name}</option>
                ))}
              </Select>
            </Field>
            <Field label="Adresse exacte" hint="Partagée uniquement avec le worker choisi">
              <Input value={address} onChange={(e) => setAddress(e.target.value)} placeholder="Rue, résidence, repère…" />
            </Field>
          </div>
          <Button type="button" variant="secondary" onClick={gps}>
            {coords ? "✅ Position GPS enregistrée" : "📍 Utiliser ma position exacte"}
          </Button>
          <div className="grid grid-cols-3 gap-3">
            <Field label="Date">
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </Field>
            <Field label="Heure">
              <Input type="time" value={time} onChange={(e) => setTime(e.target.value)} />
            </Field>
            <Field label="Durée (min)">
              <Input type="number" min={15} step={15} value={duration} onChange={(e) => setDuration(e.target.value)} />
            </Field>
          </div>
        </Card>

        <Card className="space-y-3">
          <div className="flex gap-2">
            {(
              [
                ["fixed", "💰 Prix fixe"],
                ["offers", "🤝 Proposez votre prix"],
              ] as const
            ).map(([m, label]) => (
              <button
                type="button"
                key={m}
                onClick={() => setMode(m)}
                className={`flex-1 rounded-lg border-2 p-2 text-sm font-medium ${mode === m ? "border-indigo-600 bg-indigo-50" : "border-slate-200"}`}
              >
                {label}
              </button>
            ))}
          </div>
          <Field label={mode === "fixed" ? "Budget (FCFA)" : "Budget indicatif (FCFA, optionnel)"} hint={budget ? fcfa(Number(budget)) : undefined}>
            <Input type="number" min={0} step={500} value={budget} onChange={(e) => setBudget(e.target.value)} required={mode === "fixed"} />
          </Field>
          <Group label="Photos (optionnel)">
            <div className="flex flex-wrap gap-2">
              {photos.map((p) => (
                <img key={p} src={mediaUrl(p)} className="h-16 w-16 rounded-lg object-cover" alt="" />
              ))}
              {photos.length < 6 && (
                <label className="flex h-16 w-16 cursor-pointer items-center justify-center rounded-lg border-2 border-dashed border-slate-300 text-2xl text-slate-400">
                  +
                  <input type="file" accept="image/*" className="hidden" onChange={(e) => addPhoto(e.target.files)} />
                </label>
              )}
            </div>
          </Group>
        </Card>
        <ErrorText error={error} />
        <Button className="w-full py-3 text-base" disabled={busy}>
          POST JOB
        </Button>
      </form>
    </div>
  );
}
