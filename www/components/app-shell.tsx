"use client";

import {
  BrowserRouter,
  Routes,
  Route,
  Navigate,
  useParams,
} from "react-router-dom";
import { AuthGuard } from "@/components/auth-guard";
import { Navbar } from "@/components/navbar";

// Pages
import LoginPage from "@/components/pages/login";
import ServersPage from "@/components/pages/servers";
import NewServerPage from "@/components/pages/servers-new";
import ServerDetailPage from "@/components/pages/server-detail";
import EditServerPage from "@/components/pages/server-edit";
import NewPeerPage from "@/components/pages/peer-new";
import EditPeerPage from "@/components/pages/peer-edit";
import BackendsPage from "@/components/pages/backends";
import UsersPage from "@/components/pages/users";
import NewUserPage from "@/components/pages/user-new";
import EditUserPage from "@/components/pages/user-edit";

function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthGuard>
      <div className="flex min-h-screen flex-col bg-background">
        <Navbar />
        <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-6 lg:px-6">
          {children}
        </main>
      </div>
    </AuthGuard>
  );
}

function ParamServerDetail() {
  const { id } = useParams();
  return <ServerDetailPage id={id!} />;
}
function ParamServerEdit() {
  const { id } = useParams();
  return <EditServerPage id={id!} />;
}
function ParamNewPeer() {
  const { id } = useParams();
  return <NewPeerPage id={id!} />;
}
function ParamEditPeer() {
  const { id, peerId } = useParams();
  return <EditPeerPage id={id!} peerId={peerId!} />;
}
function ParamEditUser() {
  const { id } = useParams();
  return <EditUserPage id={id!} />;
}

export default function AppShell() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />

        {/* Authenticated routes */}
        <Route
          path="/"
          element={
            <DashboardLayout>
              <Navigate to="/servers" replace />
            </DashboardLayout>
          }
        />
        <Route path="/backends" element={<DashboardLayout><BackendsPage /></DashboardLayout>} />
        <Route path="/servers" element={<DashboardLayout><ServersPage /></DashboardLayout>} />
        <Route path="/servers/new" element={<DashboardLayout><NewServerPage /></DashboardLayout>} />
        <Route path="/servers/:id" element={<DashboardLayout><ParamServerDetail /></DashboardLayout>} />
        <Route path="/servers/:id/edit" element={<DashboardLayout><ParamServerEdit /></DashboardLayout>} />
        <Route path="/servers/:id/peers/new" element={<DashboardLayout><ParamNewPeer /></DashboardLayout>} />
        <Route path="/servers/:id/peers/:peerId/edit" element={<DashboardLayout><ParamEditPeer /></DashboardLayout>} />
        <Route path="/users" element={<DashboardLayout><UsersPage /></DashboardLayout>} />
        <Route path="/users/new" element={<DashboardLayout><NewUserPage /></DashboardLayout>} />
        <Route path="/users/:id/edit" element={<DashboardLayout><ParamEditUser /></DashboardLayout>} />

        {/* Catch-all */}
        <Route path="*" element={<Navigate to="/servers" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
