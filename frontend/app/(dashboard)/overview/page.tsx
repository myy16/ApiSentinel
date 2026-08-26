"use client";

import React from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import { Project, Endpoint } from "@apisentinel/shared";
import {
  ShieldCheck,
  Radio,
  FolderGit2,
  ShieldAlert,
  ArrowRight,
  Zap,
  Activity,
  Server,
  Share2,
  FileCode,
  Sparkles,
  AlertOctagon,
  CheckCircle2,
  Bot,
  Plus,
  Play,
  Terminal,
} from "lucide-react";

export default function OverviewPage() {
  const { user, organization, accessToken } = useAuth();

  // 1. Fetch projects
  const { data: projectsData } = useQuery({
    queryKey: ["projects", organization?.id],
    queryFn: () =>
      apiFetch<{ projects: Project[] }>("/api/projects", {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!organization?.id,
  });

  const projects = projectsData?.projects || [];
  const activeProjectId = projects[0]?.id;

  // 2. Fetch endpoints count
  const { data: endpointsData } = useQuery({
    queryKey: ["endpoints", activeProjectId],
    queryFn: () =>
      apiFetch<{ endpoints: (Endpoint & { requestCount: number })[] }>(
        `/api/projects/${activeProjectId}/endpoints`,
        {
          token: accessToken,
          organizationId: organization?.id,
        }
      ),
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
  });

  const endpoints = endpointsData?.endpoints || [];
  const totalRequests = endpoints.reduce((acc, ep) => acc + (ep.requestCount || 0), 0);

  // 3. Fetch security stats
  const { data: statsData } = useQuery({
    queryKey: ["findingsStats", activeProjectId],
    queryFn: () =>
      apiFetch<{ criticalCount: number; highCount: number; mediumCount: number; lowCount: number; totalCount: number }>(
        `/api/projects/${activeProjectId}/findings/stats`,
        {
          token: accessToken,
          organizationId: organization?.id,
        }
      ),
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
  });

  const totalFindings = statsData?.totalCount ?? 0;
  const criticalFindings = statsData?.criticalCount ?? 0;
  const highFindings = statsData?.highCount ?? 0;

  return (
    <div className="space-y-8">
      {/* Welcome Banner */}
      <div className="relative overflow-hidden rounded-2xl border border-border bg-gradient-to-r from-card via-card/90 to-primary/10 p-6 md:p-8 shadow-sm">
        <div className="max-w-2xl space-y-3">
          <div className="inline-flex items-center gap-2 rounded-full border border-border bg-background/60 px-3 py-1 text-xs font-semibold text-primary backdrop-blur">
            <Zap className="h-3.5 w-3.5" />
            <span>ApiSentinel Developer Security Console</span>
          </div>
          <h1 className="text-2xl md:text-3xl font-extrabold tracking-tight">
            Hoş Geldiniz, {user?.email?.split("@")[0]} 👋
          </h1>
          <p className="text-xs md:text-sm text-muted-foreground leading-relaxed">
            {organization?.name || "ApiSentinel"} organizasyonundaki API endpoint'lerinizi, gelen webhook trafiğinizi ve gerçek zamanlı güvenlik politikalarınızı buradan yönetebilirsiniz.
          </p>
          <div className="pt-2 flex flex-wrap items-center gap-3">
            <Link
              href="/endpoints"
              className="inline-flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-xs font-bold text-primary-foreground shadow-sm transition hover:bg-primary/90"
            >
              <Plus className="h-4 w-4" />
              <span>Endpoint Tanımla</span>
            </Link>
            <Link
              href="/requests"
              className="inline-flex items-center gap-2 rounded-xl border border-border bg-background px-4 py-2 text-xs font-semibold text-foreground shadow-sm transition hover:bg-secondary"
            >
              <Radio className="h-3.5 w-3.5 text-emerald-400" />
              <span>Canlı İstek Akışı</span>
            </Link>
            <Link
              href="/security"
              className="inline-flex items-center gap-2 rounded-xl border border-border bg-background px-4 py-2 text-xs font-semibold text-foreground shadow-sm transition hover:bg-secondary"
            >
              <ShieldAlert className="h-3.5 w-3.5 text-rose-400" />
              <span>Güvenlik Raporu</span>
            </Link>
          </div>
        </div>
      </div>

      {/* Metrics Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-2xl border border-border bg-card p-5 shadow-sm glow-card">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              Toplam İstek
            </span>
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-500/10 text-blue-400">
              <Radio className="h-4 w-4" />
            </div>
          </div>
          <p className="mt-3 text-3xl font-black text-foreground">{totalRequests}</p>
          <span className="text-xs text-muted-foreground mt-1 block">İncelenen Webhook trafiği</span>
        </div>

        <div className="rounded-2xl border border-border bg-card p-5 shadow-sm glow-card">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              Güvenlik Tehditleri
            </span>
            <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${totalFindings > 0 ? "bg-rose-500/10 text-rose-400" : "bg-emerald-500/10 text-emerald-400"}`}>
              <ShieldAlert className="h-4 w-4" />
            </div>
          </div>
          <p className="mt-3 text-3xl font-black text-foreground">{totalFindings}</p>
          <span className={`text-xs font-semibold mt-1 block ${criticalFindings > 0 ? "text-rose-400" : "text-emerald-400"}`}>
            {criticalFindings > 0 ? `${criticalFindings} Kritik İhlal Engellendi` : "Sistem Güvende"}
          </span>
        </div>

        <div className="rounded-2xl border border-border bg-card p-5 shadow-sm glow-card">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              Aktif Endpoint'ler
            </span>
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400">
              <Server className="h-4 w-4" />
            </div>
          </div>
          <p className="mt-3 text-3xl font-black text-foreground">{endpoints.length}</p>
          <span className="text-xs text-muted-foreground mt-1 block">Canlı dinlenen webhook kanalı</span>
        </div>

        <div className="rounded-2xl border border-border bg-card p-5 shadow-sm glow-card">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              Aktif Projeler
            </span>
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-purple-500/10 text-purple-400">
              <FolderGit2 className="h-4 w-4" />
            </div>
          </div>
          <p className="mt-3 text-3xl font-black text-foreground">{projects.length}</p>
          <span className="text-xs text-muted-foreground mt-1 block">Tanımlı proje havuzu</span>
        </div>
      </div>

      {/* Quick Access Grid */}
      <div className="space-y-4">
        <h2 className="text-base font-bold text-foreground">Hızlı Erişim Modülleri</h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Link
            href="/contracts"
            className="flex items-center gap-3 rounded-2xl border border-border bg-card p-4 hover:border-primary/50 transition group glow-card"
          >
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-blue-500/10 text-blue-400 group-hover:scale-105 transition">
              <FileCode className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-xs font-bold text-foreground">JSON Schema</h3>
              <p className="text-[11px] text-muted-foreground">Sözleşme doğrulama kuralları</p>
            </div>
          </Link>

          <Link
            href="/forwarding"
            className="flex items-center gap-3 rounded-2xl border border-border bg-card p-4 hover:border-primary/50 transition group glow-card"
          >
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-400 group-hover:scale-105 transition">
              <Share2 className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-xs font-bold text-foreground">Forwarding & DLQ</h3>
              <p className="text-[11px] text-muted-foreground">İletim ve kuyruk kurtarma</p>
            </div>
          </Link>

          <Link
            href="/mock"
            className="flex items-center gap-3 rounded-2xl border border-border bg-card p-4 hover:border-primary/50 transition group glow-card"
          >
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-purple-500/10 text-purple-400 group-hover:scale-105 transition">
              <Sparkles className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-xs font-bold text-foreground">Mock Lab</h3>
              <p className="text-[11px] text-muted-foreground">503/429 Sahte yanıt motoru</p>
            </div>
          </Link>

          <Link
            href="/alerts"
            className="flex items-center gap-3 rounded-2xl border border-border bg-card p-4 hover:border-primary/50 transition group glow-card"
          >
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-amber-500/10 text-amber-400 group-hover:scale-105 transition">
              <Activity className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-xs font-bold text-foreground">Alarm Kanalları</h3>
              <p className="text-[11px] text-muted-foreground">Slack, Discord & Webhook</p>
            </div>
          </Link>
        </div>
      </div>
    </div>
  );
}

