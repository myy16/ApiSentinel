"use client";

import React, { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import { Project, Endpoint, PayloadMode } from "@apisentinel/shared";
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
  ChevronDown,
  ChevronUp,
  FileJson,
  Key,
  ShieldAlert,
  Lock,
} from "lucide-react";
import { useActiveProject } from "../../../contexts/ProjectContext";

interface ForwardingConfig {
  id?: string;
  endpoint_id?: string;
  target_url?: string;
  max_retries?: number;
  timeout_ms?: number;
  custom_headers?: Record<string, string>;
  is_enabled?: boolean;
  payload_mode?: PayloadMode;
}

interface DLQRecord {
  id: string;
  endpoint_id: string;
  request_id: string;
  target_url: string;
  attempts: number;
  max_retries: number;
  last_error: { String: string; Valid: boolean } | string;
  payload: { String: string; Valid: boolean } | string;
  payload_mode?: PayloadMode;
  status: string;
  created_at: string;
  last_attempt_at: string;
}

export default function ForwardingPage() {
  const queryClient = useQueryClient();
  const { accessToken, organization } = useAuth();
  const { projects, activeProjectId, setActiveProjectId } = useActiveProject();

  const isOwner = organization?.role === "OWNER" || organization?.role === "ADMIN";
  const isViewer = organization?.role === "VIEWER";

  const [selectedEndpointId, setSelectedEndpointId] = useState<string>("");

  const [targetUrl, setTargetUrl] = useState("");
  const [maxRetries, setMaxRetries] = useState(3);
  const [timeoutMs, setTimeoutMs] = useState(5000);
  const [isEnabled, setIsEnabled] = useState(true);
  const [payloadMode, setPayloadMode] = useState<PayloadMode>("REDACTED");
  const [customHeaders, setCustomHeaders] = useState<{ key: string; value: string }[]>([]);
  const [expandedDlqId, setExpandedDlqId] = useState<string | null>(null);

  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  // Fetch endpoints for active project
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
      setPayloadMode(configData.payload_mode || "REDACTED");

      if (configData.custom_headers && typeof configData.custom_headers === "object") {
        const headerPairs = Object.entries(configData.custom_headers).map(([k, v]) => ({
          key: k,
          value: v,
        }));
        setCustomHeaders(headerPairs);
      } else {
        setCustomHeaders([]);
      }
    } else {
      setTargetUrl("");
      setMaxRetries(3);
      setTimeoutMs(5000);
      setIsEnabled(true);
      setPayloadMode("REDACTED");
      setCustomHeaders([]);
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
    mutationFn: (data: {
      targetUrl: string;
      maxRetries: number;
      timeoutMs: number;
      isEnabled: boolean;
      payloadMode: PayloadMode;
      customHeaders: Record<string, string>;
    }) =>
      apiFetch(`/api/endpoints/${activeEndpointId}/forwarding`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify({
          targetUrl: data.targetUrl,
          maxRetries: data.maxRetries,
          timeoutMs: data.timeoutMs,
          isEnabled: data.isEnabled,
          payloadMode: data.payloadMode,
          customHeaders: data.customHeaders,
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

  const handleAddHeader = () => {
    setCustomHeaders([...customHeaders, { key: "", value: "" }]);
  };

  const handleRemoveHeader = (index: number) => {
    setCustomHeaders(customHeaders.filter((_, i) => i !== index));
  };

  const handleHeaderChange = (index: number, field: "key" | "value", val: string) => {
    const updated = [...customHeaders];
    updated[index][field] = val;
    setCustomHeaders(updated);
  };

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    if (!activeEndpointId || !targetUrl.trim()) return;

    const headersMap: Record<string, string> = {};
    customHeaders.forEach((h) => {
      if (h.key.trim() && h.value.trim()) {
        headersMap[h.key.trim()] = h.value.trim();
      }
    });

    saveMutation.mutate({
      targetUrl: targetUrl.trim(),
      maxRetries: Number(maxRetries),
      timeoutMs: Number(timeoutMs),
      isEnabled,
      payloadMode,
      customHeaders: headersMap,
    });
  };

  const getErrorMessage = (record: DLQRecord) => {
    if (typeof record.last_error === "object" && record.last_error !== null) {
      return record.last_error.String || "Bilinmeyen İletim Hatası";
    }
    return record.last_error || "Bilinmeyen İletim Hatası";
  };

  const getPayloadText = (record: DLQRecord) => {
    let raw = "";
    if (typeof record.payload === "object" && record.payload !== null) {
      raw = record.payload.String || "";
    } else {
      raw = record.payload || "";
    }
    try {
      return JSON.stringify(JSON.parse(raw), null, 2);
    } catch {
      return raw || "// Boş yük";
    }
  };

  const activeEndpoint = endpoints.find((e) => e.id === activeEndpointId);

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
              <Share2 className="h-6 w-6 text-primary" />
              Upstream Forwarding & Durable Outbox / DLQ
            </h1>
          </div>
          <p className="text-sm text-muted-foreground mt-1">
            Temiz webhook isteklerini asıl sunucunuza iletin; hassas verileri AES-256-GCM ile şifreleyin ve başarısız iletimleri Outbox/DLQ ile güvenceye alın.
          </p>
        </div>

        {/* Project & Endpoint Selectors */}
        <div className="flex items-center gap-3">
          {projects.length > 0 && (
            <select
              value={activeProjectId}
              onChange={(e) => {
                setActiveProjectId(e.target.value);
                setSelectedEndpointId("");
              }}
              className="rounded-xl border border-border bg-card px-3 py-2 text-xs font-semibold text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
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
              className="rounded-xl border border-border bg-card px-3 py-2 text-xs font-semibold text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
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
          className={`flex items-center gap-2 rounded-xl p-4 text-sm font-medium ${
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
          <div className="rounded-2xl border border-border bg-card p-6 shadow-sm space-y-4 glow-card">
            <div className="flex items-center gap-3 border-b border-border pb-4">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10 text-primary border border-primary/20">
                <Server className="h-5 w-5" />
              </div>
              <div>
                <h2 className="text-base font-bold text-foreground">
                  {activeEndpoint ? activeEndpoint.name : "Endpoint"} İletim Ayarları
                </h2>
                <p className="text-xs text-muted-foreground">Upstream hedef ve güvenlik yapılandırması</p>
              </div>
            </div>

            <form onSubmit={handleSave} className="space-y-4">
              <div>
                <label className="block text-[10px] font-bold uppercase tracking-wider text-muted-foreground mb-1.5">
                  Hedef Upstream URL
                </label>
                <input
                  type="url"
                  required
                  placeholder="https://api.mycompany.com/webhooks/stripe"
                  value={targetUrl}
                  disabled={isViewer}
                  onChange={(e) => setTargetUrl(e.target.value)}
                  className="w-full rounded-xl border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground/60 focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
                />
                <p className="text-[11px] text-muted-foreground mt-1">
                  Temizlenen webhook yükünün POST edileceği asıl sunucu adresi.
                </p>
              </div>

              {/* Payload Mode Selection (#8) */}
              <div>
                <label className="block text-[10px] font-bold uppercase tracking-wider text-muted-foreground mb-1.5">
                  İletim Yükü Modu (Payload Mode)
                </label>
                <div className="grid grid-cols-2 gap-3">
                  <button
                    type="button"
                    disabled={isViewer}
                    onClick={() => setPayloadMode("REDACTED")}
                    className={`flex flex-col items-start p-3 rounded-xl border text-left transition ${
                      payloadMode === "REDACTED"
                        ? "border-primary bg-primary/10 text-primary"
                        : "border-border bg-background text-muted-foreground hover:border-border/80"
                    }`}
                  >
                    <div className="flex items-center gap-1.5 font-bold text-xs">
                      <ShieldCheck className="h-4 w-4 text-emerald-400" />
                      <span>REDACTED (Önerilen)</span>
                    </div>
                    <span className="text-[10px] text-muted-foreground mt-1">
                      Hassas verileri (PII/Secret) maskeleyerek iletir.
                    </span>
                  </button>

                  <button
                    type="button"
                    disabled={isViewer}
                    onClick={() => setPayloadMode("RAW")}
                    className={`flex flex-col items-start p-3 rounded-xl border text-left transition ${
                      payloadMode === "RAW"
                        ? "border-amber-500/50 bg-amber-500/10 text-amber-400"
                        : "border-border bg-background text-muted-foreground hover:border-border/80"
                    }`}
                  >
                    <div className="flex items-center gap-1.5 font-bold text-xs">
                      <ShieldAlert className="h-4 w-4 text-amber-400" />
                      <span>RAW (Ham Yük)</span>
                    </div>
                    <span className="text-[10px] text-muted-foreground mt-1">
                      Orijinal gövdeyi doğrudan iletir, audit log tutar.
                    </span>
                  </button>
                </div>

                {payloadMode === "RAW" && (
                  <div className="mt-2 flex items-center gap-2 rounded-lg bg-amber-500/10 border border-amber-500/20 p-2.5 text-[11px] text-amber-400">
                    <AlertTriangle className="h-4 w-4 shrink-0" />
                    <span>Dikkat: RAW modu seçildiğinde yük maskelenmeden iletilir ve denetim logu üretilir.</span>
                  </div>
                )}
              </div>

              {/* Custom Headers with AES-256-GCM Encryption (#3.1, #3.2) */}
              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <label className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground flex items-center gap-1">
                    <Lock className="h-3 w-3 text-primary" />
                    <span>Özel HTTP Başlıkları (AES-256 Şifrelenir)</span>
                  </label>
                  {!isViewer && (
                    <button
                      type="button"
                      onClick={handleAddHeader}
                      className="text-[11px] text-primary hover:underline font-bold"
                    >
                      + Başlık Ekle
                    </button>
                  )}
                </div>

                {customHeaders.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-border bg-background/50 p-3 text-center text-[11px] text-muted-foreground">
                    Özel başlık tanımlanmadı (Örn: Authorization: Bearer ...)
                  </div>
                ) : (
                  <div className="space-y-2">
                    {customHeaders.map((h, idx) => (
                      <div key={idx} className="flex items-center gap-2">
                        <input
                          type="text"
                          placeholder="Header Adı (örn. Authorization)"
                          value={h.key}
                          disabled={isViewer}
                          onChange={(e) => handleHeaderChange(idx, "key", e.target.value)}
                          className="w-1/2 rounded-lg border border-border bg-background px-2.5 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary disabled:opacity-50"
                        />
                        <input
                          type="text"
                          placeholder="Değer (örn. Bearer ****)"
                          value={h.value}
                          disabled={isViewer}
                          onChange={(e) => handleHeaderChange(idx, "value", e.target.value)}
                          className="w-1/2 rounded-lg border border-border bg-background px-2.5 py-1.5 text-xs font-mono text-foreground focus:outline-none focus:ring-1 focus:ring-primary disabled:opacity-50"
                        />
                        {!isViewer && (
                          <button
                            type="button"
                            onClick={() => handleRemoveHeader(idx)}
                            className="text-muted-foreground hover:text-destructive p-1"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-[10px] font-bold uppercase tracking-wider text-muted-foreground mb-1.5">
                    Maksimum Retry
                  </label>
                  <input
                    type="number"
                    min="1"
                    max="10"
                    value={maxRetries}
                    disabled={isViewer}
                    onChange={(e) => setMaxRetries(Number(e.target.value))}
                    className="w-full rounded-xl border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
                  />
                </div>

                <div>
                  <label className="block text-[10px] font-bold uppercase tracking-wider text-muted-foreground mb-1.5">
                    Zaman Aşımı (ms)
                  </label>
                  <input
                    type="number"
                    min="500"
                    max="30000"
                    step="500"
                    value={timeoutMs}
                    disabled={isViewer}
                    onChange={(e) => setTimeoutMs(Number(e.target.value))}
                    className="w-full rounded-xl border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
                  />
                </div>
              </div>

              <div className="flex items-center gap-3 pt-2">
                <input
                  type="checkbox"
                  id="isEnabled"
                  checked={isEnabled}
                  disabled={isViewer}
                  onChange={(e) => setIsEnabled(e.target.checked)}
                  className="h-4 w-4 rounded border-border text-primary focus:ring-primary disabled:opacity-50"
                />
                <label htmlFor="isEnabled" className="text-xs font-semibold text-foreground cursor-pointer">
                  Upstream İletimini Aktif Et
                </label>
              </div>

              <div className="pt-2">
                <button
                  type="submit"
                  disabled={saveMutation.isPending || !activeEndpointId || isViewer}
                  className="w-full flex items-center justify-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-xs font-bold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
                >
                  {saveMutation.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="h-4 w-4" />
                  )}
                  <span>{isViewer ? "Salt Okunur (Görüntüleme)" : "İletim Ayarlarını Kaydet"}</span>
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
              <h2 className="text-base font-bold text-foreground">
                Outbox & Dead Letter Queue ({dlqRecords.length})
              </h2>
            </div>

            <div className="flex items-center gap-2">
              <button
                onClick={() => refetchDLQ()}
                className="flex items-center gap-1 rounded-xl border border-border bg-card px-3 py-1.5 text-xs font-semibold text-foreground hover:bg-secondary transition"
              >
                <RotateCw className="h-3.5 w-3.5" />
                <span>Yenile</span>
              </button>

              {dlqRecords.length > 0 && isOwner && (
                <button
                  onClick={() => {
                    if (confirm("Bu endpoint'e ait tüm DLQ kayıtlarını silmek istediğinize emin misiniz?")) {
                      purgeMutation.mutate();
                    }
                  }}
                  disabled={purgeMutation.isPending}
                  className="flex items-center gap-1 rounded-xl border border-destructive/30 px-3 py-1.5 text-xs font-semibold text-destructive hover:bg-destructive/10 transition"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                  <span>Tümünü Temizle</span>
                </button>
              )}
            </div>
          </div>

          {isDLQLoading ? (
            <div className="flex h-48 items-center justify-center rounded-2xl border border-border bg-card">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
            </div>
          ) : dlqRecords.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border bg-card/40 p-12 text-center">
              <ShieldCheck className="h-10 w-10 text-emerald-400 mb-3" />
              <p className="text-sm font-semibold text-foreground">Kuyruk Tertemiz!</p>
              <p className="text-xs text-muted-foreground mt-1 max-w-sm">
                Başarısız olan veya bekleyen hiçbir webhook bulunmuyor. Tüm temiz istekler upstream hedefine ulaştırıldı.
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {dlqRecords.map((record) => {
                const isSent = record.status === "SENT" || record.status === "RESOLVED";
                const isProcessing = record.status === "PROCESSING";
                const isRetryWait = record.status === "RETRY_WAIT" || record.status === "PENDING";
                const errorText = getErrorMessage(record);
                const isExpanded = expandedDlqId === record.id;

                return (
                  <div
                    key={record.id}
                    className={`rounded-2xl border p-4 shadow-sm transition space-y-3 ${
                      isSent
                        ? "border-emerald-500/20 bg-emerald-500/5"
                        : isProcessing
                        ? "border-primary/20 bg-primary/5"
                        : "border-border bg-card hover:border-border/80"
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span
                          className={`rounded-full px-2.5 py-0.5 text-[10px] font-bold ${
                            isSent
                              ? "bg-emerald-500/20 text-emerald-400 border border-emerald-500/30"
                              : isProcessing
                              ? "bg-primary/20 text-primary border border-primary/30"
                              : isRetryWait
                              ? "bg-amber-500/20 text-amber-400 border border-amber-500/30"
                              : "bg-rose-500/20 text-rose-400 border border-rose-500/30"
                          }`}
                        >
                          {record.status}
                        </span>

                        {record.payload_mode && (
                          <span className="rounded px-2 py-0.5 text-[10px] font-mono bg-secondary text-muted-foreground border border-border">
                            {record.payload_mode}
                          </span>
                        )}

                        <span className="text-xs font-mono text-muted-foreground">
                          Deneme: {record.attempts}/{record.max_retries || 3}
                        </span>
                      </div>

                      <span className="text-[11px] text-muted-foreground font-mono">
                        {new Date(record.created_at).toLocaleString("tr-TR")}
                      </span>
                    </div>

                    <div className="space-y-1 text-xs">
                      {errorText && errorText !== "Bilinmeyen İletim Hatası" && (
                        <p className="font-mono text-rose-400 truncate">
                          <span className="text-muted-foreground">Hata: </span>
                          {errorText}
                        </p>
                      )}
                      <p className="font-mono text-muted-foreground truncate">
                        <span>Hedef: </span>
                        {record.target_url}
                      </p>
                    </div>

                    {/* Expandable Payload Toggle */}
                    <div className="pt-2 border-t border-border flex items-center justify-between">
                      <button
                        onClick={() => setExpandedDlqId(isExpanded ? null : record.id)}
                        className="flex items-center gap-1 text-[11px] text-primary hover:underline font-semibold"
                      >
                        <FileJson className="h-3.5 w-3.5" />
                        <span>{isExpanded ? "Yükü Gizle" : "Yükü İncele (Payload)"}</span>
                        {isExpanded ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
                      </button>

                      {!isSent && !isViewer && (
                        <button
                          onClick={() => retryMutation.mutate(record.id)}
                          disabled={retryMutation.isPending}
                          className="flex items-center gap-1.5 rounded-xl bg-primary px-3 py-1.5 text-xs font-bold text-primary-foreground hover:bg-primary/90 transition disabled:opacity-50"
                        >
                          {retryMutation.isPending ? (
                            <Loader2 className="h-3.5 w-3.5 animate-spin" />
                          ) : (
                            <RefreshCw className="h-3.5 w-3.5" />
                          )}
                          <span>Yeniden İlet (Retry)</span>
                        </button>
                      )}
                    </div>

                    {/* Expanded Payload Content */}
                    {isExpanded && (
                      <pre className="rounded-xl bg-background p-3 text-[11px] font-mono text-foreground border border-border overflow-x-auto leading-relaxed max-h-48">
                        {getPayloadText(record)}
                      </pre>
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
