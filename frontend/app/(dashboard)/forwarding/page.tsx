"use client";

import React, { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import { Project, Endpoint } from "@apisentinel/shared";
import {
  Share2,
  Save,
  RotateCw,
  AlertOctagon,
  CheckCircle2,
  Clock,
  ShieldCheck,
  Server,
  Loader2,
  ArrowUpRight,
  Trash2,
  RefreshCw,
  Layers,
  AlertTriangle,
} from "lucide-react";

interface ForwardingConfig {
  id?: string;
  endpoint_id?: string;
  target_url?: string;
  max_retries?: number;
  timeout_ms?: number;
  custom_headers?: Record<string, string>;
  is_enabled?: boolean;
}

interface DLQRecord {
  id: string;
  endpoint_id: string;
  request_id: string;
  target_url: string;
  attempts: number;
  last_error: { String: string; Valid: boolean } | string;
  payload: { String: string; Valid: boolean } | string;
  status: string;
  created_at: string;
  last_attempt_at: string;
}

export default function ForwardingPage() {
  const queryClient = useQueryClient();
  const { accessToken, organization } = useAuth();

  const [selectedProjectId, setSelectedProjectId] = useState<string>("");
  const [selectedEndpointId, setSelectedEndpointId] = useState<string>("");

  const [targetUrl, setTargetUrl] = useState("");
  const [maxRetries, setMaxRetries] = useState(3);
  const [timeoutMs, setTimeoutMs] = useState(5000);
  const [isEnabled, setIsEnabled] = useState(true);

  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

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
  const activeProjectId = selectedProjectId || (projects[0]?.id ?? "");

  // 2. Fetch endpoints for active project
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
  const activeEndpointId = selectedEndpointId || (endpoints[0]?.id ?? "");

  // 3. Fetch Forwarding Config for active endpoint
  const { data: configData, isLoading: isConfigLoading } = useQuery({
    queryKey: ["forwardingConfig", activeEndpointId],
    queryFn: async () => {
      try {
        return await apiFetch<ForwardingConfig>(`/api/endpoints/${activeEndpointId}/forwarding`, {
          token: accessToken,
          organizationId: organization?.id,
        });
      } catch {
        return null;
      }
    },
    enabled: !!accessToken && !!activeEndpointId && !!organization?.id,
  });

  useEffect(() => {
    if (configData) {
      setTargetUrl(configData.target_url || "");
      setMaxRetries(configData.max_retries || 3);
      setTimeoutMs(configData.timeout_ms || 5000);
      setIsEnabled(configData.is_enabled ?? true);
    } else {
      setTargetUrl("");
      setMaxRetries(3);
      setTimeoutMs(5000);
      setIsEnabled(true);
    }
  }, [configData, activeEndpointId]);

  // 4. Fetch DLQ records
  const {
    data: dlqData,
    isLoading: isDLQLoading,
    refetch: refetchDLQ,
  } = useQuery({
    queryKey: ["dlqRecords", activeEndpointId],
    queryFn: () =>
      apiFetch<DLQRecord[]>(`/api/endpoints/${activeEndpointId}/dlq`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeEndpointId && !!organization?.id,
  });

  const dlqRecords = dlqData || [];

  // 5. Save Config Mutation
  const saveMutation = useMutation({
    mutationFn: (data: { targetUrl: string; maxRetries: number; timeoutMs: number; isEnabled: boolean }) =>
      apiFetch(`/api/endpoints/${activeEndpointId}/forwarding`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify({
          targetUrl: data.targetUrl,
          maxRetries: data.maxRetries,
          timeoutMs: data.timeoutMs,
          isEnabled: data.isEnabled,
          customHeaders: {},
        }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["forwardingConfig", activeEndpointId] });
      setMessage({ type: "success", text: "Upstream Forwarding ayarları başarıyla kaydedildi!" });
      setTimeout(() => setMessage(null), 4000);
    },
    onError: (err: any) => {
      setMessage({ type: "error", text: err.message || "Ayarlar kaydedilemedi." });
    },
  });

  // 6. Retry DLQ Record Mutation
  const retryMutation = useMutation({
    mutationFn: (dlqId: string) =>
      apiFetch(`/api/dlq/${dlqId}/retry`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["dlqRecords", activeEndpointId] });
      setMessage({ type: "success", text: "DLQ İsteği başarıyla yeniden iletildi ve çözüldü!" });
      setTimeout(() => setMessage(null), 4000);
    },
    onError: (err: any) => {
      setMessage({ type: "error", text: "Yeniden deneme başarısız: " + (err.message || "Hedef sunucuya ulaşılamadı") });
    },
  });

  // 7. Purge DLQ Mutation
  const purgeMutation = useMutation({
    mutationFn: () =>
      apiFetch(`/api/endpoints/${activeEndpointId}/dlq`, {
        method: "DELETE",
        token: accessToken,
        organizationId: organization?.id,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["dlqRecords", activeEndpointId] });
      setMessage({ type: "success", text: "Bu endpoint'e ait tüm DLQ kayıtları temizlendi." });
      setTimeout(() => setMessage(null), 4000);
    },
  });

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    if (!activeEndpointId || !targetUrl.trim()) return;
    saveMutation.mutate({
      targetUrl: targetUrl.trim(),
      maxRetries: Number(maxRetries),
      timeoutMs: Number(timeoutMs),
      isEnabled,
    });
  };

  const getErrorMessage = (record: DLQRecord) => {
    if (typeof record.last_error === "object" && record.last_error !== null) {
      return record.last_error.String || "Bilinmeyen İletim Hatası";
    }
    return record.last_error || "Bilinmeyen İletim Hatası";
  };

  const activeEndpoint = endpoints.find((e) => e.id === activeEndpointId);

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <Share2 className="h-6 w-6 text-primary" />
            Upstream Forwarding & DLQ (Reverse Ingestion)
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            Temiz webhook isteklerini asıl sunucunuza iletin; başarısız iletimleri otomatik üstel geri çekilme (retry) ve Dead Letter Queue ile güvenceye alın.
          </p>
        </div>

        {/* Project & Endpoint Selectors */}
        <div className="flex items-center gap-3">
          {projects.length > 0 && (
            <select
              value={activeProjectId}
              onChange={(e) => {
                setSelectedProjectId(e.target.value);
                setSelectedEndpointId("");
              }}
              className="rounded-lg border border-border bg-card px-3 py-2 text-xs font-semibold text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            >
              {projects.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          )}

          {endpoints.length > 0 && (
            <select
              value={activeEndpointId}
              onChange={(e) => setSelectedEndpointId(e.target.value)}
              className="rounded-lg border border-border bg-card px-3 py-2 text-xs font-semibold text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            >
              {endpoints.map((ep) => (
                <option key={ep.id} value={ep.id}>
                  {ep.name} (/{ep.slug})
                </option>
              ))}
            </select>
          )}
        </div>
      </div>

      {message && (
        <div
          className={`flex items-center gap-2 rounded-lg p-4 text-sm font-medium ${
            message.type === "success"
              ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20"
              : "bg-destructive/10 text-destructive border border-destructive/20"
          }`}
        >
          {message.type === "success" ? <CheckCircle2 className="h-5 w-5" /> : <AlertTriangle className="h-5 w-5" />}
          <span>{message.text}</span>
        </div>
      )}

      {/* Main Grid: Config Form & DLQ List */}
      <div className="grid grid-cols-1 gap-8 lg:grid-cols-12">
        {/* Left Column: Forwarding Settings (5 cols) */}
        <div className="lg:col-span-5 space-y-6">
          <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
            <div className="flex items-center gap-2 border-b border-border pb-4 mb-4">
              <Server className="h-5 w-5 text-primary" />
              <h2 className="text-base font-semibold text-foreground">
                {activeEndpoint ? activeEndpoint.name : "Endpoint"} İletim Ayarları
              </h2>
            </div>

            <form onSubmit={handleSave} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                  Hedef Upstream URL
                </label>
                <input
                  type="url"
                  required
                  placeholder="https://api.mycompany.com/webhooks/stripe"
                  value={targetUrl}
                  onChange={(e) => setTargetUrl(e.target.value)}
                  className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground/60 focus:outline-none focus:ring-2 focus:ring-primary"
                />
                <p className="text-[11px] text-muted-foreground mt-1">
                  Temizlenen webhook yükünün POST edileceği asıl sunucu adresi.
                </p>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                    Maksimum Retry
                  </label>
                  <input
                    type="number"
                    min="1"
                    max="10"
                    value={maxRetries}
                    onChange={(e) => setMaxRetries(Number(e.target.value))}
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                    Zaman Aşımı (ms)
                  </label>
                  <input
                    type="number"
                    min="500"
                    max="30000"
                    step="500"
                    value={timeoutMs}
                    onChange={(e) => setTimeoutMs(Number(e.target.value))}
                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                  />
                </div>
              </div>

              <div className="flex items-center gap-3 pt-2">
                <input
                  type="checkbox"
                  id="isEnabled"
                  checked={isEnabled}
                  onChange={(e) => setIsEnabled(e.target.checked)}
                  className="h-4 w-4 rounded border-border text-primary focus:ring-primary"
                />
                <label htmlFor="isEnabled" className="text-xs font-semibold text-foreground">
                  Upstream İletimini Aktif Et
                </label>
              </div>

              <div className="pt-2">
                <button
                  type="submit"
                  disabled={saveMutation.isPending || !activeEndpointId}
                  className="w-full flex items-center justify-center gap-2 rounded-lg bg-primary px-4 py-2.5 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
                >
                  {saveMutation.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="h-4 w-4" />
                  )}
                  <span>İletim Ayarlarını Kaydet</span>
                </button>
              </div>
            </form>
          </div>
        </div>

        {/* Right Column: Dead Letter Queue (DLQ) (7 cols) */}
        <div className="lg:col-span-7 space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <AlertOctagon className="h-5 w-5 text-rose-400" />
              <h2 className="text-base font-semibold text-foreground">
                Dead Letter Queue ({dlqRecords.length})
              </h2>
            </div>

            <div className="flex items-center gap-2">
              <button
                onClick={() => refetchDLQ()}
                className="flex items-center gap-1 rounded-lg border border-border bg-card px-2.5 py-1.5 text-xs font-medium text-foreground hover:bg-secondary transition"
              >
                <RotateCw className="h-3 w-3" />
                <span>Yenile</span>
              </button>

              {dlqRecords.length > 0 && (
                <button
                  onClick={() => {
                    if (confirm("Bu endpoint'e ait tüm DLQ kayıtlarını silmek istediğinize emin misiniz?")) {
                      purgeMutation.mutate();
                    }
                  }}
                  disabled={purgeMutation.isPending}
                  className="flex items-center gap-1 rounded-lg border border-destructive/30 px-2.5 py-1.5 text-xs font-semibold text-destructive hover:bg-destructive/10 transition"
                >
                  <Trash2 className="h-3 w-3" />
                  <span>Tümünü Temizle</span>
                </button>
              )}
            </div>
          </div>

          {isDLQLoading ? (
            <div className="flex h-48 items-center justify-center rounded-xl border border-border bg-card">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
            </div>
          ) : dlqRecords.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border bg-card/40 p-12 text-center">
              <ShieldCheck className="h-10 w-10 text-emerald-400 mb-3" />
              <p className="text-sm font-semibold text-foreground">Kuyruk Tertemiz!</p>
              <p className="text-xs text-muted-foreground mt-1 max-w-sm">
                Başarısız olan veya iletilemeyen hiçbir webhook bulunmuyor. Tüm temiz istekler upstream hedefine ulaştırıldı.
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {dlqRecords.map((record) => {
                const isResolved = record.status === "RESOLVED";
                const errorText = getErrorMessage(record);

                return (
                  <div
                    key={record.id}
                    className={`rounded-xl border p-4 shadow-sm transition space-y-3 ${
                      isResolved
                        ? "border-emerald-500/20 bg-emerald-500/5"
                        : "border-border bg-card hover:border-border/80"
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span
                          className={`rounded-full px-2 py-0.5 text-[10px] font-bold ${
                            isResolved
                              ? "bg-emerald-500/20 text-emerald-400"
                              : "bg-rose-500/20 text-rose-400"
                          }`}
                        >
                          {record.status}
                        </span>
                        <span className="text-xs font-mono text-muted-foreground">
                          Deneme Sayısı: {record.attempts}
                        </span>
                      </div>

                      <span className="text-[11px] text-muted-foreground font-mono">
                        {new Date(record.created_at).toLocaleString("tr-TR")}
                      </span>
                    </div>

                    <div className="space-y-1 text-xs">
                      <p className="font-mono text-rose-400 truncate">
                        <span className="text-muted-foreground">Hata: </span>
                        {errorText}
                      </p>
                      <p className="font-mono text-muted-foreground truncate">
                        <span>Hedef: </span>
                        {record.target_url}
                      </p>
                    </div>

                    {!isResolved && (
                      <div className="flex justify-end pt-2 border-t border-border">
                        <button
                          onClick={() => retryMutation.mutate(record.id)}
                          disabled={retryMutation.isPending}
                          className="flex items-center gap-1.5 rounded-lg bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground hover:bg-primary/90 transition disabled:opacity-50"
                        >
                          {retryMutation.isPending ? (
                            <Loader2 className="h-3 w-3 animate-spin" />
                          ) : (
                            <RefreshCw className="h-3 w-3" />
                          )}
                          <span>Yeniden İlet (Retry)</span>
                        </button>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
