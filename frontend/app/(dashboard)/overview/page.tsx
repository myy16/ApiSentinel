"use client";

import React from "react";
import Link from "next/link";
import { useAuth } from "../../../hooks/useAuth";
import {
  ShieldCheck,
  Radio,
  FolderGit2,
  ShieldAlert,
  ArrowRight,
  Zap,
  Activity,
} from "lucide-react";

export default function OverviewPage() {
  const { user, organization } = useAuth();

  return (
    <div className="space-y-8">
      {/* Welcome Banner */}
      <div className="rounded-2xl border border-border bg-gradient-to-r from-card via-card/80 to-primary/10 p-8 shadow-sm">
        <div className="max-w-2xl space-y-3">
          <div className="inline-flex items-center gap-2 rounded-full border border-border bg-background/50 px-3 py-1 text-xs font-semibold text-primary backdrop-blur">
            <Zap className="h-3.5 w-3.5" />
            <span>ApiSentinel Developer Security Console</span>
          </div>
          <h1 className="text-3xl font-extrabold tracking-tight">
            Hoş Geldiniz, {user?.email?.split("@")[0]}
          </h1>
          <p className="text-sm text-muted-foreground">
            {organization?.name} organizasyonundaki API endpoint'lerinizi, webhook trafiğinizi ve güvenlik politikalarınızı buradan yönetebilirsiniz.
          </p>
          <div className="pt-2">
            <Link
              href="/projects"
              className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90"
            >
              <span>Projeleri Görüntüle</span>
              <ArrowRight className="h-4 w-4" />
            </Link>
          </div>
        </div>
      </div>

      {/* Metrics Cards */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Toplam İstek
            </span>
            <Radio className="h-4 w-4 text-primary" />
          </div>
          <p className="mt-4 text-3xl font-bold">0</p>
          <span className="text-xs text-muted-foreground">Canlı webhook trafiği</span>
        </div>

        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Güvenlik Bulguları
            </span>
            <ShieldAlert className="h-4 w-4 text-emerald-400" />
          </div>
          <p className="mt-4 text-3xl font-bold">0</p>
          <span className="text-xs text-emerald-400">Tehdit bulunamadı</span>
        </div>

        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Engellenen İstekler
            </span>
            <ShieldCheck className="h-4 w-4 text-blue-400" />
          </div>
          <p className="mt-4 text-3xl font-bold">0</p>
          <span className="text-xs text-muted-foreground">Policy Block oranı %0</span>
        </div>

        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Sistem Durumu
            </span>
            <Activity className="h-4 w-4 text-emerald-400" />
          </div>
          <p className="mt-4 text-xl font-bold text-emerald-400">Çevrimiçi</p>
          <span className="text-xs text-muted-foreground">Valkey + Fastify + Postgres</span>
        </div>
      </div>
    </div>
  );
}
