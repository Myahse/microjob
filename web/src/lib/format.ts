import type { JobStatus } from "./api";

export const fcfa = (n: number) => `${new Intl.NumberFormat("fr-FR").format(n)} FCFA`;

export const km = (d: number | null | undefined) => (d == null ? "" : `${d.toFixed(1)} km`);

export function when(iso: string | null | undefined): string {
  if (!iso) return "Dès que possible";
  const d = new Date(iso);
  const today = new Date();
  const time = d.toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" });
  if (d.toDateString() === today.toDateString()) return `Aujourd'hui ${time}`;
  const tomorrow = new Date(today);
  tomorrow.setDate(today.getDate() + 1);
  if (d.toDateString() === tomorrow.toDateString()) return `Demain ${time}`;
  return `${d.toLocaleDateString("fr-FR", { day: "2-digit", month: "short" })} ${time}`;
}

export function ago(iso: string): string {
  const s = (Date.now() - new Date(iso).getTime()) / 1000;
  if (s < 60) return "à l'instant";
  if (s < 3600) return `il y a ${Math.floor(s / 60)} min`;
  if (s < 86400) return `il y a ${Math.floor(s / 3600)} h`;
  return new Date(iso).toLocaleDateString("fr-FR", { day: "2-digit", month: "short" });
}

export const STATUS_LABEL: Record<JobStatus, string> = {
  POSTED: "Publié",
  MATCHING: "Recherche de worker",
  ASSIGNED: "Worker trouvé",
  WORKER_ON_THE_WAY: "Worker en route",
  ARRIVED: "Worker arrivé",
  STARTED: "En cours",
  AWAITING_CONFIRMATION: "À confirmer",
  COMPLETED: "Terminé",
  PAID: "Payé",
  RATED: "Évalué",
  CANCELLED: "Annulé",
  DISPUTED: "Litige",
};

export const STATUS_COLOR: Record<JobStatus, string> = {
  POSTED: "bg-slate-100 text-slate-700",
  MATCHING: "bg-amber-100 text-amber-800",
  ASSIGNED: "bg-indigo-100 text-indigo-800",
  WORKER_ON_THE_WAY: "bg-indigo-100 text-indigo-800",
  ARRIVED: "bg-indigo-100 text-indigo-800",
  STARTED: "bg-blue-100 text-blue-800",
  AWAITING_CONFIRMATION: "bg-orange-100 text-orange-800",
  COMPLETED: "bg-emerald-100 text-emerald-800",
  PAID: "bg-emerald-100 text-emerald-800",
  RATED: "bg-emerald-100 text-emerald-800",
  CANCELLED: "bg-rose-100 text-rose-700",
  DISPUTED: "bg-rose-100 text-rose-700",
};

// The client-facing lifecycle (§16), used for the progress timeline.
export const TIMELINE: JobStatus[] = [
  "MATCHING",
  "ASSIGNED",
  "WORKER_ON_THE_WAY",
  "ARRIVED",
  "STARTED",
  "AWAITING_CONFIRMATION",
  "PAID",
  "RATED",
];

export const REPORT_TYPES: Record<string, string> = {
  not_done: "Travail non réalisé",
  bad_quality: "Mauvaise qualité",
  payment: "Paiement",
  behavior: "Comportement",
  fraud: "Fraude",
  other: "Autre",
};

// Launch zone: Cocody neighbourhoods (§42). Coordinates are area centres.
export const ZONES: { name: string; lat: number; lng: number }[] = [
  { name: "Cocody Centre", lat: 5.348, lng: -3.987 },
  { name: "Riviera", lat: 5.36, lng: -3.97 },
  { name: "Riviera Palmeraie", lat: 5.372, lng: -3.945 },
  { name: "Angré", lat: 5.395, lng: -3.99 },
  { name: "II Plateaux", lat: 5.37, lng: -3.995 },
  { name: "Blockhaus", lat: 5.333, lng: -3.993 },
  { name: "Danga", lat: 5.338, lng: -4.003 },
  { name: "Bonoumin", lat: 5.367, lng: -3.958 },
];

export function currentPosition(): Promise<{ lat: number; lng: number }> {
  return new Promise((resolve, reject) => {
    if (!navigator.geolocation) return reject(new Error("Géolocalisation indisponible"));
    navigator.geolocation.getCurrentPosition(
      (p) => resolve({ lat: p.coords.latitude, lng: p.coords.longitude }),
      () => reject(new Error("Position refusée")),
      { enableHighAccuracy: true, timeout: 10000 },
    );
  });
}
