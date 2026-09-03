"use client";

import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import { Project, CapturedRequest, ReplayJob, ReplayEnvironment } from "@apisentinel/shared";
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
} from "lucide-react";
import { useActiveProject } from "../../../contexts/ProjectContext";

export default function ReplayPage() {
  const queryClient = useQueryClient();
  const { accessToken, organization } = useAuth();
  const { projects, activeProjectId } = useActiveProject();

  const [isNewReplayOpen, setIsNewReplayOpen] = useState(false);
  const [selectedRequestId, setSelectedRequestId] = useState<string>("");
  const [environment, setEnvironment] = useState<ReplayEnvironment>("STAGING");
  const [targetUrl, setTargetUrl] = useState<string>("https://httpbin.org/post");
  const [justification, setJustification] = useState<string>("Staging ortamında yeni sürüm testi");
  const [customHeaderKey, setCustomHeaderKey] = useState<string>("");
  const [customHeaderVal, setCustomHeaderVal] = useState<string>("");
  const [customHeaders, setCustomHeaders] = useState<Record<string, string>>({
    "X-ApiSentinel-Replay-Env": "staging",
  });

  const [replayError, setReplayError] = useState<string | null>(null);
  const [lastReplayResult, setLastReplayResult] = useState<any | null>(null);
  const [selectedReplayDetail, setSelectedReplayDetail] = useState<ReplayJob | null>(null);
  const [copied, setCopied] = useState<boolean>(false);

  // 1. Fetch captured requests for selection
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
  const { data: replaysData, isLoading } = useQuery({
    queryKey: ["replays", activeProjectId],
    queryFn: () =>
      apiFetch<{ replays: ReplayJob[] }>(`/api/projects/${activeProjectId}/replays`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
  });

  const replays = replaysData?.replays || [];

  // Replay mutation
  const replayMutation = useMutation({
    mutationFn: (vars: {
      requestId: string;
      targetUrl: string;
      environment: ReplayEnvironment;
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

  const handleEnvironmentSelect = (env: ReplayEnvironment) => {
    setEnvironment(env);
    if (env === "STAGING") {
      setTargetUrl("https://httpbin.org/post");
      setCustomHeaders((prev) => ({ ...prev, "X-ApiSentinel-Replay-Env": "staging" }));
    } else if (env === "LOCAL") {
      setTargetUrl("http://localhost:8080/api/webhooks");
      setCustomHeaders((prev) => ({ ...prev, "X-ApiSentinel-Replay-Env": "local" }));
    } else if (env === "DEV") {
      setTargetUrl("https://dev.api.internal/webhook");
      setCustomHeaders((prev) => ({ ...prev, "X-ApiSentinel-Replay-Env": "dev" }));
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
      customHeaders,
      justification,
    });
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
              Replay Lab (Prod-to-Dev / Staging Yönlendirici)
            </h1>
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            Canlı ortamda yakalanan webhook isteklerini Staging, Dev veya Local hedeflere güvenle yeniden ateşleyin
          </p>
        </div>

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
      </div>

      {/* Info Alert Box */}
      <div className="rounded-2xl border border-blue-500/20 bg-blue-500/5 p-4 flex items-start gap-3">
        <ShieldCheck className="h-5 w-5 text-blue-400 shrink-0 mt-0.5" />
        <div className="text-xs space-y-1">
          <p className="font-bold text-foreground">Güvenli ve İzole Replay Simülasyonu</p>
          <p className="text-muted-foreground leading-relaxed">
            Replay istekleri <code className="text-primary font-mono">X-ApiSentinel-Replayed: true</code> başlığı ile işaretlenir ve SSRF koruması ile taranır. Yapılan tüm replay işlemleri gerekçesiyle birlikte <strong>Audit Trail</strong> günlüğüne işlenir.
          </p>
        </div>
      </div>

      {/* Execution Drawer / Modal */}
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
            {/* Step 1: Select Source Request */}
            <div>
              <label className="block text-xs font-bold text-foreground mb-1.5">
                1. Kaynak İstek (Canlı Yakalanan Webhook):
              </label>
              {requests.length === 0 ? (
                <div className="text-xs text-muted-foreground p-3 rounded-xl border border-dashed border-border bg-muted/20">
                  Henüz yakalanmış bir istek bulunamadı. Önce bir webhook gönderin.
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

            {/* Step 2: Target Environment Preset */}
            <div>
              <label className="block text-xs font-bold text-foreground mb-2">
                2. Hedef Ortam (Environment):
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
                      {env === "STAGING" && "Test / Ön-canlı sunucusu"}
                      {env === "DEV" && "Geliştirme ortamı"}
                      {env === "LOCAL" && "Yerel localhost / tünel"}
                      {env === "CUSTOM" && "Özel URL adresi"}
                    </p>
                  </button>
                ))}
              </div>
            </div>

            {/* Step 3: Target URL */}
            <div>
              <label className="block text-xs font-bold text-foreground mb-1.5">
                3. Hedef Upstream URL (Target URL Override):
              </label>
              <div className="relative">
                <Globe className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                <input
                  type="url"
                  required
                  value={targetUrl}
                  onChange={(e) => setTargetUrl(e.target.value)}
                  placeholder="https://staging.internal/api/webhook"
                  className="w-full rounded-xl border border-input bg-background/60 pl-9 pr-3 py-2 text-xs font-mono text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
            </div>

            {/* Step 4: Custom Headers */}
            <div className="space-y-2">
              <label className="block text-xs font-bold text-foreground">
                4. Özel Başlıklar (Custom Replay Headers):
              </label>
              <div className="flex flex-wrap items-center gap-2">
                <input
                  type="text"
                  placeholder="Başlık (örn: Authorization)"
                  value={customHeaderKey}
                  onChange={(e) => setCustomHeaderKey(e.target.value)}
                  className="flex-1 min-w-[140px] rounded-xl border border-input bg-background/60 px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                />
                <input
                  type="text"
                  placeholder="Değer (örn: Bearer dev_token)"
                  value={customHeaderVal}
                  onChange={(e) => setCustomHeaderVal(e.target.value)}
                  className="flex-1 min-w-[140px] rounded-xl border border-input bg-background/60 px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
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

              {/* Injected headers badge list */}
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

            {/* Step 5: Justification */}
            <div>
              <label className="block text-xs font-bold text-foreground mb-1.5">
                5. İşlem Gerekçesi (Audit Justification):
              </label>
              <input
                type="text"
                value={justification}
                onChange={(e) => setJustification(e.target.value)}
                placeholder="Örn: Staging ödeme akışı doğrulaması"
                className="w-full rounded-xl border border-input bg-background/60 px-3 py-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>

            {/* Error Message */}
            {replayError && (
              <div className="p-3 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive text-xs flex items-center gap-2">
                <XCircle className="h-4 w-4 shrink-0" />
                <span>{replayError}</span>
              </div>
            )}

            {/* Action Bar */}
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

          {/* Last Result Card */}
          {lastReplayResult && (
            <div className="mt-4 p-4 rounded-2xl border border-emerald-500/30 bg-emerald-500/5 space-y-3 animate-in fade-in duration-300">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-emerald-400" />
                  <span className="text-xs font-bold text-foreground">Replay Başarıyla Gerçekleştirildi</span>
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

              {lastReplayResult.responseBody && (
                <div className="space-y-1">
                  <span className="text-[10px] uppercase font-bold text-muted-foreground">Upstream Yanıt Gövdesi:</span>
                  <pre className="p-3 rounded-xl bg-background border border-border text-[11px] font-mono text-muted-foreground max-h-36 overflow-auto">
                    {lastReplayResult.responseBody}
                  </pre>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Replay History Table */}
      <div className="rounded-2xl border border-border bg-card p-5 space-y-4 shadow-sm">
        <div className="flex items-center justify-between border-b border-border pb-3">
          <div className="flex items-center gap-2">
            <Clock className="h-4 w-4 text-primary" />
            <h2 className="text-sm font-bold text-foreground">Replay İşlem Geçmişi (Replay Executions)</h2>
          </div>
          <span className="text-xs text-muted-foreground">{replays.length} işlem kaydı</span>
        </div>

        {isLoading ? (
          <div className="py-12 text-center text-muted-foreground flex flex-col items-center gap-2">
            <Loader2 className="h-6 w-6 animate-spin text-primary" />
            <span className="text-xs">Replay geçmişi yükleniyor...</span>
          </div>
        ) : replays.length === 0 ? (
          <div className="py-12 text-center text-muted-foreground space-y-2">
            <Repeat className="h-10 w-10 mx-auto opacity-30" />
            <p className="text-xs font-bold text-foreground">Henüz bir replay işlemi yapılmadı</p>
            <p className="text-xs text-muted-foreground max-w-sm mx-auto">
              Yukarıdaki <strong>"Yeni Replay Ateşle"</strong> butonuna tıklayarak canlı bir isteği Staging veya Dev hedefine yönlendirebilirsiniz.
            </p>
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

            {selectedReplayDetail.customHeaders && Object.keys(selectedReplayDetail.customHeaders).length > 0 && (
              <div>
                <span className="text-[10px] uppercase font-bold text-muted-foreground block mb-1">Enjekte Edilen Özel Başlıklar:</span>
                <pre className="p-3 rounded-xl bg-background border border-border text-xs font-mono text-muted-foreground">
                  {JSON.stringify(selectedReplayDetail.customHeaders, null, 2)}
                </pre>
              </div>
            )}

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
