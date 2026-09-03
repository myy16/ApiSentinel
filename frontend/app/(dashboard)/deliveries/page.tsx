"use client";

import React, { useState, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { useActiveProject } from "../../../contexts/ProjectContext";
import { useSSE } from "../../../hooks/useSSE";
import { apiFetch } from "../../../lib/api";
import {
  DeliveryJob,
  DeliveryAttempt,
  DeliveryTimelineData,
  DeliveryKPIs,
  DeliveryState,
  RequestState,
  Endpoint,
  IncidentAnalysis,
} from "@apisentinel/shared";
import {
  SendHorizonal,
  Activity,
  CheckCircle2,
  AlertTriangle,
  Clock,
  RotateCcw,
  Search,
  Filter,
  RefreshCw,
  Eye,
  ShieldCheck,
  ShieldAlert,
  Server,
  Terminal,
  Copy,
  Check,
  X,
  AlertOctagon,
  ArrowRight,
  Globe,
  Loader2,
  Stethoscope,
  Wrench,
  Lightbulb,
  ExternalLink,
  Bot,
  Sparkles,
  Play,
} from "lucide-react";

export default function DeliveriesPage() {
  const { accessToken, organization, user } = useAuth();
  const { activeProjectId } = useActiveProject();
  const queryClient = useQueryClient();

  const [selectedJobId, setSelectedJobId] = useState<string | null>(null);
  const [replayModalJob, setReplayModalJob] = useState<DeliveryJob | null>(null);
  const [overrideIdempotency, setOverrideIdempotency] = useState(false);
  const [justification, setJustification] = useState("");
  const [replayError, setReplayError] = useState<string | null>(null);

  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("ALL");
  const [selectedEndpointId, setSelectedEndpointId] = useState<string>("ALL");
  const [activeAttemptTab, setActiveAttemptTab] = useState<number>(0);
  const [copiedSnippet, setCopiedSnippet] = useState<boolean>(false);
  const [aiAnalysis, setAiAnalysis] = useState<IncidentAnalysis | null>(null);
  const [copiedCurl, setCopiedCurl] = useState<boolean>(false);

  // 1. Fetch Endpoints for filter
  const { data: endpointsData } = useQuery({
    queryKey: ["endpoints", activeProjectId],
    queryFn: () =>
      apiFetch<{ endpoints: Endpoint[] }>(`/api/projects/${activeProjectId}/endpoints`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
  });
  const endpoints = endpointsData?.endpoints || [];

  // 2. Fetch Delivery KPIs
  const { data: kpisData, isLoading: isKPIsLoading } = useQuery({
    queryKey: ["delivery-kpis", activeProjectId],
    queryFn: () =>
      apiFetch<DeliveryKPIs>(`/api/projects/${activeProjectId}/delivery-kpis`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
    refetchInterval: 5000,
  });

  // 3. Fetch Deliveries list for selected endpoint / all
  const {
    data: deliveriesData,
    isLoading: isDeliveriesLoading,
    refetch: refetchDeliveries,
  } = useQuery({
    queryKey: ["deliveries", activeProjectId, selectedEndpointId],
    queryFn: async () => {
      if (selectedEndpointId !== "ALL") {
        const res = await apiFetch<{ deliveries: DeliveryJob[] }>(
          `/api/endpoints/${selectedEndpointId}/deliveries?limit=100`,
          {
            token: accessToken,
            organizationId: organization?.id,
          }
        );
        return res.deliveries || [];
      }
      // If ALL endpoints, gather from each endpoint
      const allJobs: DeliveryJob[] = [];
      for (const ep of endpoints) {
        try {
          const res = await apiFetch<{ deliveries: DeliveryJob[] }>(
            `/api/endpoints/${ep.id}/deliveries?limit=50`,
            {
              token: accessToken,
              organizationId: organization?.id,
            }
          );
          if (res.deliveries) {
            allJobs.push(...res.deliveries);
          }
        } catch {
          // ignore individual fetch errors
        }
      }
      return allJobs.sort(
        (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
      );
    },
    enabled: !!accessToken && !!activeProjectId && !!organization?.id && endpoints.length > 0,
  });

  // 4. Real-time SSE Integration
  const sseQueryKeys = useMemo(
    () => [
      ["deliveries", activeProjectId || "", selectedEndpointId],
      ["delivery-kpis", activeProjectId || ""],
    ],
    [activeProjectId, selectedEndpointId]
  );
  useSSE({
    projectId: activeProjectId,
    token: accessToken,
    organizationId: organization?.id ?? null,
    queryKeys: sseQueryKeys,
    enabled: !!accessToken && !!activeProjectId,
  });

  // 5. Fetch Timeline for selected job
  const { data: timelineData, isLoading: isTimelineLoading } = useQuery({
    queryKey: ["delivery-timeline", selectedJobId],
    queryFn: () =>
      apiFetch<DeliveryTimelineData>(`/api/deliveries/${selectedJobId}/timeline`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!selectedJobId,
  });

  // AI Incident Explainer Mutation (Milestone 15)
  const aiExplainMutation = useMutation({
    mutationFn: (jobId: string) =>
      apiFetch<IncidentAnalysis>(`/api/deliveries/${jobId}/ai-explain`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
      }),
    onSuccess: (data) => {
      setAiAnalysis(data);
    },
  });

  // 6. Safe Replay Mutation
  const replayMutation = useMutation({
    mutationFn: async ({
      jobId,
      override,
      reason,
    }: {
      jobId: string;
      override: boolean;
      reason: string;
    }) => {
      return apiFetch<{ success: boolean; message: string }>(`/api/deliveries/${jobId}/replay`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify({
          overrideIdempotency: override,
          justification: reason,
        }),
      });
    },
    onSuccess: () => {
      setReplayModalJob(null);
      setOverrideIdempotency(false);
      setJustification("");
      setReplayError(null);
      queryClient.invalidateQueries({ queryKey: ["deliveries"] });
      queryClient.invalidateQueries({ queryKey: ["delivery-kpis"] });
      if (selectedJobId) {
        queryClient.invalidateQueries({ queryKey: ["delivery-timeline", selectedJobId] });
      }
    },
    onError: (err: any) => {
      setReplayError(err.message || "Replay işlemi başarısız oldu");
    },
  });

  const allDeliveries = deliveriesData || [];

  // Filter deliveries
  const filteredDeliveries = allDeliveries.filter((job) => {
    if (statusFilter !== "ALL" && job.status !== statusFilter) {
      return false;
    }
    if (searchQuery) {
      const q = searchQuery.toLowerCase();
      return (
        job.id.toLowerCase().includes(q) ||
        job.targetUrl.toLowerCase().includes(q) ||
        (job.idempotencyKey && job.idempotencyKey.toLowerCase().includes(q))
      );
    }
    return true;
  });

  const getStatusBadge = (status: DeliveryState) => {
    switch (status) {
      case "DELIVERED":
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-bold bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <CheckCircle2 className="h-3.5 w-3.5" />
            DELIVERED
          </span>
        );
      case "PROCESSING":
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-bold bg-primary/10 text-primary border border-primary/20 animate-pulse">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            PROCESSING
          </span>
        );
      case "RETRY_WAIT":
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-bold bg-amber-500/10 text-amber-400 border border-amber-500/20">
            <Clock className="h-3.5 w-3.5" />
            RETRY_WAIT
          </span>
        );
      case "DEAD_LETTER":
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-bold bg-rose-500/10 text-rose-400 border border-rose-500/20">
            <AlertOctagon className="h-3.5 w-3.5" />
            DEAD_LETTER (DLQ)
          </span>
        );
      case "PENDING":
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-bold bg-slate-500/10 text-slate-300 border border-slate-500/20">
            <Clock className="h-3.5 w-3.5" />
            PENDING
          </span>
        );
      default:
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-bold bg-muted text-muted-foreground border border-border">
            {status}
          </span>
        );
    }
  };

  return (
    <div className="space-y-6">
      {/* Top Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <div className="inline-flex items-center gap-2 rounded-full border border-primary/30 bg-primary/10 px-3 py-1 text-xs font-semibold text-primary mb-2">
            <SendHorizonal className="h-3.5 w-3.5" />
            <span>Delivery Control Plane</span>
          </div>
          <h1 className="text-2xl font-extrabold tracking-tight">Webhook Teslimat Yönetimi</h1>
          <p className="text-xs md:text-sm text-muted-foreground">
            Doğrulanan webhook'ların upstream sunucularınıza güvenilir iletimini, deneme geçmişini ve DLQ kurtarma akışlarını yönetin.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => refetchDeliveries()}
            className="inline-flex items-center gap-2 px-3 py-2 rounded-xl border border-border bg-card hover:bg-muted text-xs font-semibold transition shadow-sm"
          >
            <RefreshCw className="h-3.5 w-3.5 text-primary" />
            <span>Yenile</span>
          </button>
        </div>
      </div>

      {/* Delivery KPI Summary Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="rounded-2xl border border-border bg-card/60 p-4 backdrop-blur shadow-sm">
          <div className="flex items-center justify-between text-muted-foreground mb-2">
            <span className="text-xs font-bold uppercase tracking-wider">Teslimat Başarısı</span>
            <Activity className="h-4 w-4 text-emerald-400" />
          </div>
          <div className="text-2xl font-black text-foreground">
            {kpisData ? `${kpisData.successRate.toFixed(1)}%` : "100%"}
          </div>
          <p className="text-[11px] text-muted-foreground mt-1">
            Toplam {kpisData?.totalDeliveries || 0} olaydan {kpisData?.delivered || 0} başarılı
          </p>
        </div>

        <div className="rounded-2xl border border-border bg-card/60 p-4 backdrop-blur shadow-sm">
          <div className="flex items-center justify-between text-muted-foreground mb-2">
            <span className="text-xs font-bold uppercase tracking-wider">DLQ Backlog</span>
            <AlertOctagon className={`h-4 w-4 ${(kpisData?.dlqBacklog || 0) > 0 ? "text-rose-400" : "text-emerald-400"}`} />
          </div>
          <div className="text-2xl font-black text-foreground">
            {kpisData?.dlqBacklog || 0}
          </div>
          <p className="text-[11px] text-muted-foreground mt-1">
            Kurtarma bekleyen başarısız teslimat
          </p>
        </div>

        <div className="rounded-2xl border border-border bg-card/60 p-4 backdrop-blur shadow-sm">
          <div className="flex items-center justify-between text-muted-foreground mb-2">
            <span className="text-xs font-bold uppercase tracking-wider">Retry In-Flight</span>
            <Clock className="h-4 w-4 text-amber-400" />
          </div>
          <div className="text-2xl font-black text-foreground">
            {kpisData?.retryWait || 0}
          </div>
          <p className="text-[11px] text-muted-foreground mt-1">
            Exponential backoff bekleyen işler
          </p>
        </div>

        <div className="rounded-2xl border border-border bg-card/60 p-4 backdrop-blur shadow-sm">
          <div className="flex items-center justify-between text-muted-foreground mb-2">
            <span className="text-xs font-bold uppercase tracking-wider">Aktif Endpoints</span>
            <Globe className="h-4 w-4 text-primary" />
          </div>
          <div className="text-2xl font-black text-foreground">
            {endpoints.length}
          </div>
          <p className="text-[11px] text-muted-foreground mt-1">
            Forwarding devrede olan rotalar
          </p>
        </div>
      </div>

      {/* Filter & Search Bar */}
      <div className="flex flex-col md:flex-row gap-3 items-center justify-between bg-card/40 p-3 rounded-2xl border border-border">
        <div className="flex flex-wrap items-center gap-2 w-full md:w-auto">
          {/* Endpoint Filter */}
          <select
            value={selectedEndpointId}
            onChange={(e) => setSelectedEndpointId(e.target.value)}
            className="rounded-xl border border-border bg-card px-3 py-1.5 text-xs font-semibold text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="ALL">Tüm Endpoint'ler ({endpoints.length})</option>
            {endpoints.map((ep) => (
              <option key={ep.id} value={ep.id}>
                {ep.name} (/{ep.slug})
              </option>
            ))}
          </select>

          {/* Status Filter */}
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="rounded-xl border border-border bg-card px-3 py-1.5 text-xs font-semibold text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="ALL">Tüm Durumlar</option>
            <option value="DELIVERED">DELIVERED (Başarılı)</option>
            <option value="RETRY_WAIT">RETRY_WAIT (Yeniden Denenecek)</option>
            <option value="DEAD_LETTER">DEAD_LETTER (DLQ)</option>
            <option value="PENDING">PENDING (Kuyrukta)</option>
          </select>
        </div>

        {/* Text Search */}
        <div className="relative w-full md:w-72">
          <Search className="absolute left-3 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
          <input
            type="text"
            placeholder="Target URL, Request ID veya Idempotency Key..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full rounded-xl border border-border bg-card pl-9 pr-3 py-1.5 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
      </div>

      {/* Deliveries Table & Timeline Split View */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
        {/* Deliveries List Table (Left 7 cols or full) */}
        <div className={selectedJobId ? "lg:col-span-7" : "lg:col-span-12"}>
          <div className="rounded-2xl border border-border bg-card/60 overflow-hidden shadow-sm">
            <div className="px-4 py-3 border-b border-border bg-card/80 flex items-center justify-between">
              <span className="text-xs font-bold text-foreground">
                Teslimat Olayları ({filteredDeliveries.length})
              </span>
              <span className="text-[11px] text-muted-foreground">
                Canlı SSE akışı aktif
              </span>
            </div>

            {isDeliveriesLoading ? (
              <div className="p-12 text-center text-muted-foreground flex flex-col items-center gap-2">
                <Loader2 className="h-6 w-6 animate-spin text-primary" />
                <span className="text-xs">Teslimat kayıtları yükleniyor...</span>
              </div>
            ) : filteredDeliveries.length === 0 ? (
              <div className="p-12 text-center text-muted-foreground">
                <SendHorizonal className="h-8 w-8 mx-auto text-muted-foreground/40 mb-2" />
                <p className="text-sm font-semibold">Henüz teslimat kaydı bulunmuyor</p>
                <p className="text-xs text-muted-foreground mt-1">
                  Endpoint'inize gelen webhook'lar güvenlik kontrolünden geçtikten sonra burada listelenir.
                </p>
              </div>
            ) : (
              <div className="divide-y divide-border overflow-x-auto">
                {filteredDeliveries.map((job) => {
                  const isSelected = selectedJobId === job.id;
                  return (
                    <div
                      key={job.id}
                      onClick={() => setSelectedJobId(job.id)}
                      className={`p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 cursor-pointer transition hover:bg-muted/40 ${
                        isSelected ? "bg-primary/5 border-l-4 border-l-primary" : ""
                      }`}
                    >
                      <div className="space-y-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          {getStatusBadge(job.status)}
                          <span className="text-xs font-mono font-bold text-foreground truncate max-w-[200px]">
                            {job.targetUrl}
                          </span>
                          <span className="text-[10px] px-2 py-0.5 rounded bg-muted text-muted-foreground font-mono">
                            Deneme: {job.attempts}/{job.maxRetries}
                          </span>
                        </div>

                        <div className="flex items-center gap-3 text-[11px] text-muted-foreground">
                          <span>
                            {new Date(job.createdAt).toLocaleTimeString()} · {new Date(job.createdAt).toLocaleDateString()}
                          </span>
                          {job.lastError && (
                            <span className="text-rose-400 truncate max-w-[250px]" title={job.lastError}>
                              Hata: {job.lastError}
                            </span>
                          )}
                        </div>
                      </div>

                      <div className="flex items-center gap-2 shrink-0">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            setSelectedJobId(job.id);
                          }}
                          className="px-2.5 py-1 rounded-lg border border-border bg-card hover:bg-muted text-xs font-semibold text-foreground transition"
                        >
                          Timeline
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            setReplayModalJob(job);
                          }}
                          className="px-2.5 py-1 rounded-lg bg-primary/10 hover:bg-primary/20 text-primary text-xs font-bold transition flex items-center gap-1"
                        >
                          <RotateCcw className="h-3 w-3" />
                          <span>Replay</span>
                        </button>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>

        {/* Timeline & Attempt Inspector Drawer (Right 5 cols) */}
        {selectedJobId && (
          <div className="lg:col-span-5">
            <div className="rounded-2xl border border-border bg-card/80 p-5 space-y-5 shadow-sm sticky top-6">
              <div className="flex items-center justify-between pb-3 border-b border-border">
                <div className="flex items-center gap-2">
                  <Activity className="h-4 w-4 text-primary" />
                  <h2 className="text-sm font-bold text-foreground">Delivery Timeline & Attempt Telemetry</h2>
                </div>
                <button
                  onClick={() => setSelectedJobId(null)}
                  className="p-1 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>

              {isTimelineLoading ? (
                <div className="py-12 text-center text-muted-foreground flex flex-col items-center gap-2">
                  <Loader2 className="h-6 w-6 animate-spin text-primary" />
                  <span className="text-xs">Timeline verisi yükleniyor...</span>
                </div>
              ) : timelineData ? (
                <div className="space-y-6">
                  {/* Smart Diagnostics & Quick Fix Card */}
                  {timelineData.diagnostic && timelineData.diagnostic.category !== "SUCCESS" && (
                    <div className="rounded-2xl border border-rose-500/30 bg-rose-500/5 p-4 space-y-3">
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <Stethoscope className="h-4 w-4 text-rose-400" />
                          <span className="text-xs font-bold text-foreground">Akıllı İletim Teşhisi</span>
                        </div>
                        <span
                          className={`text-[10px] font-extrabold px-2 py-0.5 rounded-full border ${
                            timelineData.diagnostic.severity === "CRITICAL"
                              ? "bg-rose-500/20 text-rose-300 border-rose-500/40"
                              : "bg-amber-500/20 text-amber-300 border-amber-500/40"
                          }`}
                        >
                          {timelineData.diagnostic.category}
                        </span>
                      </div>

                      <div>
                        <div className="text-xs font-bold text-rose-300">
                          {timelineData.diagnostic.title}
                        </div>
                        <p className="text-xs text-muted-foreground mt-1">
                          {timelineData.diagnostic.rootCause}
                        </p>
                      </div>

                      {/* Suggested Action Box */}
                      <div className="rounded-xl border border-border bg-card/80 p-3 space-y-2">
                        <div className="flex items-center gap-1.5 text-xs font-bold text-foreground">
                          <Lightbulb className="h-3.5 w-3.5 text-amber-400" />
                          <span>Önerilen Çözüm Adımı</span>
                        </div>
                        <p className="text-xs text-muted-foreground">
                          {timelineData.diagnostic.suggestedAction}
                        </p>

                        {timelineData.diagnostic.quickFixSnippet && (
                          <div className="mt-2 pt-2 border-t border-border flex items-center justify-between gap-2">
                            <code className="text-[11px] font-mono text-primary truncate">
                              {timelineData.diagnostic.quickFixSnippet}
                            </code>
                            <button
                              onClick={() => {
                                if (timelineData?.diagnostic?.quickFixSnippet) {
                                  navigator.clipboard.writeText(timelineData.diagnostic.quickFixSnippet);
                                  setCopiedSnippet(true);
                                  setTimeout(() => setCopiedSnippet(false), 2000);
                                }
                              }}
                              className="px-2 py-1 rounded-lg bg-secondary hover:bg-muted text-xs font-semibold text-foreground flex items-center gap-1 shrink-0"
                            >
                              {copiedSnippet ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
                              <span>{copiedSnippet ? "Kopyalandı" : "Kopyala"}</span>
                            </button>
                          </div>
                        )}
                      </div>
                    </div>
                  )}

                  {/* AI Incident Explainer & Root-Cause Assistant (Milestone 15) */}
                  <div className="rounded-2xl border border-primary/30 bg-primary/5 p-4 space-y-3">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Bot className="h-4 w-4 text-primary" />
                        <span className="text-xs font-bold text-foreground">AI Kök Neden & Çözüm Asistanı</span>
                      </div>
                      <span className="text-[10px] font-bold text-primary bg-primary/10 px-2 py-0.5 rounded-full border border-primary/20">
                        Gizlilik Korumalı (Sanitized)
                      </span>
                    </div>

                    {!aiAnalysis ? (
                      <div className="flex flex-col items-center justify-center p-4 text-center space-y-2.5 bg-background/50 rounded-xl border border-border">
                        <p className="text-xs text-muted-foreground max-w-xs">
                          Bu iletim hatasını yapay zeka ile analiz ederek anlaşılır Türkçe kök neden ve cURL çözüm rehberi üretin.
                        </p>
                        <button
                          type="button"
                          disabled={aiExplainMutation.isPending}
                          onClick={() => aiExplainMutation.mutate(timelineData.job.id)}
                          className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-xs font-bold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
                        >
                          {aiExplainMutation.isPending ? (
                            <>
                              <Loader2 className="h-3.5 w-3.5 animate-spin" />
                              <span>Arıza Analiz Ediliyor...</span>
                            </>
                          ) : (
                            <>
                              <Sparkles className="h-3.5 w-3.5" />
                              <span>AI İle Kök Neden Analizi Yap</span>
                            </>
                          )}
                        </button>
                      </div>
                    ) : (
                      <div className="space-y-3 text-xs animate-in fade-in duration-150">
                        <div className="p-3 rounded-xl bg-background/90 border border-border space-y-1.5">
                          <div className="flex items-center justify-between">
                            <span className="font-bold text-foreground text-xs">{aiAnalysis.incidentSummary}</span>
                            <span
                              className={`text-[10px] font-bold px-2 py-0.5 rounded border ${
                                aiAnalysis.canSafelyReplay
                                  ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/20"
                                  : "bg-amber-500/10 text-amber-400 border-amber-500/20"
                              }`}
                            >
                              {aiAnalysis.canSafelyReplay ? "✅ Güvenle Replay Edilebilir" : "⚠️ Önce Backend'i Düzeltin"}
                            </span>
                          </div>
                          <p className="text-[11px] text-muted-foreground leading-relaxed">
                            {aiAnalysis.rootCause}
                          </p>
                        </div>

                        {/* Suggested Fix and Action Steps */}
                        <div className="p-3 rounded-xl bg-background/90 border border-border space-y-2">
                          <div className="flex items-center gap-1.5 font-bold text-foreground text-xs">
                            <Lightbulb className="h-3.5 w-3.5 text-amber-400" />
                            <span>Önerilen Çözüm Yolu</span>
                          </div>
                          <p className="text-[11px] text-muted-foreground">
                            {aiAnalysis.suggestedFix}
                          </p>

                          {aiAnalysis.actionSteps && aiAnalysis.actionSteps.length > 0 && (
                            <ul className="space-y-1 pl-4 list-decimal text-[11px] text-muted-foreground">
                              {aiAnalysis.actionSteps.map((step, sIdx) => (
                                <li key={sIdx}>{step}</li>
                              ))}
                            </ul>
                          )}

                          {aiAnalysis.curlReproduction && (
                            <div className="pt-2 border-t border-border flex items-center justify-between gap-2">
                              <code className="text-[10px] font-mono text-primary truncate max-w-xs">
                                {aiAnalysis.curlReproduction}
                              </code>
                              <button
                                type="button"
                                onClick={() => {
                                  if (aiAnalysis?.curlReproduction) {
                                    navigator.clipboard.writeText(aiAnalysis.curlReproduction);
                                    setCopiedCurl(true);
                                    setTimeout(() => setCopiedCurl(false), 2000);
                                  }
                                }}
                                className="px-2 py-1 rounded bg-secondary hover:bg-muted text-[11px] font-semibold text-foreground flex items-center gap-1 shrink-0"
                              >
                                {copiedCurl ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
                                <span>{copiedCurl ? "Kopyalandı" : "cURL"}</span>
                              </button>
                            </div>
                          )}
                        </div>

                        <div className="flex items-center justify-between pt-1 text-[10px] text-muted-foreground">
                          <span>Model: {aiAnalysis.provider}</span>
                          {aiAnalysis.redactionCount > 0 && (
                            <span className="text-emerald-400 font-bold">🛡️ {aiAnalysis.redactionCount} hassas veri maskelendi</span>
                          )}
                        </div>
                      </div>
                    )}
                  </div>

                  {/* Step Timeline */}
                  <div className="space-y-4">
                    <div className="text-xs font-bold text-muted-foreground uppercase tracking-wider">
                      Yaşam Döngüsü Adımları
                    </div>

                    <div className="space-y-3 relative pl-4 border-l-2 border-border">
                      {timelineData.timeline.map((step, idx) => (
                        <div key={idx} className="relative">
                          <div
                            className={`absolute -left-[21px] top-1.5 h-3 w-3 rounded-full border-2 bg-background ${
                              step.status === "COMPLETED" || step.status === "SUCCESS"
                                ? "border-emerald-500 bg-emerald-500"
                                : step.status === "FAILED"
                                ? "border-rose-500 bg-rose-500"
                                : "border-amber-500 bg-amber-500"
                            }`}
                          />
                          <div className="text-xs font-bold text-foreground">
                            {step.description}
                          </div>
                          <div className="text-[11px] text-muted-foreground flex items-center gap-2 mt-0.5">
                            {step.timestamp && (
                              <span>{new Date(step.timestamp).toLocaleTimeString()}</span>
                            )}
                            {step.latencyMs !== undefined && (
                              <span>Gecikme: {step.latencyMs} ms</span>
                            )}
                            {step.statusCode && (
                              <span className={step.statusCode >= 400 ? "text-rose-400 font-mono" : "text-emerald-400 font-mono"}>
                                HTTP {step.statusCode}
                              </span>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>

                  {/* Attempt Telemetry Tabs */}
                  {timelineData.attempts.length > 0 && (
                    <div className="space-y-3 pt-3 border-t border-border">
                      <div className="flex items-center justify-between">
                        <span className="text-xs font-bold text-muted-foreground uppercase tracking-wider">
                          Deneme Geçmişi ({timelineData.attempts.length})
                        </span>
                      </div>

                      <div className="flex gap-1 bg-muted/40 p-1 rounded-xl">
                        {timelineData.attempts.map((att, index) => (
                          <button
                            key={att.id}
                            onClick={() => setActiveAttemptTab(index)}
                            className={`flex-1 py-1.5 rounded-lg text-xs font-bold transition flex items-center justify-center gap-1.5 ${
                              activeAttemptTab === index
                                ? "bg-card text-foreground shadow-sm"
                                : "text-muted-foreground hover:text-foreground"
                            }`}
                          >
                            <span>#{att.attemptNumber}</span>
                            <span
                              className={`text-[10px] ${
                                (att.responseStatusCode || 0) < 400 && att.responseStatusCode
                                  ? "text-emerald-400"
                                  : "text-rose-400"
                              }`}
                            >
                              {att.responseStatusCode ? `${att.responseStatusCode}` : "ERR"}
                            </span>
                          </button>
                        ))}
                      </div>

                      {/* Active Attempt Details */}
                      {timelineData.attempts[activeAttemptTab] && (
                        <div className="rounded-xl border border-border bg-card/60 p-3.5 space-y-3 text-xs">
                          <div className="flex items-center justify-between text-[11px] text-muted-foreground">
                            <span>Gecikme: {timelineData.attempts[activeAttemptTab].latencyMs} ms</span>
                            <span>{new Date(timelineData.attempts[activeAttemptTab].startedAt).toLocaleTimeString()}</span>
                          </div>

                          {timelineData.attempts[activeAttemptTab].errorMessage && (
                            <div className="p-2.5 rounded-lg bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs font-mono">
                              {timelineData.attempts[activeAttemptTab].errorMessage}
                            </div>
                          )}

                          <div>
                            <div className="font-bold text-[11px] text-muted-foreground mb-1">
                              Maskelenmiş Gönderilen Başlıklar (Redacted):
                            </div>
                            <pre className="p-2 rounded-lg bg-background border border-border text-[11px] font-mono text-muted-foreground overflow-x-auto max-h-24">
                              {JSON.stringify(timelineData.attempts[activeAttemptTab].requestHeadersSent, null, 2)}
                            </pre>
                          </div>

                          {timelineData.attempts[activeAttemptTab].responseBodySnippet && (
                            <div>
                              <div className="font-bold text-[11px] text-muted-foreground mb-1">
                                Upstream Yanıt Özeti (Max 2KB):
                              </div>
                              <pre className="p-2 rounded-lg bg-background border border-border text-[11px] font-mono text-muted-foreground overflow-x-auto max-h-32">
                                {timelineData.attempts[activeAttemptTab].responseBodySnippet}
                              </pre>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  )}

                  <div className="pt-2">
                    <button
                      onClick={() => setReplayModalJob(timelineData.job)}
                      className="w-full py-2.5 rounded-xl bg-primary text-primary-foreground text-xs font-bold shadow transition hover:bg-primary/90 flex items-center justify-center gap-2"
                    >
                      <RotateCcw className="h-4 w-4" />
                      <span>Bu Olayı Yeniden İlet (Replay)</span>
                    </button>
                  </div>
                </div>
              ) : null}
            </div>
          </div>
        )}
      </div>

      {/* Safe Replay Modal with Idempotency Guard */}
      {replayModalJob && (
        <div className="fixed inset-0 z-50 bg-background/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="w-full max-w-lg rounded-2xl border border-border bg-card p-6 space-y-5 shadow-2xl animate-in fade-in zoom-in-95">
            <div className="flex items-center justify-between pb-3 border-b border-border">
              <div className="flex items-center gap-2">
                <RotateCcw className="h-5 w-5 text-primary" />
                <h3 className="text-base font-bold text-foreground">Güvenli Webhook Replay</h3>
              </div>
              <button
                onClick={() => setReplayModalJob(null)}
                className="p-1 rounded-lg hover:bg-muted text-muted-foreground hover:text-foreground transition"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="space-y-3 text-xs text-muted-foreground">
              <p>
                Bu webhook isteği tekrar upstream sunucusuna gönderilmek üzere kuyruğa eklenecektir.
              </p>
              <div className="p-3 rounded-xl bg-muted/40 border border-border font-mono space-y-1">
                <div><strong className="text-foreground">Target URL:</strong> {replayModalJob.targetUrl}</div>
                <div><strong className="text-foreground">Request ID:</strong> {replayModalJob.requestId}</div>
                {replayModalJob.idempotencyKey && (
                  <div><strong className="text-foreground">Idempotency Key:</strong> {replayModalJob.idempotencyKey}</div>
                )}
              </div>
            </div>

            {/* Idempotency Override Warning Box */}
            <div className="p-4 rounded-xl border border-amber-500/30 bg-amber-500/5 space-y-3">
              <label className="flex items-start gap-2.5 cursor-pointer">
                <input
                  type="checkbox"
                  checked={overrideIdempotency}
                  onChange={(e) => setOverrideIdempotency(e.target.checked)}
                  className="mt-0.5 h-4 w-4 rounded border-border text-primary focus:ring-primary"
                />
                <div className="text-xs">
                  <span className="font-bold text-foreground block">
                    Idempotency Kontrolünü Atla (Riskli İşlem)
                  </span>
                  <span className="text-muted-foreground">
                    Ödeme veya sipariş olaylarında bu seçeneğin işaretlenmesi upstream backend'de mükerrer işlem oluşmasına neden olabilir. Yalnızca OWNER yetkisiyle çalışır.
                  </span>
                </div>
              </label>

              {overrideIdempotency && (
                <div className="space-y-1.5 pt-2 border-t border-amber-500/20">
                  <label className="text-xs font-bold text-foreground">
                    Zorunlu Gerekçe / Açıklama:
                  </label>
                  <input
                    type="text"
                    placeholder="Örn: Upstream veritabanı kesintisi sonrası manuel onarım..."
                    value={justification}
                    onChange={(e) => setJustification(e.target.value)}
                    className="w-full rounded-lg border border-border bg-background px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
              )}
            </div>

            {replayError && (
              <div className="p-3 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-400 text-xs font-semibold">
                {replayError}
              </div>
            )}

            <div className="flex items-center justify-end gap-2 pt-2 border-t border-border">
              <button
                type="button"
                onClick={() => setReplayModalJob(null)}
                className="px-4 py-2 rounded-xl border border-border bg-muted/40 hover:bg-muted text-xs font-semibold text-foreground transition"
              >
                İptal
              </button>
              <button
                type="button"
                disabled={replayMutation.isPending || (overrideIdempotency && !justification.trim())}
                onClick={() =>
                  replayMutation.mutate({
                    jobId: replayModalJob.id,
                    override: overrideIdempotency,
                    reason: justification,
                  })
                }
                className="px-4 py-2 rounded-xl bg-primary text-primary-foreground text-xs font-bold shadow hover:bg-primary/90 transition disabled:opacity-50 flex items-center gap-2"
              >
                {replayMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <RotateCcw className="h-4 w-4" />
                )}
                <span>Replay Başlat</span>
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
