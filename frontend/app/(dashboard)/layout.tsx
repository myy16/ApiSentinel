"use client";

import React, { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "../../hooks/useAuth";
import {
  Shield,
  LayoutDashboard,
  FolderGit2,
  Radio,
  ShieldAlert,
  Repeat,
  Sparkles,
  FileCode,
  Terminal,
  Settings,
  LogOut,
  Building2,
  Loader2,
} from "lucide-react";

const navItems = [
  { name: "Genel Bakış", href: "/overview", icon: LayoutDashboard },
  { name: "Projeler", href: "/projects", icon: FolderGit2 },
  { name: "Canlı İstekler", href: "/requests", icon: Radio },
  { name: "Güvenlik Bulguları", href: "/security", icon: ShieldAlert },
  { name: "Replay Lab", href: "/replay", icon: Repeat },
  { name: "Mock Lab", href: "/mock", icon: Sparkles },
  { name: "Sözleşmeler", href: "/contracts", icon: FileCode },
  { name: "Local Agent", href: "/agents", icon: Terminal },
  { name: "Ayarlar", href: "/settings", icon: Settings },
];

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const { user, organization, isLoading, logout } = useAuth();

  useEffect(() => {
    if (!isLoading && !user) {
      router.push("/login");
    }
  }, [user, isLoading, router]);

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  if (!user) {
    return null;
  }

  return (
    <div className="flex min-h-screen bg-background text-foreground">
      {/* Sidebar */}
      <aside className="sticky top-0 flex h-screen w-64 flex-col border-r border-border bg-card/40 backdrop-blur">
        {/* Brand */}
        <div className="flex h-16 items-center gap-3 border-b border-border px-6">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/20 text-primary">
            <Shield className="h-5 w-5" />
          </div>
          <span className="text-lg font-bold tracking-tight">ApiSentinel</span>
        </div>

        {/* Organization Badge */}
        <div className="border-b border-border p-4">
          <div className="flex items-center gap-2.5 rounded-lg border border-border bg-card p-2.5">
            <div className="flex h-7 w-7 items-center justify-center rounded bg-secondary text-muted-foreground">
              <Building2 className="h-4 w-4" />
            </div>
            <div className="flex flex-col truncate">
              <span className="text-xs font-semibold text-foreground truncate">
                {organization?.name || "Kişisel Organizasyon"}
              </span>
              <span className="text-[10px] text-muted-foreground truncate">{user.email}</span>
            </div>
          </div>
        </div>

        {/* Nav Links */}
        <nav className="flex-1 space-y-1 overflow-y-auto p-4">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = pathname.startsWith(item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition ${
                  isActive
                    ? "bg-primary text-primary-foreground shadow-sm"
                    : "text-muted-foreground hover:bg-secondary hover:text-foreground"
                }`}
              >
                <Icon className="h-4 w-4 shrink-0" />
                <span>{item.name}</span>
              </Link>
            );
          })}
        </nav>

        {/* User Footer */}
        <div className="border-t border-border p-4">
          <button
            onClick={() => {
              logout();
              router.push("/login");
            }}
            className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm font-medium text-destructive transition hover:bg-destructive/10"
          >
            <LogOut className="h-4 w-4" />
            <span>Çıkış Yap</span>
          </button>
        </div>
      </aside>

      {/* Main Content Area */}
      <div className="flex flex-1 flex-col overflow-hidden">
        <header className="flex h-16 items-center justify-between border-b border-border bg-card/20 px-8 backdrop-blur">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold text-muted-foreground">Organizasyon:</span>
            <span className="text-sm font-bold text-foreground">{organization?.name || "Varsayılan"}</span>
          </div>
          <div className="flex items-center gap-4">
            <span className="rounded-full bg-emerald-500/10 px-2.5 py-0.5 text-xs font-semibold text-emerald-400">
              System Online
            </span>
          </div>
        </header>
        <main className="flex-1 overflow-y-auto p-8">{children}</main>
      </div>
    </div>
  );
}
