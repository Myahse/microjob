import { useState } from "react";
import { api } from "../lib/api";
import { useLoad, useRealtime } from "../lib/session";
import { Empty, ErrorText, JobCard, PageTitle, Tabs } from "../components/ui";

export function MyJobs() {
  const [scope, setScope] = useState<"active" | "done" | "cancelled">("active");
  const jobs = useLoad(() => api.myJobs(scope), [scope]);
  useRealtime((e) => e.type === "job" && jobs.reload(), [scope]);
  return (
    <div className="mx-auto max-w-3xl">
      <PageTitle>Mes jobs</PageTitle>
      <Tabs value={scope} onChange={setScope} tabs={[["active", "Actifs"], ["done", "Terminés"], ["cancelled", "Annulés"]]} />
      <ErrorText error={jobs.error} />
      <div className="space-y-3">{jobs.data?.length ? jobs.data.map((j) => <JobCard key={j.id} job={j} />) : <Empty>Rien ici pour l'instant.</Empty>}</div>
    </div>
  );
}

export function Missions() {
  const [scope, setScope] = useState<"available" | "applied" | "active" | "done">("available");
  const jobs = useLoad(() => (scope === "available" ? api.availableJobs() : api.missions(scope)), [scope]);
  useRealtime((e) => (e.type === "job" || (e.type === "notification" && e.data.kind === "job_nearby")) && jobs.reload(), [scope]);
  return (
    <div className="mx-auto max-w-3xl">
      <PageTitle>Mes missions</PageTitle>
      <Tabs
        value={scope}
        onChange={setScope}
        tabs={[
          ["available", "Disponibles"],
          ["applied", "Offres envoyées"],
          ["active", "En cours"],
          ["done", "Terminées"],
        ]}
      />
      <ErrorText error={jobs.error} />
      <div className="space-y-3">{jobs.data?.length ? jobs.data.map((j) => <JobCard key={j.id} job={j} />) : !jobs.error && <Empty>Rien ici pour l'instant.</Empty>}</div>
    </div>
  );
}
