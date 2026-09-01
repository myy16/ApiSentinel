"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import Image from "next/image";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "../../hooks/useAuth";
import { ProjectProvider, useActiveProject } from "../../contexts/ProjectContext";
import { ThemeToggle } from "../../components/ThemeToggle";
import {
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
  User,
  Plus,
  PanelLeftClose,
  PanelLeftOpen,
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
  const [isCollapsed, setIsCollapsed] = useState(false);

  useEffect(() => {
    try {
      const stored = localStorage.getItem("apisentinel_sidebar_collapsed");
      if (stored === "true") {
        setIsCollapsed(true);
      }
    } catch {}
  }, []);

  const toggleSidebar = () => {
    setIsCollapsed((prev) => {
      const next = !prev;
      try {
        localStorage.setItem("apisentinel_sidebar_collapsed", String(next));
      } catch {}
      return next;
    });
  };

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
          <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-gradient-to-br from-indigo-100 via-purple-100 to-violet-100 dark:from-indigo-950/70 dark:via-purple-950/60 dark:to-violet-950/70 border-2 border-indigo-200/90 dark:border-purple-500/40 p-2.5 shadow-md shadow-indigo-500/15">
            <Image
              src="/logo-emblem.png"
              alt="ApiSentinel Emblem"
              width={44}
              height={44}
              className="h-full w-full object-contain animate-pulse drop-shadow-[0_2px_8px_rgba(37,99,235,0.4)]"
              priority
            />
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

      {/* Fixed Sticky Collapsible Sidebar */}
      <aside
        className={`fixed inset-y-0 left-0 z-50 flex h-screen flex-col border-r border-border bg-card/95 backdrop-blur-md transition-all duration-200 ease-in-out md:sticky md:top-0 md:translate-x-0 shrink-0 ${
          mobileOpen ? "translate-x-0 shadow-2xl" : "-translate-x-full"
        } ${isCollapsed ? "md:w-20" : "md:w-64 w-64"}`}
      >
        {/* Brand Header */}
        <div className="flex h-16 items-center border-b border-border shrink-0 px-3">
          {isCollapsed ? (
            <div className="flex w-full items-center justify-center">
              <Link
                href="/overview"
                className="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-indigo-100 via-purple-100 to-violet-100 dark:from-indigo-950/70 dark:via-purple-950/60 dark:to-violet-950/70 border border-indigo-200/90 dark:border-purple-500/40 p-1.5 shadow-sm hover:border-indigo-400 dark:hover:border-purple-400 transition shrink-0"
                title="ApiSentinel — Genel Bakış"
              >
                <Image
                  src="/logo-emblem.png"
                  alt="ApiSentinel Emblem"
                  width={24}
                  height={24}
                  className="h-full w-full object-contain drop-shadow-[0_1px_4px_rgba(37,99,235,0.35)]"
                />
              </Link>
            </div>
          ) : (
            <div className="flex w-full items-center justify-between">
              <Link
                href="/overview"
                className="flex items-center gap-2.5 min-w-0"
                title="ApiSentinel Security Gateway"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-indigo-100 via-purple-100 to-violet-100 dark:from-indigo-950/70 dark:via-purple-950/60 dark:to-violet-950/70 border border-indigo-200/90 dark:border-purple-500/40 p-1.5 shadow-sm hover:border-indigo-400 dark:hover:border-purple-400 transition">
                  <Image
                    src="/logo-emblem.png"
                    alt="ApiSentinel Emblem"
                    width={24}
                    height={24}
                    className="h-full w-full object-contain drop-shadow-[0_1px_4px_rgba(37,99,235,0.35)]"
                  />
                </div>
                <div className="flex flex-col truncate min-w-0">
                  <span className="text-base font-bold tracking-tight text-foreground hover:text-primary transition truncate">
                    ApiSentinel
                  </span>
                  <span className="text-[10px] font-mono text-muted-foreground uppercase tracking-widest truncate">
                    Security Gateway
                  </span>
                </div>
              </Link>

              {/* Mobile Close Button */}
              <button
                onClick={() => setMobileOpen(false)}
                className="md:hidden rounded-lg p-1.5 text-muted-foreground hover:bg-secondary hover:text-foreground"
                aria-label="Menüyü Kapat"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
          )}
        </div>

        {/* Organization Info */}
        <div className="border-b border-border p-2.5 shrink-0">
          <div
            className={`flex items-center gap-2.5 rounded-xl border border-border bg-secondary/30 p-2 ${
              isCollapsed ? "justify-center" : ""
            }`}
            title={`${organization?.name || "Kişisel Organizasyon"} (${user.email})`}
          >
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary border border-primary/20">
              <Building2 className="h-4 w-4" />
            </div>
            {!isCollapsed && (
              <div className="flex flex-col truncate min-w-0">
                <span className="text-xs font-semibold text-foreground truncate">
                  {organization?.name || "Kişisel Organizasyon"}
                </span>
                <span className="text-[10px] text-muted-foreground truncate">{user.email}</span>
              </div>
            )}
          </div>
        </div>

        {/* Scrollable Navigation Links */}
        <nav className="flex-1 space-y-1 overflow-y-auto overflow-x-hidden p-2.5">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = pathname.startsWith(item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                title={isCollapsed ? item.name : undefined}
                className={`group flex items-center rounded-xl py-2 text-xs font-semibold transition ${
                  isCollapsed ? "justify-center px-2" : "justify-between px-3"
                } ${
                  isActive
                    ? "bg-primary text-primary-foreground shadow-sm"
                    : "text-muted-foreground hover:bg-secondary/70 hover:text-foreground"
                }`}
              >
                <div className={`flex items-center gap-3 truncate ${isCollapsed ? "justify-center" : ""}`}>
                  <Icon
                    className={`h-4 w-4 shrink-0 transition ${
                      isActive ? "text-primary-foreground" : "text-muted-foreground group-hover:text-foreground"
                    }`}
                  />
                  {!isCollapsed && <span className="truncate">{item.name}</span>}
                </div>
                {isActive && !isCollapsed && (
                  <span className="h-1.5 w-1.5 rounded-full bg-white animate-pulse shrink-0" />
                )}
              </Link>
            );
          })}
        </nav>

        {/* Fixed Footer: Güvenli Çıkış Yap */}
        <div className="border-t border-border p-2.5 shrink-0 bg-card/50">
          <button
            onClick={async () => {
              await logout();
              router.push("/login");
            }}
            title="Güvenli Çıkış Yap"
            className={`flex w-full items-center rounded-xl py-2 text-xs font-semibold text-destructive transition hover:bg-destructive/10 ${
              isCollapsed ? "justify-center px-2" : "gap-2.5 px-3"
            }`}
          >
            <LogOut className="h-4 w-4 shrink-0" />
            {!isCollapsed && <span>Güvenli Çıkış Yap</span>}
          </button>
        </div>
      </aside>

      {/* Main Content Area */}
      <div className="flex flex-1 flex-col overflow-hidden min-w-0">
        {/* Header */}
        <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-border bg-card/60 px-4 md:px-8 backdrop-blur-md">
          <div className="flex items-center gap-3">
            {/* Mobile Hamburger Toggle */}
            <button
              onClick={() => setMobileOpen(true)}
              className="md:hidden rounded-lg border border-border p-2 text-muted-foreground hover:bg-secondary hover:text-foreground"
              aria-label="Menüyü Aç"
            >
              <Menu className="h-5 w-5" />
            </button>

            {/* Desktop Sidebar Toggle in Main Header */}
            <button
              onClick={toggleSidebar}
              className="hidden md:flex h-8 w-8 items-center justify-center rounded-lg border border-border text-muted-foreground hover:bg-secondary hover:text-foreground transition"
              title={isCollapsed ? "Yan Menüyü Genişlet" : "Yan Menüyü Daralt"}
              aria-label="Yan Menüyü Aç/Kapat"
            >
              {isCollapsed ? <PanelLeftOpen className="h-4 w-4" /> : <PanelLeftClose className="h-4 w-4" />}
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

            {/* Theme Toggle (Light / Dark / System) */}
            <ThemeToggle />

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
