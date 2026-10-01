import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import "./index.css";
import Layout from "./components/Layout";
import { SessionProvider, useSession } from "./lib/session";
import { Login, Register } from "./pages/Auth";
import Home from "./pages/Home";
import PostJob from "./pages/PostJob";
import JobDetail from "./pages/JobDetail";
import { MyJobs, Missions } from "./pages/Lists";
import Messages from "./pages/Messages";
import WalletPage from "./pages/Wallet";
import Profile from "./pages/Profile";
import PublicProfile from "./pages/PublicProfile";
import Search from "./pages/Search";
import Notifications from "./pages/Notifications";
import AdminLayout from "./pages/admin/AdminLayout";
import AdminDashboard from "./pages/admin/Dashboard";
import AdminUsers from "./pages/admin/Users";
import AdminJobs from "./pages/admin/Jobs";
import AdminCategories from "./pages/admin/Categories";
import AdminReports from "./pages/admin/Reports";
import AdminPayments from "./pages/admin/Payments";

function Protected() {
  const { user, loading } = useSession();
  if (loading) return <p className="p-8 text-center text-slate-500">Chargement…</p>;
  if (!user) return <Navigate to="/login" replace />;
  return <Layout />;
}

function AdminOnly() {
  const { user } = useSession();
  if (!user?.is_admin) return <Navigate to="/" replace />;
  return <AdminLayout />;
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <SessionProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route element={<Protected />}>
            <Route index element={<Home />} />
            <Route path="post" element={<PostJob />} />
            <Route path="jobs" element={<MyJobs />} />
            <Route path="jobs/:id" element={<JobDetail />} />
            <Route path="missions" element={<Missions />} />
            <Route path="messages" element={<Messages />} />
            <Route path="wallet" element={<WalletPage />} />
            <Route path="profile" element={<Profile />} />
            <Route path="users/:id" element={<PublicProfile />} />
            <Route path="search" element={<Search />} />
            <Route path="notifications" element={<Notifications />} />
            <Route path="admin" element={<AdminOnly />}>
              <Route index element={<AdminDashboard />} />
              <Route path="users" element={<AdminUsers />} />
              <Route path="jobs" element={<AdminJobs />} />
              <Route path="categories" element={<AdminCategories />} />
              <Route path="reports" element={<AdminReports />} />
              <Route path="payments" element={<AdminPayments />} />
            </Route>
          </Route>
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </SessionProvider>
  </StrictMode>,
);
