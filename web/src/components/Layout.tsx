import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { useSession } from "../lib/session";
import { Avatar } from "./ui";

const linkCls = ({ isActive }: { isActive: boolean }) =>
  `rounded-lg px-3 py-2 text-sm font-medium ${isActive ? "bg-indigo-50 text-indigo-700" : "text-slate-600 hover:bg-slate-100"}`;

export default function Layout() {
  const { user, unread, signOut } = useSession();
  const navigate = useNavigate();
  if (!user) return null;
  const links: [string, string][] = [["/", "Accueil"]];
  if (user.is_client) links.push(["/jobs", "Mes jobs"]);
  if (user.is_worker) links.push(["/missions", "Missions"], ["/wallet", "Revenus"]);
  links.push(["/search", "Trouver"], ["/messages", "Messages"]);
  if (user.is_admin) links.push(["/admin", "Admin"]);

  return (
    <div className="min-h-screen bg-slate-50">
      <header className="sticky top-0 z-10 border-b border-slate-200 bg-white/95 backdrop-blur">
        <div className="mx-auto flex max-w-6xl items-center gap-2 px-4 py-2">
          <NavLink to="/" className="mr-2 text-lg font-extrabold text-indigo-600">
            MicroJob
          </NavLink>
          <nav className="hidden flex-1 gap-1 md:flex">
            {links.map(([to, label]) => (
              <NavLink key={to} to={to} end={to === "/"} className={linkCls}>
                {label}
              </NavLink>
            ))}
          </nav>
          <div className="ml-auto flex items-center gap-2">
            <NavLink to="/post" className="rounded-lg bg-indigo-600 px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-700">
              + Poster un job
            </NavLink>
            <NavLink to="/notifications" className="relative rounded-lg p-2 hover:bg-slate-100" aria-label="Notifications">
              🔔
              {unread > 0 && (
                <span className="absolute -right-0.5 -top-0.5 rounded-full bg-rose-600 px-1.5 text-[10px] font-bold text-white">{unread}</span>
              )}
            </NavLink>
            <NavLink to="/profile" className="rounded-full" aria-label="Profil">
              <Avatar url={user.avatar_url} name={user.first_name} size={34} />
            </NavLink>
            <button
              onClick={() => {
                signOut();
                navigate("/login");
              }}
              className="hidden rounded-lg px-2 py-2 text-sm text-slate-500 hover:bg-slate-100 sm:block"
            >
              Déconnexion
            </button>
          </div>
        </div>
        <nav className="flex gap-1 overflow-x-auto px-2 pb-2 md:hidden">
          {links.map(([to, label]) => (
            <NavLink key={to} to={to} end={to === "/"} className={linkCls}>
              {label}
            </NavLink>
          ))}
        </nav>
      </header>
      <main className="mx-auto max-w-6xl px-4 py-6">
        <Outlet />
      </main>
    </div>
  );
}
