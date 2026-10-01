import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import { API_URL, api, setUnauthorizedHandler, tokenStore, type User } from "./api";

export interface RealtimeEvent {
  type: "notification" | "message" | "job";
  data: any;
}

type Listener = (e: RealtimeEvent) => void;

interface Session {
  user: User | null;
  loading: boolean;
  signIn: (token: string, user: User) => void;
  signOut: () => void;
  setUser: (u: User) => void;
  refresh: () => Promise<void>;
  subscribe: (fn: Listener) => () => void;
  unread: number;
  setUnread: (n: number | ((n: number) => number)) => void;
}

const Ctx = createContext<Session | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [token, setToken] = useState<string | null>(tokenStore.get());
  const [unread, setUnread] = useState(0);
  const listeners = useRef(new Set<Listener>());

  const signOut = useCallback(() => {
    tokenStore.set(null);
    setToken(null);
    setUser(null);
  }, []);

  useEffect(() => setUnauthorizedHandler(signOut), [signOut]);

  const refresh = useCallback(async () => {
    if (!tokenStore.get()) {
      setLoading(false);
      return;
    }
    try {
      setUser(await api.me());
      const n = await api.notifications();
      setUnread(n.unread);
    } catch {
      signOut();
    } finally {
      setLoading(false);
    }
  }, [signOut]);

  useEffect(() => {
    refresh();
  }, [refresh, token]);

  // Realtime: one WebSocket per session, with reconnect.
  useEffect(() => {
    if (!token || !user) return;
    let ws: WebSocket | null = null;
    let closed = false;
    let retry: ReturnType<typeof setTimeout>;
    const connect = () => {
      const base = API_URL || window.location.origin;
      const url = base.replace(/^http/, "ws") + `/api/ws?token=${encodeURIComponent(token)}`;
      ws = new WebSocket(url);
      ws.onmessage = (m) => {
        const e = JSON.parse(m.data) as RealtimeEvent;
        if (e.type === "notification") setUnread((n) => n + 1);
        listeners.current.forEach((fn) => fn(e));
      };
      ws.onclose = () => {
        if (!closed) retry = setTimeout(connect, 3000);
      };
    };
    connect();
    return () => {
      closed = true;
      clearTimeout(retry);
      ws?.close();
    };
  }, [token, user?.id]); // eslint-disable-line react-hooks/exhaustive-deps

  const signIn = (t: string, u: User) => {
    tokenStore.set(t);
    setToken(t);
    setUser(u);
  };

  const subscribe = useCallback((fn: Listener) => {
    listeners.current.add(fn);
    return () => {
      listeners.current.delete(fn);
    };
  }, []);

  return (
    <Ctx.Provider value={{ user, loading, signIn, signOut, setUser, refresh, subscribe, unread, setUnread }}>
      {children}
    </Ctx.Provider>
  );
}

export function useSession() {
  const s = useContext(Ctx);
  if (!s) throw new Error("useSession outside provider");
  return s;
}

/** Calls fn for realtime events matching the filter. */
export function useRealtime(fn: Listener, deps: unknown[] = []) {
  const { subscribe } = useSession();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => subscribe(fn), [subscribe, ...deps]);
}

/** Loads data and reloads it on demand. */
export function useLoad<T>(load: () => Promise<T>, deps: unknown[] = []) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const reload = useCallback(async () => {
    try {
      setData(await load());
      setError(null);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
  useEffect(() => {
    setLoading(true);
    reload();
  }, [reload]);
  return { data, setData, error, loading, reload };
}
