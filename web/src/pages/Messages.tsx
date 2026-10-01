import { Link } from "react-router-dom";
import { api } from "../lib/api";
import { STATUS_LABEL, ago } from "../lib/format";
import { useLoad, useRealtime } from "../lib/session";
import { Avatar, Card, Empty, ErrorText, PageTitle } from "../components/ui";

export default function Messages() {
  const convs = useLoad(() => api.conversations());
  useRealtime((e) => e.type === "message" && convs.reload());
  return (
    <div className="mx-auto max-w-3xl">
      <PageTitle>Messages</PageTitle>
      <ErrorText error={convs.error} />
      <div className="space-y-2">
        {convs.data?.length
          ? convs.data.map((c) => (
              <Link key={c.job_id} to={`/jobs/${c.job_id}`}>
                <Card className="flex items-center gap-3 hover:border-indigo-300">
                  <Avatar url={c.other.avatar_url} name={c.other.first_name} />
                  <div className="min-w-0 flex-1">
                    <div className="flex justify-between gap-2">
                      <p className="font-semibold">
                        {c.other.first_name} {c.other.last_name}
                      </p>
                      {c.last_at && <span className="text-xs text-slate-400">{ago(c.last_at)}</span>}
                    </div>
                    <p className="truncate text-xs text-slate-500">
                      {c.job_title} · {STATUS_LABEL[c.job_status]}
                      {c.closed && " · fermée"}
                    </p>
                    <p className="truncate text-sm text-slate-700">{c.last_message}</p>
                  </div>
                </Card>
              </Link>
            ))
          : <Empty>Une conversation s'ouvre automatiquement quand un job est attribué.</Empty>}
      </div>
    </div>
  );
}
