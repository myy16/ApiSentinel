"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "../../hooks/useAuth";
import { ProjectProvider, useActiveProject } from "../../contexts/ProjectContext";
import {
  Shield,
  LayoutDashboard,
  FolderGit2,
  LogOut,
  Building2,
  Loader2,
  Globe,
  Radio,
  ShieldAlert,
  BellRing,
  Repeat,
  Sparkles,
  FileCode,
  Terminal,
  Settings,
  Menu,
  X,
  ChevronRight,
  Activity,
  User,
  ChevronDown,
  Plus,
} from "lucide-react";

const navItems = [
  { name: "Genel Bakış", href: "/overview", icon: LayoutDashboard },
  { name: "Projeler", href: "/projects", icon: FolderGit2 },
  { name: "Endpoints", href: "/endpoints", icon: Globe },
  { name: "Canlı İstekler", href: "/requests", icon: Radio },
  { name: "Güvenlik Bulguları", href: "/security", icon: ShieldAlert },
  { name: "Bildirim Kanalları", href: "/alerts", icon: BellRing },
  { name: "Upstream Forwarding", href: "/forwarding", icon: Repeat },
  { name: "Replay Lab", href: "/replay", icon: Repeat },
  { name: "Mock Lab", href: "/mock", icon: Sparkles },
  { name: "Sözleşmeler", href: "/contracts", icon: FileCode },
  { name: "Local Agent", href: "/agents", icon: Terminal },
  { name: "Ayarlar", href: "/settings", icon: Settings },
];

function GlobalProjectSwitcher() {
  const { projects, activeProjectId, setActiveProjectId, isLoading } = useActiveProject();

  if (isLoading) {
    return (
      <div className="flex items-center gap-1.5 rounded-xl border border-border bg-card px-3 py-1.5 text-xs text-muted-foreground">
        <Loader2 className="h-3 w-3 animate-spin text-primary" />
        <span>Projeler...</span>
      </div>
    );
  }

  if (projects.length === 0) {
    return (
      <Link
        href="/projects"
        className="flex items-center gap-1.5 rounded-xl border border-dashed border-primary/40 bg-primary/5 px-2.5 py-1 text-xs font-semibold text-primary hover:bg-primary/10 transition"
      >
        <Plus className="h-3 w-3" />
        <span>Proje Oluştur</span>
      </Link>
    );
  }

  return (
    <div className="flex items-center gap-1.5 rounded-xl border border-border bg-card/80 px-2.5 py-1 text-xs shadow-sm hover:border-primary/40 transition">
      <FolderGit2 className="h-3.5 w-3.5 text-primary shrink-0" />
      <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground hidden lg:inline">
        Proje:
      </span>
      <select
        value={activeProjectId}
        onChange={(e) => setActiveProjectId(e.target.value)}
        className="bg-transparent font-bold text-foreground text-xs focus:outline-none cursor-pointer pr-1 max-w-[150px] sm:max-w-[200px] truncate"
        title="Aktif Çalışma Projesi (Tüm sayfalarda geçerlidir)"
      >
        {projects.map((p) => (
          <option key={p.id} value={p.id} className="bg-card text-foreground">
            {p.name}
          </option>
        ))}
      </select>
    </div>
  );
}

function DashboardContent({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { user, organization, isLoading, logout } = useAuth();
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    if (!isLoading && !user) {
      router.push("/login");
    }
  }, [user, isLoading, router]);

  // Close mobile sidebar on route change
  useEffect(() => {
    setMobileOpen(false);
  }, [pathname]);

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="flex flex-col items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-primary/20 text-primary">
            <Shield className="h-6 w-6 animate-pulse" />
          </div>
          <Loader2 className="h-5 w-5 animate-spin text-primary" />
          <span className="text-xs font-medium text-muted-foreground">ApiSentinel Yükleniyor...</span>
        </div>
      </div>
    );
  }

  if (!user) {
    return null;
  }

  const activeNavItem = navItems.find((item) => pathname.startsWith(item.href)) || navItems[0];

  return (
    <div className="flex min-h-screen bg-background text-foreground">
      {/* Mobile Drawer Overlay */}
      {mobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/60 backdrop-blur-sm md:hidden"
          onClick={() => setMobileOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside
        className={`fixed inset-y-0 left-0 z-50 flex w-64 flex-col border-r border-border bg-card/95 backdrop-blur-md transition-transform duration-200 ease-in-out md:static md:translate-x-0 ${
          mobileOpen ? "translate-x-0 shadow-2xl" : "-translate-x-full"
        }`}
      >
        {/* Brand */}
        <div className="flex h-16 items-center justify-between border-b border-border px-5">
          <Link href="/overview" className="flex items-center gap-2.5 group">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-primary/30 to-primary/10 border border-primary/30 text-primary shadow-sm group-hover:scale-105 transition">
              <Shield className="h-5 w-5" />
            </div>
            <div className="flex flex-col">
              <span className="text-base font-bold tracking-tight text-foreground group-hover:text-primary transition">
                ApiSentinel
              </span>
              <span className="text-[10px] font-mono text-muted-foreground uppercase tracking-widest">
                Security Gateway
              </span>
            </div>
          </Link>

          <button
            onClick={() => setMobileOpen(false)}
            className="md:hidden rounded-lg p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Organization Badge */}
        <div className="border-b border-border p-3.5">
          <div className="flex items-center gap-2.5 rounded-xl border border-border bg-secondary/30 p-2.5">
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary border border-primary/20">
              <Building2 className="h-4 w-4" />
            </div>
            <div className="flex flex-col truncate min-w-0">
              <span className="text-xs font-semibold text-foreground truncate">
                {organization?.name || "Kişisel Organizasyon"}
              </span>
              <span className="text-[10px] text-muted-foreground truncate">{user.email}</span>
            </div>
          </div>
        </div>

        {/* Nav Links */}
        <nav className="flex-1 space-y-1 overflow-y-auto p-3">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = pathname.startsWith(item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`group flex items-center justify-between rounded-xl px-3 py-2 text-xs font-semibold transition ${
                  isActive
                    ? "bg-primary text-primary-foreground shadow-sm"
                    : "text-muted-foreground hover:bg-secondary/70 hover:text-foreground"
                }`}
              >
                <div className="flex items-center gap-3 truncate">
                  <Icon
                    className={`h-4 w-4 shrink-0 transition ${
                      isActive ? "text-primary-foreground" : "text-muted-foreground group-hover:text-foreground"
                    }`}
                  />
                  <span className="truncate">{item.name}</span>
                </div>
                {isActive && (
                  <span className="h-1.5 w-1.5 rounded-full bg-white animate-pulse" />
                )}
              </Link>
            );
          })}
        </nav>

        {/* User Footer */}
        <div className="border-t border-border p-3">
          <button
            onClick={async () => {
              await logout();
              router.push("/login");
            }}
            className="flex w-full items-center gap-2.5 rounded-xl px-3 py-2 text-xs font-semibold text-destructive transition hover:bg-destructive/10"
          >
            <LogOut className="h-4 w-4" />
            <span>Güvenli Çıkış Yap</span>
          </button>
        </div>
      </aside>

      {/* Main Content Area */}
      <div className="flex flex-1 flex-col overflow-hidden min-w-0">
        {/* Header */}
        <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-border bg-card/60 px-4 md:px-8 backdrop-blur-md">
          <div className="flex items-center gap-3">
            <button
              onClick={() => setMobileOpen(true)}
              className="md:hidden rounded-lg border border-border p-2 text-muted-foreground hover:bg-secondary hover:text-foreground"
              aria-label="Menüyü Aç"
            >
              <Menu className="h-5 w-5" />
            </button>

            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <span className="hidden sm:inline font-medium">Konsol</span>
              <ChevronRight className="hidden sm:inline h-3.5 w-3.5" />
              <span className="font-semibold text-foreground text-sm">{activeNavItem?.name}</span>
            </div>
          </div>

          <div className="flex items-center gap-3">
            {/* Global Project Switcher */}
            <GlobalProjectSwitcher />

            {/* Live Gateway Indicator */}
            <div className="hidden sm:flex items-center gap-2 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-400">
              <span className="relative flex h-2 w-2">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75"></span>
                <span className="relative inline-flex h-2 w-2 rounded-full bg-emerald-500"></span>
              </span>
              <span className="font-mono">Gateway Active</span>
            </div>

            {/* User Quick Badge */}
            <Link
              href="/settings"
              className="flex items-center gap-2 rounded-xl border border-border bg-card px-2.5 py-1 text-xs font-semibold text-foreground hover:border-primary/50 transition"
              title="Profil ve Ayarlar"
            >
              <div className="flex h-6 w-6 items-center justify-center rounded-lg bg-primary/20 text-primary">
                <User className="h-3.5 w-3.5" />
              </div>
              <span className="hidden sm:inline truncate max-w-[120px]">{user.email.split("@")[0]}</span>
            </Link>
          </div>
        </header>

        {/* Main Content Body */}
        <main className="flex-1 overflow-y-auto p-4 sm:p-6 lg:p-8">{children}</main>
      </div>
    </div>
  );
}

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <ProjectProvider>
      <DashboardContent>{children}</DashboardContent>
    </ProjectProvider>
  );
}


