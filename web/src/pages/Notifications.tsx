import { useEffect } from "react";
import { Link } from "react-router-dom";
import { api } from "../lib/api";
import { ago } from "../lib/format";
import { useLoad, useRealtime, useSession } from "../lib/session";
import { Card, Empty, PageTitle } from "../components/ui";

export default function Notifications() {
  const { setUnread } = useSession();
  const list = useLoad(() => api.notifications());
  useRealtime((e) => e.type === "notification" && list.reload());

  useEffect(() => {
    if (list.data && list.data.unread > 0) {
      api.markRead().then(() => setUnread(0));
    }
  }, [list.data, setUnread]);

  return (
    <div className="mx-auto max-w-2xl">
      <PageTitle>Notifications</PageTitle>
      <div className="space-y-2">
        {list.data?.items.length
          ? list.data.items.map((n) => {
              const body = (
                <Card className={n.read_at ? "" : "border-indigo-300 bg-indigo-50"}>
                  <div className="flex justify-between gap-2">
                    <p className="font-semibold">{n.title}</p>
                    <span className="shrink-0 text-xs text-slate-400">{ago(n.created_at)}</span>
                  </div>
                  {n.body && <p className="text-sm text-slate-600">{n.body}</p>}
                </Card>
              );
              return n.job_id ? (
                <Link key={n.id} to={`/jobs/${n.job_id}`} className="block">
                  {body}
                </Link>
              ) : (
                <div key={n.id}>{body}</div>
              );
            })
          : <Empty>Aucune notification.</Empty>}
      </div>
    </div>
  );
}
