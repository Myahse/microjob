import { NavLink, Outlet } from "react-router-dom";

const links: [string, string][] = [
  ["/admin", "📊 Statistiques"],
  ["/admin/users", "👥 Utilisateurs"],
  ["/admin/jobs", "🧰 Jobs"],
  ["/admin/categories", "🗂 Catégories"],
  ["/admin/reports", "⚖️ Litiges"],
  ["/admin/payments", "💳 Paiements"],
];

export default function AdminLayout() {
  return (
    <div className="grid gap-4 md:grid-cols-[12rem_1fr]">
      <nav className="flex gap-1 overflow-x-auto md:flex-col">
        {links.map(([to, label]) => (
          <NavLink
            key={to}
            to={to}
            end={to === "/admin"}
            className={({ isActive }) =>
              `whitespace-nowrap rounded-lg px-3 py-2 text-sm font-medium ${isActive ? "bg-slate-900 text-white" : "text-slate-600 hover:bg-slate-100"}`
            }
          >
            {label}
          </NavLink>
        ))}
      </nav>
      <div className="min-w-0">
        <Outlet />
      </div>
    </div>
  );
}
