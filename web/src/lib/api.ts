// Typed client for the MicroJob Go API.

export const API_URL: string = import.meta.env.VITE_API_URL ?? "";

export type JobStatus =
  | "POSTED"
  | "MATCHING"
  | "ASSIGNED"
  | "WORKER_ON_THE_WAY"
  | "ARRIVED"
  | "STARTED"
  | "AWAITING_CONFIRMATION"
  | "COMPLETED"
  | "PAID"
  | "RATED"
  | "CANCELLED"
  | "DISPUTED";

export interface Skill {
  id: number;
  name: string;
  category_id: number;
  category_name?: string;
  verified?: boolean;
  active: boolean;
}

export interface Category {
  id: number;
  slug: string;
  name: string;
  icon: string;
  commission_rate: number | null;
  active: boolean;
  sort_order: number;
  skills: Skill[];
}

export interface WorkerProfile {
  headline: string;
  available: boolean;
  hourly_rate: number;
  radius_km: number;
  rating: number;
  rating_count: number;
  completed_jobs: number;
  completion_rate: number;
  skills: Skill[];
  badges: string[];
}

export interface User {
  id: string;
  first_name: string;
  last_name: string;
  username: string;
  avatar_url: string;
  bio: string;
  zone: string;
  is_client: boolean;
  is_worker: boolean;
  identity_verified: boolean;
  phone_verified: boolean;
  email_verified: boolean;
  client_rating: number;
  client_rating_count: number;
  worker?: WorkerProfile;
  created_at: string;
  email?: string;
  phone?: string;
  is_admin?: boolean;
  status?: "active" | "suspended" | "banned";
  latitude?: number;
  longitude?: number;
}

export interface PersonRef {
  id: string;
  first_name: string;
  last_name: string;
  avatar_url: string;
  rating: number;
  rating_count: number;
  completed_jobs?: number;
}

export interface Application {
  id: string;
  price: number;
  message: string;
  status: "pending" | "accepted" | "rejected" | "withdrawn";
  created_at: string;
  worker?: PersonRef;
  headline?: string;
  distance_km?: number;
}

export interface Payment {
  id: string;
  amount: number;
  platform_fee: number;
  worker_amount: number;
  refunded_amount: number;
  commission_rate: number;
  provider: string;
  status: "held" | "released" | "refunded" | "partially_refunded";
  created_at: string;
}

export interface Job {
  id: string;
  title: string;
  description: string;
  category: { id: number; name: string; icon: string };
  skill_id: number | null;
  skill_name: string | null;
  zone: string;
  pricing_mode: "fixed" | "offers";
  budget: number;
  agreed_price: number | null;
  scheduled_at: string | null;
  duration_minutes: number;
  photos: string[];
  status: JobStatus;
  created_at: string;
  updated_at: string;
  client: PersonRef;
  worker: PersonRef | null;
  applications_count: number;
  my_role: "client" | "worker" | "candidate" | "admin";
  address?: string;
  latitude?: number;
  longitude?: number;
  distance_km?: number;
  assignment?: {
    assigned_at: string;
    on_the_way_at: string | null;
    arrived_at: string | null;
    started_at: string | null;
    finished_at: string | null;
    completed_at: string | null;
    worker_latitude: number | null;
    worker_longitude: number | null;
    worker_distance_km: number | null;
    location_updated_at: string | null;
  };
  payment?: Payment;
  my_application?: Application;
  reviewed_by_me: boolean;
  reviewed_by_other: boolean;
  open_report: boolean;
  has_conversation: boolean;
  history?: { status: string; note: string; created_at: string }[];
}

export interface Message {
  id: string;
  job_id: string;
  sender_id: string | null;
  kind: "text" | "image" | "location" | "system";
  body: string;
  attachment_url: string;
  latitude: number | null;
  longitude: number | null;
  created_at: string;
}

export interface Conversation {
  job_id: string;
  job_title: string;
  job_status: JobStatus;
  other: PersonRef;
  last_message: string | null;
  last_at: string | null;
  closed: boolean;
}

export interface Notification {
  id: string;
  kind: string;
  title: string;
  body: string;
  job_id: string | null;
  read_at: string | null;
  created_at: string;
}

export interface Wallet {
  balance: number;
  total_earned: number;
  earned_this_month: number;
  pending_withdrawals: number;
  transactions: { id: string; amount: number; kind: string; job_id: string | null; description: string; created_at: string }[];
  withdrawals: Withdrawal[];
}

export interface Withdrawal {
  id: string;
  user_id?: string;
  user_name?: string;
  amount: number;
  method: "mobile_money" | "bank";
  destination: string;
  status: "pending" | "paid" | "rejected";
  created_at: string;
  processed_at: string | null;
}

export interface Review {
  id: string;
  job_title: string;
  reviewer_name: string;
  reviewer_role: "client" | "worker";
  rating: number;
  comment: string;
  created_at: string;
}

export interface WorkerSearchResult {
  id: string;
  first_name: string;
  last_name: string;
  avatar_url: string;
  headline: string;
  zone: string;
  rating: number;
  rating_count: number;
  completed_jobs: number;
  hourly_rate: number;
  available: boolean;
  distance_km: number | null;
  skills: string[];
}

export interface Report {
  id: string;
  job_id: string;
  job_title: string;
  job_status: JobStatus;
  reporter_id: string;
  reporter_name: string;
  reporter_role: "client" | "worker";
  type: string;
  description: string;
  status: "open" | "resolved" | "rejected";
  resolution: string | null;
  refund_amount: number | null;
  resolution_note: string;
  held_amount: number | null;
  created_at: string;
  resolved_at: string | null;
}

export interface AdminPayment extends Payment {
  job_id: string;
  job_title: string;
  client_name: string;
  worker_name: string;
}

export interface Stats {
  jobs_posted: number;
  jobs_accepted: number;
  jobs_completed: number;
  jobs_cancelled: number;
  acceptance_rate: number;
  completion_rate: number;
  active_workers: number;
  available_workers: number;
  active_clients: number;
  users: number;
  open_reports: number;
  pending_withdrawals: number;
  gmv: number;
  commission: number;
  avg_minutes_to_assignment: number | null;
  daily: { day: string; posted: number; completed: number; gmv: number }[];
  by_category: { name: string; icon: string; jobs: number; done: number }[];
}

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

const TOKEN_KEY = "microjob.token";

export const tokenStore = {
  get: (): string | null => {
    try {
      return localStorage.getItem(TOKEN_KEY);
    } catch {
      return null;
    }
  },
  set: (t: string | null) => {
    try {
      if (t) localStorage.setItem(TOKEN_KEY, t);
      else localStorage.removeItem(TOKEN_KEY);
    } catch {
      /* storage unavailable */
    }
  },
};

let onUnauthorized: () => void = () => {};
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn;
}

export async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  const token = tokenStore.get();
  if (token) headers.Authorization = `Bearer ${token}`;
  let payload: BodyInit | undefined;
  if (body instanceof FormData) {
    payload = body;
  } else if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const res = await fetch(API_URL + path, { method, headers, body: payload });
  if (res.status === 401 && token) onUnauthorized();
  if (res.status === 204) return undefined as T;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new ApiError(res.status, data.error ?? res.statusText);
  return data as T;
}

const get = <T>(p: string) => request<T>("GET", p);
const post = <T>(p: string, b?: unknown) => request<T>("POST", p, b);
const put = <T>(p: string, b?: unknown) => request<T>("PUT", p, b);
const patch = <T>(p: string, b?: unknown) => request<T>("PATCH", p, b);

export const mediaUrl = (u: string) => (u.startsWith("/") ? API_URL + u : u);

export const api = {
  register: (b: Record<string, unknown>) => post<{ token: string; user: User }>("/api/auth/register", b),
  login: (identifier: string, password: string) => post<{ token: string; user: User }>("/api/auth/login", { identifier, password }),
  me: () => get<User>("/api/me"),
  updateMe: (b: Partial<User>) => patch<User>("/api/me", b),
  updateWorker: (b: { headline?: string; hourly_rate?: number; radius_km?: number; available?: boolean; skill_ids?: number[] }) =>
    put<User>("/api/me/worker", b),
  setAvailability: (b: { available: boolean; latitude?: number; longitude?: number; zone?: string }) => put<User>("/api/me/availability", b),
  profile: (id: string) => get<{ user: User; reviews: Review[] }>(`/api/users/${id}`),
  categories: () => get<Category[]>("/api/categories"),
  searchWorkers: (q: string, lat?: number, lng?: number, categoryId?: number) => {
    const p = new URLSearchParams({ q });
    if (lat !== undefined && lng !== undefined) {
      p.set("lat", String(lat));
      p.set("lng", String(lng));
    }
    if (categoryId) p.set("category_id", String(categoryId));
    return get<WorkerSearchResult[]>(`/api/workers/search?${p}`);
  },
  nearbyCount: (lat: number, lng: number, radiusKm = 5) =>
    get<{ count: number; radius_km: number }>(`/api/workers/nearby-count?lat=${lat}&lng=${lng}&radius_km=${radiusKm}`),

  createJob: (b: Record<string, unknown>) => post<{ job: Job; notified_workers: number }>("/api/jobs", b),
  myJobs: (scope: "active" | "done" | "cancelled") => get<Job[]>(`/api/jobs/mine?scope=${scope}`),
  missions: (scope: "active" | "done" | "applied") => get<Job[]>(`/api/jobs/missions?scope=${scope}`),
  availableJobs: () => get<Job[]>("/api/jobs/available"),
  job: (id: string) => get<Job>(`/api/jobs/${id}`),
  accept: (id: string) => post<Job>(`/api/jobs/${id}/accept`),
  decline: (id: string) => post<void>(`/api/jobs/${id}/decline`),
  applications: (id: string) => get<Application[]>(`/api/jobs/${id}/applications`),
  apply: (id: string, price: number, message: string) => post<Job>(`/api/jobs/${id}/applications`, { price, message }),
  choose: (id: string, appId: string) => post<Job>(`/api/jobs/${id}/applications/${appId}/choose`),
  advance: (id: string, status: JobStatus) => post<Job>(`/api/jobs/${id}/status`, { status }),
  confirm: (id: string) => post<Job>(`/api/jobs/${id}/confirm`),
  cancel: (id: string, reason: string) => post<Job>(`/api/jobs/${id}/cancel`, { reason }),
  shareLocation: (id: string, latitude: number, longitude: number) => put<void>(`/api/jobs/${id}/location`, { latitude, longitude }),
  messages: (id: string) => get<Message[]>(`/api/jobs/${id}/messages`),
  sendMessage: (id: string, b: Partial<Message>) => post<Message>(`/api/jobs/${id}/messages`, b),
  review: (id: string, b: Record<string, unknown>) => post<Job>(`/api/jobs/${id}/reviews`, b),
  report: (id: string, type: string, description: string) => post<Job>(`/api/jobs/${id}/reports`, { type, description }),
  conversations: () => get<Conversation[]>("/api/conversations"),

  wallet: () => get<Wallet>("/api/wallet"),
  withdraw: (amount: number, method: string, destination: string) => post<Wallet>("/api/wallet/withdrawals", { amount, method, destination }),
  notifications: () => get<{ items: Notification[]; unread: number }>("/api/notifications"),
  markRead: (ids: string[] = []) => post<void>("/api/notifications/read", { ids }),
  upload: (file: File) => {
    const fd = new FormData();
    fd.append("file", file);
    return post<{ url: string }>("/api/uploads", fd);
  },

  admin: {
    stats: () => get<Stats>("/api/admin/stats"),
    users: (q = "", role = "") => get<User[]>(`/api/admin/users?q=${encodeURIComponent(q)}&role=${role}`),
    updateUser: (id: string, b: Record<string, unknown>) => patch<User>(`/api/admin/users/${id}`, b),
    verifySkill: (userId: string, skillId: number, verified: boolean) =>
      post<User>(`/api/admin/users/${userId}/skills/${skillId}/verify`, { verified }),
    jobs: (status = "", q = "") => get<Job[]>(`/api/admin/jobs?status=${status}&q=${encodeURIComponent(q)}`),
    cancelJob: (id: string, reason: string) => post<Job>(`/api/admin/jobs/${id}/cancel`, { reason }),
    categories: () => get<Category[]>("/api/admin/categories"),
    createCategory: (b: Record<string, unknown>) => post<Category[]>("/api/admin/categories", b),
    updateCategory: (id: number, b: Record<string, unknown>) => patch<Category[]>(`/api/admin/categories/${id}`, b),
    createSkill: (categoryId: number, name: string) => post<Category[]>(`/api/admin/categories/${categoryId}/skills`, { name }),
    updateSkill: (id: number, b: Record<string, unknown>) => patch<Category[]>(`/api/admin/skills/${id}`, b),
    reports: (status = "") => get<Report[]>(`/api/admin/reports?status=${status}`),
    resolveReport: (id: string, resolution: string, refund_amount: number, note: string) =>
      post<Report[]>(`/api/admin/reports/${id}/resolve`, { resolution, refund_amount, note }),
    payments: (status = "") => get<AdminPayment[]>(`/api/admin/payments?status=${status}`),
    withdrawals: (status = "") => get<Withdrawal[]>(`/api/admin/withdrawals?status=${status}`),
    processWithdrawal: (id: string, status: "paid" | "rejected") => post<Withdrawal[]>(`/api/admin/withdrawals/${id}`, { status }),
    settings: () => get<Record<string, string>>("/api/admin/settings"),
    updateSettings: (b: Record<string, string>) => put<Record<string, string>>("/api/admin/settings", b),
  },
};
