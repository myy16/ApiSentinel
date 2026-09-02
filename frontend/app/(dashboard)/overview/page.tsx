"use client";

import React from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { useActiveProject } from "../../../contexts/ProjectContext";
import { apiFetch } from "../../../lib/api";
import { Project, Endpoint, DeliveryKPIs } from "@apisentinel/shared";
import {
  SendHorizonal,
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
  Clock,
  Plus,
  Play,
  Terminal,
  History,
  ShieldCheck,
} from "lucide-react";

export default function OverviewPage() {
  const { user, organization, accessToken } = useAuth();
  const { projects, activeProjectId } = useActiveProject();

  // 1. Fetch Endpoints
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

  // 2. Fetch Delivery KPIs
  const { data: deliveryKPIs } = useQuery({
    queryKey: ["delivery-kpis", activeProjectId],
    queryFn: () =>
      apiFetch<DeliveryKPIs>(`/api/projects/${activeProjectId}/delivery-kpis`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
    refetchInterval: 5000,
  });

  // 3. Fetch Security Stats
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

  const successRate = deliveryKPIs ? deliveryKPIs.successRate.toFixed(1) : "100.0";
  const dlqBacklog = deliveryKPIs?.dlqBacklog ?? 0;
  const retryInFlight = deliveryKPIs?.retryWait ?? 0;

  return (
    <div className="space-y-8">
      {/* Welcome Banner */}
      <div className="relative overflow-hidden rounded-2xl border border-border bg-gradient-to-r from-card via-card/90 to-primary/10 p-6 md:p-8 shadow-sm">
        <div className="max-w-2xl space-y-3">
          <div className="inline-flex items-center gap-2 rounded-full border border-primary/30 bg-primary/10 px-3 py-1 text-xs font-semibold text-primary backdrop-blur">
            <SendHorizonal className="h-3.5 w-3.5" />
            <span>Webhook Delivery Security Platform</span>
          </div>
          <h1 className="text-2xl md:text-3xl font-extrabold tracking-tight">
            Hoş Geldiniz, {user?.email?.split("@")[0]} 👋
          </h1>
          <p className="text-xs md:text-sm text-muted-foreground leading-relaxed">
            {organization?.name || "ApiSentinel"} organizasyonundaki webhook teslimatlarını, upstream güvenilirlik metriklerini ve DLQ kurtarma akışlarını yönetin.
          </p>
          <div className="pt-2 flex flex-wrap items-center gap-3">
            <Link
              href="/deliveries"
              className="inline-flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-xs font-bold text-primary-foreground shadow-sm transition hover:bg-primary/90"
            >
              <SendHorizonal className="h-4 w-4" />
              <span>Delivery Control Plane</span>
            </Link>
            <Link
              href="/endpoints"
              className="inline-flex items-center gap-2 rounded-xl border border-border bg-background px-4 py-2 text-xs font-semibold text-foreground shadow-sm transition hover:bg-secondary"
            >
              <Plus className="h-3.5 w-3.5" />
              <span>Endpoint Tanımla</span>
            </Link>
            <Link
              href="/requests"
              className="inline-flex items-center gap-2 rounded-xl border border-border bg-background px-4 py-2 text-xs font-semibold text-foreground shadow-sm transition hover:bg-secondary"
            >
              <Radio className="h-3.5 w-3.5 text-emerald-400" />
              <span>Canlı İstek Akışı</span>
            </Link>
          </div>
        </div>
      </div>

      {/* Core Delivery & Security Metrics */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {/* Delivery Success Rate */}
        <div className="rounded-2xl border border-border bg-card p-5 shadow-sm glow-card">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              Teslimat Başarısı
            </span>
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400">
              <Activity className="h-4 w-4" />
            </div>
          </div>
          <p className="mt-3 text-3xl font-black text-foreground">{successRate}%</p>
          <span className="text-xs text-muted-foreground mt-1 block">
            {deliveryKPIs?.delivered ?? 0} başarılı / {deliveryKPIs?.totalDeliveries ?? 0} toplam iletim
          </span>
        </div>

        {/* DLQ Backlog */}
        <div className="rounded-2xl border border-border bg-card p-5 shadow-sm glow-card">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              DLQ Backlog
            </span>
            <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${dlqBacklog > 0 ? "bg-rose-500/10 text-rose-400" : "bg-emerald-500/10 text-emerald-400"}`}>
              <AlertOctagon className="h-4 w-4" />
            </div>
          </div>
          <p className="mt-3 text-3xl font-black text-foreground">{dlqBacklog}</p>
          <span className={`text-xs font-semibold mt-1 block ${dlqBacklog > 0 ? "text-rose-400" : "text-emerald-400"}`}>
            {dlqBacklog > 0 ? `${dlqBacklog} Başarısız Webhook Kurtarma Bekliyor` : "DLQ Temiz — Sıfır Kayıp"}
          </span>
        </div>

        {/* In-Flight Retries */}
        <div className="rounded-2xl border border-border bg-card p-5 shadow-sm glow-card">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              Retry In-Flight
            </span>
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-amber-500/10 text-amber-400">
              <Clock className="h-4 w-4" />
            </div>
          </div>
          <p className="mt-3 text-3xl font-black text-foreground">{retryInFlight}</p>
          <span className="text-xs text-muted-foreground mt-1 block">
            Exponential backoff bekleyen kuyruk
          </span>
        </div>

        {/* Security Threats Blocked */}
        <div className="rounded-2xl border border-border bg-card p-5 shadow-sm glow-card">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
              Güvenlik Tehditleri
            </span>
            <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${totalFindings > 0 ? "bg-purple-500/10 text-purple-400" : "bg-emerald-500/10 text-emerald-400"}`}>
              <ShieldAlert className="h-4 w-4" />
            </div>
          </div>
          <p className="mt-3 text-3xl font-black text-foreground">{totalFindings}</p>
          <span className={`text-xs font-semibold mt-1 block ${criticalFindings > 0 ? "text-rose-400" : "text-emerald-400"}`}>
            {criticalFindings > 0 ? `${criticalFindings} Kritik Tehdit Engellendi` : "Tüm Politikalar Aktif"}
          </span>
        </div>
      </div>

      {/* Primary Delivery Control Banner */}
      <div className="rounded-2xl border border-primary/20 bg-primary/5 p-6 flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <SendHorizonal className="h-5 w-5 text-primary" />
            <h2 className="text-base font-bold text-foreground">Webhook Delivery Control Plane</h2>
          </div>
          <p className="text-xs text-muted-foreground">
            Upstream backend teslimatlarınızı, deneme geçmişini ve güvenli tek tıkla Replay işlemlerini yönetin.
          </p>
        </div>
        <Link
          href="/deliveries"
          className="inline-flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-xs font-bold text-primary-foreground shadow transition hover:bg-primary/90 shrink-0 self-start md:self-auto"
        >
          <span>Teslimat Ekranına Git</span>
          <ArrowRight className="h-3.5 w-3.5" />
        </Link>
      </div>

      {/* Quick Access Grid */}
      <div className="space-y-4">
        <h2 className="text-base font-bold text-foreground">Operasyon ve Güvenlik Modülleri</h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Link
            href="/deliveries"
            className="flex items-center gap-3 rounded-2xl border border-border bg-card p-4 hover:border-primary/50 transition group glow-card"
          >
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-400 group-hover:scale-105 transition">
              <SendHorizonal className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-xs font-bold text-foreground">Delivery Timeline</h3>
              <p className="text-[11px] text-muted-foreground">Uçtan uca iletim ve deneme telemetrisi</p>
            </div>
          </Link>

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
            href="/audit"
            className="flex items-center gap-3 rounded-2xl border border-border bg-card p-4 hover:border-primary/50 transition group glow-card"
          >
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-purple-500/10 text-purple-400 group-hover:scale-105 transition">
              <History className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-xs font-bold text-foreground">Audit Trail</h3>
              <p className="text-[11px] text-muted-foreground">Replay ve güvenlik denetim kayıtları</p>
            </div>
          </Link>

          <Link
            href="/alerts"
            className="flex items-center gap-3 rounded-2xl border border-border bg-card p-4 hover:border-primary/50 transition group glow-card"
          >
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-amber-500/10 text-amber-400 group-hover:scale-105 transition">
              <ShieldCheck className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-xs font-bold text-foreground">Alarm Kanalları</h3>
              <p className="text-[11px] text-muted-foreground">Slack, Discord ve Webhook bildirimleri</p>
            </div>
          </Link>
        </div>
      </div>
    </div>
  );
}
