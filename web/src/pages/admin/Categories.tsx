import { useState } from "react";
import { api, type Category } from "../../lib/api";
import { useLoad } from "../../lib/session";
import { Button, Card, ErrorText, Input, PageTitle } from "../../components/ui";

function CategoryCard({ c, onChange }: { c: Category; onChange: (cats: Category[]) => void }) {
  const [skill, setSkill] = useState("");
  const [rate, setRate] = useState(c.commission_rate == null ? "" : String(c.commission_rate));
  const [error, setError] = useState<string | null>(null);
  const run = async (fn: () => Promise<Category[]>) => {
    setError(null);
    try {
      onChange(await fn());
    } catch (e) {
      setError((e as Error).message);
    }
  };
  return (
    <Card className={c.active ? "" : "opacity-60"}>
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-2xl">{c.icon}</span>
        <h3 className="flex-1 font-bold">
          {c.name} {!c.active && <span className="text-xs font-normal text-rose-600">(désactivée)</span>}
        </h3>
        <Input
          className="w-28"
          placeholder="défaut"
          value={rate}
          onChange={(e) => setRate(e.target.value)}
          title="Commission de la catégorie (0.10 = 10 %). Vide = taux par défaut."
        />
        <Button
          variant="secondary"
          onClick={() => run(() => api.admin.updateCategory(c.id, rate === "" ? { clear_commission_rate: true } : { commission_rate: Number(rate) }))}
        >
          Taux
        </Button>
        <Button variant={c.active ? "secondary" : "primary"} onClick={() => run(() => api.admin.updateCategory(c.id, { active: !c.active }))}>
          {c.active ? "Désactiver" : "Activer"}
        </Button>
      </div>
      <div className="mt-3 flex flex-wrap gap-1.5">
        {c.skills.map((s) => (
          <button
            key={s.id}
            onClick={() => run(() => api.admin.updateSkill(s.id, { active: !s.active }))}
            className={`rounded-full border px-3 py-1 text-sm ${s.active ? "border-slate-300" : "border-dashed text-slate-400 line-through"}`}
            title={s.active ? "Désactiver" : "Activer"}
          >
            {s.name}
          </button>
        ))}
      </div>
      <form
        className="mt-3 flex gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          if (skill.trim()) run(() => api.admin.createSkill(c.id, skill)).then(() => setSkill(""));
        }}
      >
        <Input placeholder="Nouvelle compétence" value={skill} onChange={(e) => setSkill(e.target.value)} />
        <Button variant="secondary">Ajouter</Button>
      </form>
      <ErrorText error={error} />
    </Card>
  );
}

export default function Categories() {
  const cats = useLoad(() => api.admin.categories());
  const [name, setName] = useState("");
  const [icon, setIcon] = useState("");
  const [error, setError] = useState<string | null>(null);
  const create = async () => {
    setError(null);
    try {
      cats.setData(await api.admin.createCategory({ name, icon }));
      setName("");
      setIcon("");
    } catch (e) {
      setError((e as Error).message);
    }
  };
  return (
    <div className="space-y-3">
      <PageTitle>Catégories</PageTitle>
      <Card className="flex flex-wrap gap-2">
        <Input className="w-20" placeholder="🎉" value={icon} onChange={(e) => setIcon(e.target.value)} />
        <Input className="flex-1" placeholder="Nouvelle catégorie (ex. Événementiel)" value={name} onChange={(e) => setName(e.target.value)} />
        <Button onClick={create} disabled={!name.trim()}>
          Créer
        </Button>
      </Card>
      <ErrorText error={error ?? cats.error} />
      {cats.data?.map((c) => <CategoryCard key={c.id} c={c} onChange={(list) => cats.setData(list)} />)}
    </div>
  );
}
