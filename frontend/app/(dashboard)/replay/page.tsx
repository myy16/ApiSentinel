"use client";

import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import {
  Project,
  CapturedRequest,
  ReplayJob,
  ReplayEnvironment,
  ReplayTestSuite,
  TestSuiteRunReport,
} from "@apisentinel/shared";
import {
  Repeat,
  Play,
  Clock,
  Globe,
  ShieldCheck,
  ShieldAlert,
  Loader2,
  CheckCircle2,
  XCircle,
  ExternalLink,
  Search,
  Code,
  FileJson,
  Layers,
  Zap,
  Server,
  Terminal,
  Activity,
  ArrowRight,
  Plus,
  Trash2,
  Check,
  Copy,
  Sliders,
  Sparkles,
  RefreshCw,
  FolderPlus,
  ListOrdered,
  Percent,
} from "lucide-react";
import { useActiveProject } from "../../../contexts/ProjectContext";

export default function ReplayPage() {
  const queryClient = useQueryClient();
  const { accessToken, organization } = useAuth();
  const { projects, activeProjectId } = useActiveProject();

  const [activeTab, setActiveTab] = useState<"single" | "suites">("single");

  // Single Replay States
  const [isNewReplayOpen, setIsNewReplayOpen] = useState(false);
  const [selectedRequestId, setSelectedRequestId] = useState<string>("");
  const [environment, setEnvironment] = useState<ReplayEnvironment>("STAGING");
  const [targetUrl, setTargetUrl] = useState<string>("https://httpbin.org/post");
  const [renewIdempotency, setRenewIdempotency] = useState<boolean>(true);
  const [justification, setJustification] = useState<string>("Staging ortamında güvenli test");
  const [customHeaderKey, setCustomHeaderKey] = useState<string>("");
  const [customHeaderVal, setCustomHeaderVal] = useState<string>("");
  const [customHeaders, setCustomHeaders] = useState<Record<string, string>>({
    "X-ApiSentinel-Replay-Env": "staging",
  });
  const [replayError, setReplayError] = useState<string | null>(null);
  const [lastReplayResult, setLastReplayResult] = useState<any | null>(null);
  const [selectedReplayDetail, setSelectedReplayDetail] = useState<ReplayJob | null>(null);

  // Test Suite States
  const [isNewSuiteOpen, setIsNewSuiteOpen] = useState(false);
  const [suiteName, setSuiteName] = useState<string>("");
  const [suiteDescription, setSuiteDescription] = useState<string>("");
  const [suiteSelectedRequests, setSuiteSelectedRequests] = useState<string[]>([]);
  const [suiteEnvironment, setSuiteEnvironment] = useState<ReplayEnvironment>("STAGING");
  const [suiteTargetUrl, setSuiteTargetUrl] = useState<string>("https://httpbin.org/post");
  const [suiteRenewIdemp, setSuiteRenewIdemp] = useState<boolean>(true);
  const [suiteRunReport, setSuiteRunReport] = useState<TestSuiteRunReport | null>(null);
  const [runningSuiteId, setRunningSuiteId] = useState<string | null>(null);

  // 1. Fetch captured requests
  const { data: requestsData } = useQuery({
    queryKey: ["requests", activeProjectId],
    queryFn: () =>
      apiFetch<{ requests: CapturedRequest[] }>(`/api/projects/${activeProjectId}/requests`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
  });

  const requests = requestsData?.requests || [];

  // 2. Fetch past replay jobs
  const { data: replaysData, isLoading: isReplaysLoading } = useQuery({
    queryKey: ["replays", activeProjectId],
    queryFn: () =>
      apiFetch<{ replays: ReplayJob[] }>(`/api/projects/${activeProjectId}/replays`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
  });

  const replays = replaysData?.replays || [];

  // 3. Fetch Test Suites
  const { data: suitesData, isLoading: isSuitesLoading } = useQuery({
    queryKey: ["test-suites", activeProjectId],
    queryFn: () =>
      apiFetch<{ suites: any[]; count: number }>(`/api/projects/${activeProjectId}/test-suites`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
  });

  const suites: ReplayTestSuite[] = (suitesData?.suites || []).map((s: any) => ({
    id: s.id,
    projectId: s.project_id || s.projectId,
    name: s.name,
    description: s.description,
    requestIds: Array.isArray(s.request_ids) ? s.request_ids : JSON.parse(s.request_ids || "[]"),
    targetEnvironment: s.target_environment || s.targetEnvironment || "STAGING",
    targetUrl: s.target_url || s.targetUrl,
    renewIdempotency: s.renew_idempotency !== undefined ? s.renew_idempotency : true,
    customHeaders: typeof s.custom_headers === "object" ? s.custom_headers : {},
    createdAt: s.created_at || s.createdAt,
    updatedAt: s.updated_at || s.updatedAt,
  }));

  // Single Replay Mutation
  const replayMutation = useMutation({
    mutationFn: (vars: {
      requestId: string;
      targetUrl: string;
      environment: ReplayEnvironment;
      renewIdempotency: boolean;
      customHeaders: Record<string, string>;
      justification: string;
    }) =>
      apiFetch<any>(`/api/requests/${vars.requestId}/replay`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify({
          targetUrl: vars.targetUrl,
          environment: vars.environment,
          renewIdempotency: vars.renewIdempotency,
          customHeaders: vars.customHeaders,
          justification: vars.justification,
          overrideIdempotency: true,
        }),
      }),
    onSuccess: (data) => {
      setLastReplayResult(data);
      queryClient.invalidateQueries({ queryKey: ["replays", activeProjectId] });
      queryClient.invalidateQueries({ queryKey: ["audit-logs", activeProjectId] });
      setReplayError(null);
    },
    onError: (err: any) => {
      setReplayError(err.message || "Replay işlemi başarısız oldu.");
    },
  });

  // Create Suite Mutation
  const createSuiteMutation = useMutation({
    mutationFn: (vars: {
      name: string;
      description: string;
      requestIds: string[];
      targetEnvironment: string;
      targetUrl: string;
      renewIdempotency: boolean;
    }) =>
      apiFetch<any>(`/api/projects/${activeProjectId}/test-suites`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify(vars),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["test-suites", activeProjectId] });
      setIsNewSuiteOpen(false);
      setSuiteName("");
      setSuiteDescription("");
      setSuiteSelectedRequests([]);
    },
  });

  // Run Suite Mutation
  const runSuiteMutation = useMutation({
    mutationFn: (suiteId: string) => {
      setRunningSuiteId(suiteId);
      return apiFetch<TestSuiteRunReport>(`/api/test-suites/${suiteId}/run`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
      });
    },
    onSuccess: (data) => {
      setSuiteRunReport(data);
      setRunningSuiteId(null);
      queryClient.invalidateQueries({ queryKey: ["replays", activeProjectId] });
    },
    onError: () => {
      setRunningSuiteId(null);
    },
  });

  const handleEnvironmentSelect = (env: ReplayEnvironment) => {
    setEnvironment(env);
    if (env === "STAGING") {
      setTargetUrl("https://httpbin.org/post");
    } else if (env === "LOCAL") {
      setTargetUrl("http://localhost:8080/api/webhooks");
    } else if (env === "DEV") {
      setTargetUrl("https://dev.api.internal/webhook");
    }
  };

  const handleAddCustomHeader = () => {
    if (!customHeaderKey.trim()) return;
    setCustomHeaders((prev) => ({
      ...prev,
      [customHeaderKey.trim()]: customHeaderVal.trim(),
    }));
    setCustomHeaderKey("");
    setCustomHeaderVal("");
  };

  const handleRemoveCustomHeader = (key: string) => {
    setCustomHeaders((prev) => {
      const copy = { ...prev };
      delete copy[key];
      return copy;
    });
  };

  const handleExecuteReplay = (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedRequestId || !targetUrl.trim()) return;
    setLastReplayResult(null);
    replayMutation.mutate({
      requestId: selectedRequestId,
      targetUrl: targetUrl.trim(),
      environment,
      renewIdempotency,
      customHeaders,
      justification,
    });
  };

  const toggleSuiteRequest = (id: string) => {
    setSuiteSelectedRequests((prev) =>
      prev.includes(id) ? prev.filter((item) => item !== id) : [...prev, id]
    );
  };

  return (
    <div className="space-y-6 animate-in fade-in duration-300">
      {/* Top Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="flex items-center gap-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-primary/10 text-primary border border-primary/20">
              <Repeat className="h-5 w-5" />
            </div>
            <h1 className="text-xl font-bold tracking-tight text-foreground">
              Replay Lab & Test Suites
            </h1>
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            Idempotency-Safe tekli replay yönlendirici ve sıralı test senaryoları koşucusu
          </p>
        </div>

        {/* Action Button depending on tab */}
        {activeTab === "single" ? (
          <button
            onClick={() => {
              setIsNewReplayOpen(true);
              setLastReplayResult(null);
              setReplayError(null);
              if (!selectedRequestId && requests.length > 0) {
                setSelectedRequestId(requests[0].id);
              }
            }}
            className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-xs font-bold text-primary-foreground shadow-sm transition hover:bg-primary/90"
          >
            <Play className="h-4 w-4" />
            <span>Yeni Replay Ateşle</span>
          </button>
        ) : (
          <button
            onClick={() => setIsNewSuiteOpen(true)}
            className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-xs font-bold text-primary-foreground shadow-sm transition hover:bg-primary/90"
          >
            <Plus className="h-4 w-4" />
            <span>Yeni Test Paketi Oluştur</span>
          </button>
        )}
      </div>

      {/* Mode Switcher Tabs */}
      <div className="flex border-b border-border gap-6">
        <button
          onClick={() => setActiveTab("single")}
          className={`pb-3 text-xs font-bold transition flex items-center gap-2 border-b-2 ${
            activeTab === "single"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          }`}
        >
          <Zap className="h-4 w-4" />
          <span>⚡ Tekli Replay Yönlendirici</span>
        </button>

        <button
          onClick={() => setActiveTab("suites")}
          className={`pb-3 text-xs font-bold transition flex items-center gap-2 border-b-2 ${
            activeTab === "suites"
              ? "border-primary text-primary"
              : "border-transparent text-muted-foreground hover:text-foreground"
          }`}
        >
          <ListOrdered className="h-4 w-4" />
          <span>🧪 Test Paketleri & Senaryolar ({suites.length})</span>
        </button>
      </div>

      {/* TAB 1: SINGLE REPLAY */}
      {activeTab === "single" && (
        <div className="space-y-6">
          {/* Info Alert Box */}
          <div className="rounded-2xl border border-blue-500/20 bg-blue-500/5 p-4 flex items-start gap-3">
            <ShieldCheck className="h-5 w-5 text-blue-400 shrink-0 mt-0.5" />
            <div className="text-xs space-y-1">
              <p className="font-bold text-foreground">Idempotency-Safe Güvenli Replay</p>
              <p className="text-muted-foreground leading-relaxed">
                Replay esnasında <code className="text-primary font-mono">renewIdempotencyKeys</code> aktif olduğunda hem HTTP başlıklarındaki (<code className="font-mono">Idempotency-Key</code>) hem de JSON gövdesindeki (<code className="font-mono">event_id, payment_id</code>) benzersiz kimlikler yeni UUIDv4 ile değiştirilerek upstream sunucunun isteği <strong>Duplicate</strong> olarak reddetmesi önlenir.
              </p>
            </div>
          </div>

          {/* New Replay Drawer */}
          {isNewReplayOpen && (
            <div className="rounded-2xl border border-border bg-card p-6 shadow-sm space-y-6">
              <div className="flex items-center justify-between border-b border-border pb-4">
                <div className="flex items-center gap-2">
                  <Zap className="h-5 w-5 text-amber-400" />
                  <h2 className="text-sm font-bold text-foreground">Replay Konfigürasyonu & Hedef Ortam</h2>
                </div>
                <button
                  onClick={() => setIsNewReplayOpen(false)}
                  className="text-xs text-muted-foreground hover:text-foreground"
                >
                  Kapat
                </button>
              </div>

              <form onSubmit={handleExecuteReplay} className="space-y-5">
                {/* Kaynak İstek */}
                <div>
                  <label className="block text-xs font-bold text-foreground mb-1.5">
                    1. Kaynak İstek (Canlı Yakalanan Webhook):
                  </label>
                  {requests.length === 0 ? (
                    <div className="text-xs text-muted-foreground p-3 rounded-xl border border-dashed border-border bg-muted/20">
                      Henüz yakalanmış bir istek bulunamadı.
                    </div>
                  ) : (
                    <select
                      value={selectedRequestId}
                      onChange={(e) => setSelectedRequestId(e.target.value)}
                      className="w-full rounded-xl border border-border bg-background px-3 py-2 text-xs font-semibold text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                    >
                      {requests.map((r) => (
                        <option key={r.id} value={r.id}>
                          #{r.requestId} — [{r.httpMethod}] {new Date(r.createdAt).toLocaleString()} (HTTP {r.responseStatus || 200})
                        </option>
                      ))}
                    </select>
                  )}
                </div>

                {/* Hedef Ortam */}
                <div>
                  <label className="block text-xs font-bold text-foreground mb-2">
                    2. Hedef Ortam (Target Environment):
                  </label>
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
                    {(["STAGING", "DEV", "LOCAL", "CUSTOM"] as ReplayEnvironment[]).map((env) => (
                      <button
                        key={env}
                        type="button"
                        onClick={() => handleEnvironmentSelect(env)}
                        className={`p-3 rounded-xl border text-left transition space-y-1 ${
                          environment === env
                            ? "border-primary bg-primary/10 ring-1 ring-primary"
                            : "border-border bg-card/60 hover:bg-muted/40"
                        }`}
                      >
                        <div className="flex items-center justify-between">
                          <span className="text-xs font-bold text-foreground">{env}</span>
                          {environment === env && <Check className="h-3.5 w-3.5 text-primary" />}
                        </div>
                        <p className="text-[10px] text-muted-foreground">
                          {env === "STAGING" && "Test / Ön-canlı"}
                          {env === "DEV" && "Geliştirme"}
                          {env === "LOCAL" && "Yerel localhost"}
                          {env === "CUSTOM" && "Özel URL"}
                        </p>
                      </button>
                    ))}
                  </div>
                </div>

                {/* Hedef URL */}
                <div>
                  <label className="block text-xs font-bold text-foreground mb-1.5">
                    3. Hedef Upstream URL:
                  </label>
                  <div className="relative">
                    <Globe className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                    <input
                      type="url"
                      required
                      value={targetUrl}
                      onChange={(e) => setTargetUrl(e.target.value)}
                      className="w-full rounded-xl border border-input bg-background/60 pl-9 pr-3 py-2 text-xs font-mono text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                </div>

                {/* Idempotency Safe Switch */}
                <div className="rounded-xl border border-border bg-muted/20 p-3.5 flex items-center justify-between">
                  <div className="space-y-0.5">
                    <div className="flex items-center gap-1.5">
                      <RefreshCw className="h-3.5 w-3.5 text-primary" />
                      <span className="text-xs font-bold text-foreground">
                        Idempotency Anahtarlarını Otomatik Yenile (Safe Mode)
                      </span>
                    </div>
                    <p className="text-[11px] text-muted-foreground">
                      Header ve Payload içindeki <code className="font-mono">Idempotency-Key</code>, <code className="font-mono">event_id</code> değerlerini benzersiz yeni UUID ile değiştirir.
                    </p>
                  </div>
                  <input
                    type="checkbox"
                    checked={renewIdempotency}
                    onChange={(e) => setRenewIdempotency(e.target.checked)}
                    className="h-4 w-4 rounded text-primary focus:ring-primary"
                  />
                </div>

                {/* Özel Başlıklar */}
                <div className="space-y-2">
                  <label className="block text-xs font-bold text-foreground">
                    4. Özel Başlıklar (Custom Headers):
                  </label>
                  <div className="flex flex-wrap items-center gap-2">
                    <input
                      type="text"
                      placeholder="Başlık (örn: Authorization)"
                      value={customHeaderKey}
                      onChange={(e) => setCustomHeaderKey(e.target.value)}
                      className="flex-1 min-w-[140px] rounded-xl border border-input bg-background/60 px-3 py-1.5 text-xs text-foreground"
                    />
                    <input
                      type="text"
                      placeholder="Değer (örn: Bearer token_...)"
                      value={customHeaderVal}
                      onChange={(e) => setCustomHeaderVal(e.target.value)}
                      className="flex-1 min-w-[140px] rounded-xl border border-input bg-background/60 px-3 py-1.5 text-xs text-foreground"
                    />
                    <button
                      type="button"
                      onClick={handleAddCustomHeader}
                      className="px-3 py-1.5 rounded-xl bg-secondary hover:bg-muted text-xs font-bold text-foreground transition flex items-center gap-1"
                    >
                      <Plus className="h-3.5 w-3.5" />
                      <span>Ekle</span>
                    </button>
                  </div>

                  {Object.keys(customHeaders).length > 0 && (
                    <div className="flex flex-wrap gap-2 pt-1">
                      {Object.entries(customHeaders).map(([k, v]) => (
                        <div
                          key={k}
                          className="flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-muted/40 border border-border text-[11px] font-mono"
                        >
                          <span className="text-primary font-bold">{k}:</span>
                          <span className="text-foreground truncate max-w-[180px]">{v}</span>
                          <button
                            type="button"
                            onClick={() => handleRemoveCustomHeader(k)}
                            className="text-muted-foreground hover:text-destructive ml-1"
                          >
                            <Trash2 className="h-3 w-3" />
                          </button>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                {/* Gerekçe */}
                <div>
                  <label className="block text-xs font-bold text-foreground mb-1.5">
                    5. İşlem Gerekçesi (Audit Justification):
                  </label>
                  <input
                    type="text"
                    value={justification}
                    onChange={(e) => setJustification(e.target.value)}
                    className="w-full rounded-xl border border-input bg-background/60 px-3 py-2 text-xs text-foreground"
                  />
                </div>

                {replayError && (
                  <div className="p-3 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-center gap-2">
                    <XCircle className="h-4 w-4 shrink-0" />
                    <span>{replayError}</span>
                  </div>
                )}

                <div className="flex justify-end gap-3 pt-2">
                  <button
                    type="button"
                    onClick={() => setIsNewReplayOpen(false)}
                    className="px-4 py-2 rounded-xl text-xs font-semibold text-muted-foreground hover:text-foreground"
                  >
                    İptal
                  </button>
                  <button
                    type="submit"
                    disabled={replayMutation.isPending || !selectedRequestId || !targetUrl.trim()}
                    className="flex items-center gap-2 rounded-xl bg-primary px-5 py-2 text-xs font-bold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
                  >
                    {replayMutation.isPending ? (
                      <>
                        <Loader2 className="h-4 w-4 animate-spin" />
                        <span>Ateşleniyor...</span>
                      </>
                    ) : (
                      <>
                        <Play className="h-4 w-4" />
                        <span>Replay Ateşle</span>
                      </>
                    )}
                  </button>
                </div>
              </form>

              {/* Sonuç Kartı */}
              {lastReplayResult && (
                <div className="mt-4 p-4 rounded-2xl border border-emerald-500/30 bg-emerald-500/5 space-y-3 animate-in fade-in duration-300">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <CheckCircle2 className="h-4 w-4 text-emerald-400" />
                      <span className="text-xs font-bold text-foreground">Replay Başarıyla Tamamlandı</span>
                    </div>
                    <span
                      className={`text-[10px] font-extrabold px-2 py-0.5 rounded-full border ${
                        lastReplayResult.responseStatus < 400
                          ? "bg-emerald-500/20 text-emerald-300 border-emerald-500/40"
                          : "bg-rose-500/20 text-rose-300 border-rose-500/40"
                      }`}
                    >
                      HTTP {lastReplayResult.responseStatus || "ERR"}
                    </span>
                  </div>

                  {lastReplayResult.replacements && Object.keys(lastReplayResult.replacements).length > 0 && (
                    <div className="p-2.5 rounded-xl bg-background/80 border border-border space-y-1">
                      <span className="text-[10px] uppercase font-bold text-primary block">
                        🔄 Yenilenen Idempotency Anahtarları:
                      </span>
                      <div className="space-y-0.5">
                        {Object.entries(lastReplayResult.replacements).map(([k, v]: any) => (
                          <div key={k} className="text-[11px] font-mono text-muted-foreground">
                            <span className="text-foreground font-bold">{k}:</span> {v}
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
                    <div>
                      <span className="text-muted-foreground block text-[10px] uppercase font-bold">Ortam:</span>
                      <span className="font-bold text-foreground">{lastReplayResult.environment}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground block text-[10px] uppercase font-bold">Gecikme:</span>
                      <span className="font-bold text-foreground">{lastReplayResult.latencyMs} ms</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground block text-[10px] uppercase font-bold">Hedef URL:</span>
                      <span className="font-mono text-primary truncate block">{lastReplayResult.targetUrl}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground block text-[10px] uppercase font-bold">Durum:</span>
                      <span className="font-bold text-emerald-400">{lastReplayResult.status}</span>
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Replay History Table */}
          <div className="rounded-2xl border border-border bg-card p-5 space-y-4 shadow-sm">
            <div className="flex items-center justify-between border-b border-border pb-3">
              <div className="flex items-center gap-2">
                <Clock className="h-4 w-4 text-primary" />
                <h2 className="text-sm font-bold text-foreground">Replay İşlem Geçmişi</h2>
              </div>
              <span className="text-xs text-muted-foreground">{replays.length} işlem kaydı</span>
            </div>

            {isReplaysLoading ? (
              <div className="py-12 text-center text-muted-foreground flex flex-col items-center gap-2">
                <Loader2 className="h-6 w-6 animate-spin text-primary" />
                <span className="text-xs">Replay geçmişi yükleniyor...</span>
              </div>
            ) : replays.length === 0 ? (
              <div className="py-12 text-center text-muted-foreground space-y-2">
                <Repeat className="h-10 w-10 mx-auto opacity-30" />
                <p className="text-xs font-bold text-foreground">Henüz bir replay işlemi yapılmadı</p>
              </div>
            ) : (
              <div className="divide-y divide-border">
                {replays.map((r) => (
                  <div
                    key={r.id}
                    onClick={() => setSelectedReplayDetail(r)}
                    className="py-3.5 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 hover:bg-muted/20 px-3 rounded-xl transition cursor-pointer"
                  >
                    <div className="flex items-start gap-3">
                      <div
                        className={`mt-1 flex h-7 w-7 items-center justify-center rounded-lg border text-xs font-bold ${
                          r.status === "COMPLETED" && (r.responseStatus || 0) < 400
                            ? "bg-emerald-500/10 border-emerald-500/20 text-emerald-400"
                            : "bg-rose-500/10 border-rose-500/20 text-rose-400"
                        }`}
                      >
                        {r.httpMethod || "POST"}
                      </div>

                      <div>
                        <div className="flex items-center gap-2">
                          <span className="text-xs font-bold text-foreground">
                            {r.endpointName || "Webhook"}
                          </span>
                          <span className="text-xs font-mono text-muted-foreground">
                            #{r.requestId || "req"}
                          </span>
                          <span
                            className={`text-[10px] font-extrabold px-1.5 py-0.5 rounded ${
                              r.environment === "STAGING"
                                ? "bg-blue-500/10 text-blue-400 border border-blue-500/30"
                                : r.environment === "LOCAL"
                                ? "bg-purple-500/10 text-purple-400 border border-purple-500/30"
                                : r.environment === "DEV"
                                ? "bg-amber-500/10 text-amber-400 border border-amber-500/30"
                                : "bg-muted text-muted-foreground"
                            }`}
                          >
                            {r.environment || "CUSTOM"}
                          </span>
                        </div>

                        <div className="text-[11px] text-muted-foreground flex items-center gap-3 mt-1">
                          <span className="font-mono truncate max-w-xs">{r.targetUrl}</span>
                          {r.latencyMs !== undefined && <span>{r.latencyMs} ms</span>}
                          <span>{new Date(r.createdAt).toLocaleString()}</span>
                        </div>
                      </div>
                    </div>

                    <div className="flex items-center gap-3 self-end sm:self-center">
                      <span
                        className={`text-xs font-mono font-bold px-2.5 py-1 rounded-lg border ${
                          (r.responseStatus || 0) < 400 && r.responseStatus
                            ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400"
                            : "bg-rose-500/10 border-rose-500/30 text-rose-400"
                        }`}
                      >
                        HTTP {r.responseStatus || "ERR"}
                      </span>
                      <ArrowRight className="h-4 w-4 text-muted-foreground" />
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* TAB 2: TEST SUITES & SCENARIOS */}
      {activeTab === "suites" && (
        <div className="space-y-6">
          {/* Create Suite Modal */}
          {isNewSuiteOpen && (
            <div className="rounded-2xl border border-border bg-card p-6 shadow-sm space-y-5">
              <div className="flex items-center justify-between border-b border-border pb-3">
                <div className="flex items-center gap-2">
                  <FolderPlus className="h-5 w-5 text-primary" />
                  <h3 className="text-sm font-bold text-foreground">Yeni Replay Test Paketi (Suite) Oluştur</h3>
                </div>
                <button
                  onClick={() => setIsNewSuiteOpen(false)}
                  className="text-xs text-muted-foreground hover:text-foreground"
                >
                  Kapat
                </button>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-xs font-bold text-foreground mb-1">Paket Adı:</label>
                  <input
                    type="text"
                    value={suiteName}
                    onChange={(e) => setSuiteName(e.target.value)}
                    placeholder="Örn: Checkout & Ödeme Regresyon Senaryosu"
                    className="w-full rounded-xl border border-input bg-background/60 px-3 py-2 text-xs text-foreground"
                  />
                </div>

                <div>
                  <label className="block text-xs font-bold text-foreground mb-1">Açıklama:</label>
                  <input
                    type="text"
                    value={suiteDescription}
                    onChange={(e) => setSuiteDescription(e.target.value)}
                    placeholder="Örn: Sipariş oluşturma ve ödeme webhook'larının staging doğrulaması"
                    className="w-full rounded-xl border border-input bg-background/60 px-3 py-2 text-xs text-foreground"
                  />
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div>
                    <label className="block text-xs font-bold text-foreground mb-1">Hedef Ortam:</label>
                    <select
                      value={suiteEnvironment}
                      onChange={(e) => setSuiteEnvironment(e.target.value as ReplayEnvironment)}
                      className="w-full rounded-xl border border-border bg-background px-3 py-2 text-xs font-semibold text-foreground"
                    >
                      <option value="STAGING">STAGING (Test / Ön-canlı)</option>
                      <option value="DEV">DEV (Geliştirme)</option>
                      <option value="LOCAL">LOCAL (Yerel Port 8080)</option>
                      <option value="CUSTOM">CUSTOM (Özel Adres)</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-xs font-bold text-foreground mb-1">Hedef URL:</label>
                    <input
                      type="url"
                      value={suiteTargetUrl}
                      onChange={(e) => setSuiteTargetUrl(e.target.value)}
                      className="w-full rounded-xl border border-input bg-background/60 px-3 py-2 text-xs font-mono text-foreground"
                    />
                  </div>
                </div>

                <div className="rounded-xl border border-border bg-muted/20 p-3 flex items-center justify-between">
                  <div className="text-xs space-y-0.5">
                    <span className="font-bold text-foreground">Idempotency-Safe Yenileme</span>
                    <p className="text-[11px] text-muted-foreground">Her adımda benzersiz event ve request ID'leri otomatik üretir.</p>
                  </div>
                  <input
                    type="checkbox"
                    checked={suiteRenewIdemp}
                    onChange={(e) => setSuiteRenewIdemp(e.target.checked)}
                    className="h-4 w-4 rounded text-primary focus:ring-primary"
                  />
                </div>

                {/* Request Selector Table */}
                <div>
                  <label className="block text-xs font-bold text-foreground mb-2">
                    Senaryoya Eklenecek İstekler ({suiteSelectedRequests.length} seçili):
                  </label>
                  <div className="max-h-56 overflow-y-auto rounded-xl border border-border divide-y divide-border">
                    {requests.map((r) => {
                      const isChecked = suiteSelectedRequests.includes(r.id);
                      return (
                        <div
                          key={r.id}
                          onClick={() => toggleSuiteRequest(r.id)}
                          className={`p-2.5 flex items-center justify-between text-xs cursor-pointer transition ${
                            isChecked ? "bg-primary/10" : "hover:bg-muted/20"
                          }`}
                        >
                          <div className="flex items-center gap-2.5">
                            <input
                              type="checkbox"
                              checked={isChecked}
                              onChange={() => {}}
                              className="h-3.5 w-3.5 rounded text-primary"
                            />
                            <span className="font-mono font-bold text-foreground">#{r.requestId}</span>
                            <span className="text-muted-foreground font-mono">[{r.httpMethod}]</span>
                            <span className="text-muted-foreground">{new Date(r.createdAt).toLocaleTimeString()}</span>
                          </div>
                          <span className="text-[11px] font-mono text-primary font-bold">
                            HTTP {r.responseStatus || 200}
                          </span>
                        </div>
                      );
                    })}
                  </div>
                </div>

                <div className="flex justify-end gap-3 pt-2">
                  <button
                    type="button"
                    onClick={() => setIsNewSuiteOpen(false)}
                    className="px-4 py-2 rounded-xl text-xs font-semibold text-muted-foreground hover:text-foreground"
                  >
                    İptal
                  </button>
                  <button
                    type="button"
                    disabled={createSuiteMutation.isPending || !suiteName.trim() || suiteSelectedRequests.length === 0}
                    onClick={() =>
                      createSuiteMutation.mutate({
                        name: suiteName.trim(),
                        description: suiteDescription.trim(),
                        requestIds: suiteSelectedRequests,
                        targetEnvironment: suiteEnvironment,
                        targetUrl: suiteTargetUrl.trim(),
                        renewIdempotency: suiteRenewIdemp,
                      })
                    }
                    className="flex items-center gap-2 rounded-xl bg-primary px-5 py-2 text-xs font-bold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
                  >
                    {createSuiteMutation.isPending ? (
                      <>
                        <Loader2 className="h-4 w-4 animate-spin" />
                        <span>Kaydediliyor...</span>
                      </>
                    ) : (
                      <>
                        <Check className="h-4 w-4" />
                        <span>Paketi Kaydet</span>
                      </>
                    )}
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* Test Suite Run Report Modal */}
          {suiteRunReport && (
            <div className="rounded-2xl border border-border bg-card p-6 shadow-sm space-y-4 animate-in fade-in duration-300">
              <div className="flex items-center justify-between border-b border-border pb-3">
                <div className="flex items-center gap-2">
                  <Sparkles className="h-5 w-5 text-primary" />
                  <h3 className="text-sm font-bold text-foreground">
                    Test Paketi Raporu: {suiteRunReport.suiteName}
                  </h3>
                </div>
                <button
                  onClick={() => setSuiteRunReport(null)}
                  className="text-xs text-muted-foreground hover:text-foreground"
                >
                  Kapat
                </button>
              </div>

              {/* Progress Summary */}
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
                <div className="p-3 rounded-xl bg-muted/20 border border-border">
                  <span className="text-[10px] uppercase font-bold text-muted-foreground block">Toplam Adım:</span>
                  <span className="text-sm font-bold text-foreground">{suiteRunReport.totalSteps}</span>
                </div>
                <div className="p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/20">
                  <span className="text-[10px] uppercase font-bold text-emerald-400 block">Başarılı Adım:</span>
                  <span className="text-sm font-bold text-emerald-400">{suiteRunReport.passedSteps}</span>
                </div>
                <div className="p-3 rounded-xl bg-rose-500/10 border border-rose-500/20">
                  <span className="text-[10px] uppercase font-bold text-rose-400 block">Hatalı Adım:</span>
                  <span className="text-sm font-bold text-rose-400">{suiteRunReport.failedSteps}</span>
                </div>
                <div className="p-3 rounded-xl bg-muted/20 border border-border">
                  <span className="text-[10px] uppercase font-bold text-muted-foreground block">Toplam Gecikme:</span>
                  <span className="text-sm font-bold text-foreground">{suiteRunReport.totalLatencyMs} ms</span>
                </div>
              </div>

              {/* Step by Step Breakdown */}
              <div className="space-y-2">
                <span className="text-xs font-bold text-foreground block">Adım Detayları:</span>
                <div className="divide-y divide-border rounded-xl border border-border overflow-hidden">
                  {suiteRunReport.stepResults.map((step) => (
                    <div key={step.stepIndex} className="p-3 bg-card/60 flex items-center justify-between text-xs">
                      <div className="flex items-center gap-3">
                        <span
                          className={`flex h-6 w-6 items-center justify-center rounded-full text-[11px] font-bold ${
                            step.status === "PASSED"
                              ? "bg-emerald-500/20 text-emerald-400"
                              : "bg-rose-500/20 text-rose-400"
                          }`}
                        >
                          {step.stepIndex}
                        </span>
                        <div>
                          <span className="font-bold text-foreground">İstek #{step.requestId}</span>
                          <span className="text-muted-foreground font-mono ml-2 text-[11px]">{step.targetUrl}</span>
                        </div>
                      </div>

                      <div className="flex items-center gap-3">
                        <span className="text-[11px] text-muted-foreground">{step.latencyMs} ms</span>
                        <span
                          className={`font-mono font-bold px-2 py-0.5 rounded text-[11px] border ${
                            step.responseStatus < 400 && step.responseStatus
                              ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400"
                              : "bg-rose-500/10 border-rose-500/30 text-rose-400"
                          }`}
                        >
                          HTTP {step.responseStatus || "ERR"}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Test Suites List */}
          <div className="rounded-2xl border border-border bg-card p-5 space-y-4 shadow-sm">
            <div className="flex items-center justify-between border-b border-border pb-3">
              <div className="flex items-center gap-2">
                <Layers className="h-4 w-4 text-primary" />
                <h2 className="text-sm font-bold text-foreground">Kayıtlı Test Paketleri</h2>
              </div>
              <span className="text-xs text-muted-foreground">{suites.length} paket</span>
            </div>

            {isSuitesLoading ? (
              <div className="py-12 text-center text-muted-foreground flex flex-col items-center gap-2">
                <Loader2 className="h-6 w-6 animate-spin text-primary" />
                <span className="text-xs">Test paketleri yükleniyor...</span>
              </div>
            ) : suites.length === 0 ? (
              <div className="py-12 text-center text-muted-foreground space-y-2">
                <ListOrdered className="h-10 w-10 mx-auto opacity-30" />
                <p className="text-xs font-bold text-foreground">Henüz kayıtlı bir test paketi yok</p>
                <p className="text-xs text-muted-foreground">
                  Yukarıdaki <strong>"Yeni Test Paketi Oluştur"</strong> butonuyla birden fazla webhook isteğini sıralı bir senaryoya dönüştürebilirsiniz.
                </p>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {suites.map((suite) => (
                  <div
                    key={suite.id}
                    className="p-4 rounded-2xl border border-border bg-card/60 space-y-3 hover:border-primary/40 transition"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <h3 className="text-xs font-bold text-foreground">{suite.name}</h3>
                        {suite.description && (
                          <p className="text-[11px] text-muted-foreground mt-0.5">{suite.description}</p>
                        )}
                      </div>
                      <span
                        className={`text-[10px] font-extrabold px-1.5 py-0.5 rounded ${
                          suite.targetEnvironment === "STAGING"
                            ? "bg-blue-500/10 text-blue-400 border border-blue-500/30"
                            : "bg-purple-500/10 text-purple-400 border border-purple-500/30"
                        }`}
                      >
                        {suite.targetEnvironment}
                      </span>
                    </div>

                    <div className="text-[11px] text-muted-foreground space-y-1 font-mono">
                      <div>İstek Sayısı: <span className="text-foreground font-bold">{suite.requestIds.length} adım</span></div>
                      <div className="truncate">Hedef URL: <span className="text-primary">{suite.targetUrl || "Varsayılan"}</span></div>
                    </div>

                    <div className="flex items-center justify-between pt-2 border-t border-border">
                      <span className="text-[10px] text-muted-foreground">
                        {new Date(suite.createdAt).toLocaleDateString()}
                      </span>
                      <button
                        type="button"
                        disabled={runningSuiteId === suite.id}
                        onClick={() => runSuiteMutation.mutate(suite.id)}
                        className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl bg-primary text-primary-foreground text-xs font-bold transition hover:bg-primary/90 disabled:opacity-50"
                      >
                        {runningSuiteId === suite.id ? (
                          <>
                            <Loader2 className="h-3.5 w-3.5 animate-spin" />
                            <span>Koşuluyor...</span>
                          </>
                        ) : (
                          <>
                            <Play className="h-3.5 w-3.5" />
                            <span>Senaryoyu Koştur</span>
                          </>
                        )}
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Replay Details Drawer / Modal */}
      {selectedReplayDetail && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-in fade-in duration-200">
          <div className="w-full max-w-2xl rounded-2xl border border-border bg-card p-6 shadow-xl space-y-4 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between border-b border-border pb-3">
              <div className="flex items-center gap-2">
                <Activity className="h-5 w-5 text-primary" />
                <h3 className="text-sm font-bold text-foreground">Replay Telemetri Detayı</h3>
              </div>
              <button
                onClick={() => setSelectedReplayDetail(null)}
                className="text-xs text-muted-foreground hover:text-foreground"
              >
                Kapat
              </button>
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
              <div>
                <span className="text-[10px] uppercase font-bold text-muted-foreground block">Ortam:</span>
                <span className="font-bold text-foreground">{selectedReplayDetail.environment || "CUSTOM"}</span>
              </div>
              <div>
                <span className="text-[10px] uppercase font-bold text-muted-foreground block">Yanıt Kodu:</span>
                <span className="font-bold text-foreground">HTTP {selectedReplayDetail.responseStatus || "ERR"}</span>
              </div>
              <div>
                <span className="text-[10px] uppercase font-bold text-muted-foreground block">Gecikme:</span>
                <span className="font-bold text-foreground">{selectedReplayDetail.latencyMs || 0} ms</span>
              </div>
              <div>
                <span className="text-[10px] uppercase font-bold text-muted-foreground block">Tarih:</span>
                <span className="text-muted-foreground">{new Date(selectedReplayDetail.createdAt).toLocaleString()}</span>
              </div>
            </div>

            <div>
              <span className="text-[10px] uppercase font-bold text-muted-foreground block mb-1">Hedef URL:</span>
              <code className="text-xs font-mono text-primary p-2 rounded-lg bg-background border border-border block">
                {selectedReplayDetail.targetUrl}
              </code>
            </div>

            {selectedReplayDetail.responseBody && (
              <div>
                <span className="text-[10px] uppercase font-bold text-muted-foreground block mb-1">Upstream Yanıt Gövdesi:</span>
                <pre className="p-3 rounded-xl bg-background border border-border text-xs font-mono text-muted-foreground max-h-48 overflow-auto">
                  {selectedReplayDetail.responseBody}
                </pre>
              </div>
            )}

            <div className="flex justify-end pt-2">
              <button
                type="button"
                onClick={() => setSelectedReplayDetail(null)}
                className="px-4 py-2 rounded-xl bg-secondary text-xs font-bold text-foreground hover:bg-muted"
              >
                Kapat
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
