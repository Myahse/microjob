import { useEffect, useRef, useState, type FormEvent } from "react";
import { api, mediaUrl, type Message } from "../lib/api";
import { currentPosition } from "../lib/format";
import { useLoad, useRealtime, useSession } from "../lib/session";
import { Button, ErrorText, Input } from "./ui";

export default function Chat({ jobId, closed }: { jobId: string; closed?: boolean }) {
  const { user } = useSession();
  const msgs = useLoad(() => api.messages(jobId), [jobId]);
  const [text, setText] = useState("");
  const [error, setError] = useState<string | null>(null);
  const bottom = useRef<HTMLDivElement>(null);

  useRealtime(
    (e) => {
      if (e.type === "message" && e.data.job_id === jobId) {
        msgs.setData((list) => (list && !list.some((m) => m.id === e.data.id) ? [...list, e.data as Message] : list));
      }
      if (e.type === "job" && e.data.job_id === jobId) msgs.reload();
    },
    [jobId],
  );

  useEffect(() => bottom.current?.scrollIntoView({ behavior: "smooth" }), [msgs.data?.length]);

  const send = async (b: Partial<Message>) => {
    setError(null);
    try {
      const m = await api.sendMessage(jobId, b);
      msgs.setData((list) => (list && !list.some((x) => x.id === m.id) ? [...list, m] : list));
    } catch (e) {
      setError((e as Error).message);
    }
  };

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!text.trim()) return;
    send({ kind: "text", body: text });
    setText("");
  };

  const photo = async (files: FileList | null) => {
    if (!files?.[0]) return;
    try {
      const { url } = await api.upload(files[0]);
      send({ kind: "image", attachment_url: url });
    } catch (e) {
      setError((e as Error).message);
    }
  };

  const location = async () => {
    try {
      const p = await currentPosition();
      send({ kind: "location", latitude: p.lat, longitude: p.lng });
    } catch (e) {
      setError((e as Error).message);
    }
  };

  return (
    <div className="flex h-[28rem] flex-col rounded-xl border border-slate-200 bg-white">
      <div className="flex-1 space-y-2 overflow-y-auto p-3">
        {msgs.error && <ErrorText error={msgs.error} />}
        {msgs.data?.map((m) => {
          if (m.kind === "system")
            return (
              <p key={m.id} className="mx-auto max-w-[90%] rounded-lg bg-slate-100 px-3 py-1.5 text-center text-xs text-slate-600">
                {m.body}
              </p>
            );
          const mine = m.sender_id === user?.id;
          return (
            <div key={m.id} className={`flex ${mine ? "justify-end" : "justify-start"}`}>
              <div className={`max-w-[75%] rounded-2xl px-3 py-2 text-sm ${mine ? "bg-indigo-600 text-white" : "bg-slate-100 text-slate-900"}`}>
                {m.kind === "image" && <img src={mediaUrl(m.attachment_url)} alt="" className="max-h-52 rounded-lg" />}
                {m.kind === "location" && (
                  <a className="underline" target="_blank" rel="noreferrer" href={`https://www.openstreetmap.org/?mlat=${m.latitude}&mlon=${m.longitude}#map=17/${m.latitude}/${m.longitude}`}>
                    📍 Voir la position
                  </a>
                )}
                {m.body && <p className="whitespace-pre-wrap">{m.body}</p>}
                <p className={`mt-0.5 text-[10px] ${mine ? "text-indigo-200" : "text-slate-400"}`}>
                  {new Date(m.created_at).toLocaleTimeString("fr-FR", { hour: "2-digit", minute: "2-digit" })}
                </p>
              </div>
            </div>
          );
        })}
        <div ref={bottom} />
      </div>
      {closed ? (
        <p className="border-t p-3 text-center text-xs text-slate-500">Conversation fermée</p>
      ) : (
        <form onSubmit={submit} className="flex items-center gap-2 border-t p-2">
          <label className="cursor-pointer rounded-lg p-2 hover:bg-slate-100" title="Photo">
            📷
            <input type="file" accept="image/*" className="hidden" onChange={(e) => photo(e.target.files)} />
          </label>
          <button type="button" onClick={location} className="rounded-lg p-2 hover:bg-slate-100" title="Position">
            📍
          </button>
          <Input value={text} onChange={(e) => setText(e.target.value)} placeholder="Votre message…" />
          <Button>Envoyer</Button>
        </form>
      )}
      {error && <div className="p-2"><ErrorText error={error} /></div>}
    </div>
  );
}
