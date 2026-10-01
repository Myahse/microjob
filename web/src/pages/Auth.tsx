import { useState, type FormEvent, type ReactNode } from "react";
import { Link, Navigate, useNavigate } from "react-router-dom";
import { api } from "../lib/api";
import { ZONES } from "../lib/format";
import { useSession } from "../lib/session";
import { Button, ErrorText, Field, Input, Select } from "../components/ui";

function Shell({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-indigo-600 to-violet-700 p-4">
      <div className="w-full max-w-md rounded-2xl bg-white p-6 shadow-xl">
        <p className="text-2xl font-extrabold text-indigo-600">MicroJob</p>
        <p className="mb-5 text-sm text-slate-500">J'ai besoin de quelqu'un. Je poste. Quelqu'un proche accepte.</p>
        <h1 className="mb-4 text-lg font-bold">{title}</h1>
        {children}
      </div>
    </div>
  );
}

export function Login() {
  const { user, signIn } = useSession();
  const navigate = useNavigate();
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  if (user) return <Navigate to="/" replace />;

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setBusy(true);
    try {
      const res = await api.login(identifier, password);
      signIn(res.token, res.user);
      navigate("/");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Shell title="Connexion">
      <form onSubmit={submit} className="space-y-3">
        <Field label="Email, username ou téléphone">
          <Input value={identifier} onChange={(e) => setIdentifier(e.target.value)} required autoFocus />
        </Field>
        <Field label="Mot de passe">
          <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
        </Field>
        <ErrorText error={error} />
        <Button className="w-full" disabled={busy}>
          Se connecter
        </Button>
      </form>
      <p className="mt-4 text-center text-sm text-slate-600">
        Pas de compte ?{" "}
        <Link to="/register" className="font-semibold text-indigo-600">
          Créer un compte
        </Link>
      </p>
    </Shell>
  );
}

export function Register() {
  const { user, signIn } = useSession();
  const navigate = useNavigate();
  const [f, setF] = useState({ first_name: "", last_name: "", username: "", phone: "", email: "", password: "", zone: ZONES[0].name });
  const [wants, setWants] = useState(true);
  const [offers, setOffers] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  if (user) return <Navigate to="/" replace />;
  const set = (k: keyof typeof f) => (e: { target: { value: string } }) => setF({ ...f, [k]: e.target.value });

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (!wants && !offers) return setError("Choisissez au moins un rôle");
    setBusy(true);
    const zone = ZONES.find((z) => z.name === f.zone)!;
    try {
      const res = await api.register({ ...f, latitude: zone.lat, longitude: zone.lng, wants_services: wants, offers_services: offers });
      signIn(res.token, res.user);
      navigate(offers ? "/profile" : "/");
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };

  const role = (active: boolean, toggle: () => void, icon: string, label: string) => (
    <button
      type="button"
      onClick={toggle}
      className={`flex-1 rounded-xl border-2 p-3 text-left text-sm ${active ? "border-indigo-600 bg-indigo-50" : "border-slate-200"}`}
    >
      <span className="text-xl">{icon}</span>
      <span className="block font-semibold">{label}</span>
    </button>
  );

  return (
    <Shell title="Créer un compte">
      <form onSubmit={submit} className="space-y-3">
        <div className="grid grid-cols-2 gap-3">
          <Field label="Prénom">
            <Input value={f.first_name} onChange={set("first_name")} required />
          </Field>
          <Field label="Nom">
            <Input value={f.last_name} onChange={set("last_name")} required />
          </Field>
        </div>
        <Field label="Username">
          <Input value={f.username} onChange={set("username")} required pattern="[a-zA-Z0-9_.]{3,30}" />
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Téléphone">
            <Input value={f.phone} onChange={set("phone")} required placeholder="+225 07…" />
          </Field>
          <Field label="Email">
            <Input type="email" value={f.email} onChange={set("email")} required />
          </Field>
        </div>
        <Field label="Mot de passe" hint="8 caractères minimum">
          <Input type="password" value={f.password} onChange={set("password")} required minLength={8} />
        </Field>
        <Field label="Quartier">
          <Select value={f.zone} onChange={set("zone")}>
            {ZONES.map((z) => (
              <option key={z.name}>{z.name}</option>
            ))}
          </Select>
        </Field>
        <div className="flex gap-3">
          {role(wants, () => setWants(!wants), "🙋", "Je cherche des services")}
          {role(offers, () => setOffers(!offers), "🛠", "Je propose mes services")}
        </div>
        <ErrorText error={error} />
        <Button className="w-full" disabled={busy}>
          Créer mon compte
        </Button>
      </form>
      <p className="mt-4 text-center text-sm text-slate-600">
        Déjà inscrit ?{" "}
        <Link to="/login" className="font-semibold text-indigo-600">
          Se connecter
        </Link>
      </p>
    </Shell>
  );
}
