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
            {organization?.name || "ApiSentinel"} organizasyonundaki API endpoint'lerinizi, webhook trafiğinizi ve güvenlik politikalarınızı buradan yönetebilirsiniz.
          </p>
          <div className="pt-2 flex items-center gap-3">
            <Link
              href="/endpoints"
              className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90"
            >
              <span>Endpoint'leri Yönet</span>
              <ArrowRight className="h-4 w-4" />
            </Link>
            <Link
              href="/security"
              className="inline-flex items-center gap-2 rounded-lg border border-border bg-background px-4 py-2 text-sm font-semibold text-foreground shadow-sm transition hover:bg-secondary"
            >
              <span>Güvenlik Raporu</span>
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
          <p className="mt-4 text-3xl font-bold">{totalRequests}</p>
          <span className="text-xs text-muted-foreground">İncelenen Webhook trafiği</span>
        </div>

        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Güvenlik Bulguları
            </span>
            <ShieldAlert className={`h-4 w-4 ${totalFindings > 0 ? "text-rose-400" : "text-emerald-400"}`} />
          </div>
          <p className="mt-4 text-3xl font-bold">{totalFindings}</p>
          <span className={`text-xs ${criticalFindings > 0 ? "text-rose-400 font-semibold" : "text-emerald-400"}`}>
            {criticalFindings > 0 ? `${criticalFindings} Kritik İhlal Mevcut` : "Sistem Güvende"}
          </span>
        </div>

        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Aktif Endpoint'ler
            </span>
            <Server className="h-4 w-4 text-blue-400" />
          </div>
          <p className="mt-4 text-3xl font-bold">{endpoints.length}</p>
          <span className="text-xs text-muted-foreground">Canlı dinlenen webhook kanalı</span>
        </div>

        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground uppercase tracking-wider">
              Aktif Projeler
            </span>
            <FolderGit2 className="h-4 w-4 text-purple-400" />
          </div>
          <p className="mt-4 text-3xl font-bold">{projects.length}</p>
          <span className="text-xs text-muted-foreground">Tanımlı proje havuzu</span>
        </div>
      </div>

      {/* Quick Access Grid */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4 pt-2">
        <Link
          href="/contracts"
          className="flex items-center gap-3 rounded-xl border border-border bg-card p-4 hover:border-primary/50 transition group"
        >
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10 text-blue-400 group-hover:scale-105 transition">
            <FileCode className="h-5 w-5" />
          </div>
          <div>
            <h4 className="text-sm font-semibold text-foreground">JSON Schema</h4>
            <p className="text-xs text-muted-foreground">Sözleşme doğrulama kuralları</p>
          </div>
        </Link>

        <Link
          href="/forwarding"
          className="flex items-center gap-3 rounded-xl border border-border bg-card p-4 hover:border-primary/50 transition group"
        >
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400 group-hover:scale-105 transition">
            <Share2 className="h-5 w-5" />
          </div>
          <div>
            <h4 className="text-sm font-semibold text-foreground">Forwarding & DLQ</h4>
            <p className="text-xs text-muted-foreground">İletim ve kuyruk kurtarma</p>
          </div>
        </Link>

        <Link
          href="/mock"
          className="flex items-center gap-3 rounded-xl border border-border bg-card p-4 hover:border-primary/50 transition group"
        >
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-500/10 text-purple-400 group-hover:scale-105 transition">
            <Sparkles className="h-5 w-5" />
          </div>
          <div>
            <h4 className="text-sm font-semibold text-foreground">Mock Engine</h4>
            <p className="text-xs text-muted-foreground">Geliştirici sahte yanıt motoru</p>
          </div>
        </Link>

        <Link
          href="/alerts"
          className="flex items-center gap-3 rounded-xl border border-border bg-card p-4 hover:border-primary/50 transition group"
        >
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-500/10 text-amber-400 group-hover:scale-105 transition">
            <Activity className="h-5 w-5" />
          </div>
          <div>
            <h4 className="text-sm font-semibold text-foreground">Alarm Kanalları</h4>
            <p className="text-xs text-muted-foreground">Slack, Discord & Webhook</p>
          </div>
        </Link>
      </div>
    </div>
  );
}
