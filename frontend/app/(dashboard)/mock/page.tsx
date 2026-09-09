"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import { Project, Endpoint, CapturedRequest } from "@apisentinel/shared";
import {
  Sparkles,
  Plus,
  Radio,
  Clock,
  Code,
  FileJson,
  Layers,
  Loader2,
  CheckCircle2,
  AlertCircle,
  Zap,
  AlignLeft,
  XCircle,
  Trash2,
  Send,
  Copy,
  Check,
  ExternalLink,
  Power,
  Activity,
  Terminal,
} from "lucide-react";

import { useActiveProject } from "../../../contexts/ProjectContext";

interface MockRuleItem {
  id: string;
  endpointId: string;
  name: string;
  statusCode: number;
  delayMs: number;
  responseHeaders: Record<string, string>;
  responseBody: Record<string, any>;
  enabled: boolean;
}

export default function MockPage() {
  const queryClient = useQueryClient();
  const { accessToken, organization } = useAuth();
  const { projects, activeProjectId, setActiveProjectId } = useActiveProject();

  const [selectedEndpointId, setSelectedEndpointId] = useState<string>("");
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [copiedUrl, setCopiedUrl] = useState(false);

  // Per-Rule PowerShell & Test State
  const [testingRuleId, setTestingRuleId] = useState<string | null>(null);
  const [copiedRuleId, setCopiedRuleId] = useState<string | null>(null);
  const [ruleTestResults, setRuleTestResults] = useState<Record<string, {
    status: number;
    statusText: string;
    latencyMs: number;
    headers: Record<string, string>;
    body: any;
    timestamp: string;
  }>>({});

  // Form State
  const [ruleName, setRuleName] = useState("");
  const [statusCode, setStatusCode] = useState(200);
  const [delayMs, setDelayMs] = useState(0);
  const [responseBodyText, setResponseBodyText] = useState(
    JSON.stringify({ status: "mocked", message: "ApiSentinel Mock Response" }, null, 2)
  );
  const [createError, setCreateError] = useState<string | null>(null);

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
  const activeEndpoint = endpoints.find((e) => e.id === activeEndpointId);
  const isMockMode = activeEndpoint?.mode === "MOCK";

  const backendUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3001";
  const webhookUrl = activeEndpoint ? `${backendUrl}/hook/${activeEndpoint.slug}` : "";

  // Fetch mock rules for active endpoint
  const { data: mocksData, isLoading } = useQuery({
    queryKey: ["mocks", activeEndpointId],
    queryFn: () =>
      apiFetch<{ mocks: MockRuleItem[] }>(`/api/endpoints/${activeEndpointId}/mocks`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeEndpointId,
  });

  // Fetch recent captured requests for this project to show live mock history
  const { data: requestsData, refetch: refetchRequests } = useQuery({
    queryKey: ["requests", activeProjectId],
    queryFn: () =>
      apiFetch<{ requests: (CapturedRequest & { endpoint?: { name: string; slug: string } })[] }>(
        `/api/projects/${activeProjectId}/requests?limit=30`,
        {
          token: accessToken,
          organizationId: organization?.id,
        }
      ),
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
    refetchInterval: 3000,
  });

  const mockRequests = (requestsData?.requests || []).filter(
    (r) =>
      (r.endpointId === activeEndpointId || (r as any).endpoint_id === activeEndpointId) &&
      (r.processingStatus === "MOCKED" || (r as any).processing_status === "MOCKED")
  );

  // Create Mock Rule Mutation
  const createMutation = useMutation({
    mutationFn: (input: any) =>
      apiFetch<MockRuleItem>(`/api/endpoints/${activeEndpointId}/mocks`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mocks", activeEndpointId] });
      setRuleName("");
      setDelayMs(0);
      setStatusCode(200);
      setIsCreateOpen(false);
      setCreateError(null);
    },
    onError: (err: any) => {
      setCreateError(err.message || "Mock kuralı oluşturulamadı.");
    },
  });

  // Mode Toggle Mutation (Toggle between PASS and MOCK directly from Mock Lab)
  const toggleModeMutation = useMutation({
    mutationFn: (newMode: "PASS" | "MOCK") =>
      apiFetch<Endpoint>(`/api/projects/${activeProjectId}/endpoints/${activeEndpointId}`, {
        method: "PUT",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify({
          name: activeEndpoint?.name,
          mode: newMode,
          isActive: Boolean(activeEndpoint?.isActive ?? (activeEndpoint as any)?.is_active),
          upstreamUrl: activeEndpoint?.upstreamUrl || null,
          maxPayloadSizeBytes: activeEndpoint?.maxPayloadSizeBytes,
          rateLimitRpm: activeEndpoint?.rateLimitRpm,
          burstThreshold: activeEndpoint?.burstThreshold,
        }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["endpoints", activeProjectId] });
    },
  });

  // Toggle Single Rule Enabled/Disabled Mutation
  const toggleRuleMutation = useMutation({
    mutationFn: (ruleId: string) =>
      apiFetch<MockRuleItem>(`/api/endpoints/${activeEndpointId}/mocks/${ruleId}/toggle`, {
        method: "PATCH",
        token: accessToken,
        organizationId: organization?.id,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mocks", activeEndpointId] });
    },
  });

  // Delete Mock Rule Mutation
  const deleteRuleMutation = useMutation({
    mutationFn: (ruleId: string) =>
      apiFetch(`/api/endpoints/${activeEndpointId}/mocks/${ruleId}`, {
        method: "DELETE",
        token: accessToken,
        organizationId: organization?.id,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["mocks", activeEndpointId] });
    },
  });

  const handleCopyUrl = () => {
    if (!webhookUrl) return;
    navigator.clipboard.writeText(webhookUrl);
    setCopiedUrl(true);
    setTimeout(() => setCopiedUrl(false), 2000);
  };

  function getRuleMatchingPayload(ruleName: string, statusCode: number): Record<string, any> {
    const lower = ruleName.toLowerCase();
    if (statusCode >= 200 && statusCode < 300) {
      return {
        event: "payment.succeeded",
        amount: 2500,
        currency: "TRY",
        customer: {
          email: "ahmet@example.com",
          name: "Ahmet Yılmaz"
        },
        transaction_id: "tx_test_1001"
      };
    }
    if (statusCode === 400 || statusCode === 422 || lower.includes("bad request") || lower.includes("parametre")) {
      return {
        event: "payment.failed",
        error_trigger: "invalid_payload",
        missing_required_field: "signature",
        amount: -50
      };
    }
    if (statusCode === 401 || lower.includes("unauthorized") || lower.includes("hmac")) {
      return {
        event: "payment.unauthorized",
        token: "invalid_mock_signature"
      };
    }
    if (statusCode === 429 || lower.includes("rate limit") || lower.includes("aşıl")) {
      return {
        event: "traffic.burst_test",
        burst_sequence: 150,
        client_tag: "rate_limit_probe"
      };
    }
    if (statusCode >= 500 || lower.includes("bakım") || lower.includes("outage") || lower.includes("hata")) {
      return {
        event: "system.gateway_healthcheck",
        service: "payment_upstream",
        maintenance_probe: true
      };
    }
    return {
      event: "custom.mock_trigger",
      rule: ruleName,
      status: statusCode
    };
  }

  function generatePowerShellCurl(url: string, payload: Record<string, any>): string {
    const jsonStr = JSON.stringify(payload);
    return `curl.exe -i -X POST "${url}" -H "Content-Type: application/json" -d '${jsonStr}'`;
  }

  const handleRunRuleSimulation = async (rule: MockRuleItem) => {
    if (!webhookUrl) return;
    setTestingRuleId(rule.id);

    // If rule is not enabled, enable it first so backend mock engine routes to this rule
    if (!rule.enabled) {
      await toggleRuleMutation.mutateAsync(rule.id);
    }

    const payload = getRuleMatchingPayload(rule.name, rule.statusCode);
    const startTime = performance.now();
    try {
      const res = await fetch(webhookUrl, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Simulation-Client": "ApiSentinel-Mock-Lab",
        },
        body: JSON.stringify(payload),
      });

      const latencyMs = Math.round(performance.now() - startTime);
      const headersMap: Record<string, string> = {};
      res.headers.forEach((val, key) => {
        headersMap[key] = val;
      });

      let jsonResp: any = null;
      const rawText = await res.text();
      try {
        jsonResp = JSON.parse(rawText);
      } catch {
        jsonResp = rawText;
      }

      setRuleTestResults((prev) => ({
        ...prev,
        [rule.id]: {
          status: res.status,
          statusText: res.statusText || (res.status === 200 ? "OK" : "Status"),
          latencyMs,
          headers: headersMap,
          body: jsonResp,
          timestamp: new Date().toLocaleTimeString("tr-TR"),
        },
      }));

      // Refetch live requests
      setTimeout(() => {
        refetchRequests();
        queryClient.invalidateQueries({ queryKey: ["requests", activeProjectId] });
      }, 500);
    } catch (err: any) {
      setRuleTestResults((prev) => ({
        ...prev,
        [rule.id]: {
          status: 500,
          statusText: "İstek Hatası",
          latencyMs: 0,
          headers: {},
          body: { error: err.message || "İstek sunucuya ulaştırılamadı" },
          timestamp: new Date().toLocaleTimeString("tr-TR"),
        },
      }));
    } finally {
      setTestingRuleId(null);
    }
  };

  const handleCopyPowerShell = (rule: MockRuleItem) => {
    if (!webhookUrl) return;
    const payload = getRuleMatchingPayload(rule.name, rule.statusCode);
    const cmd = generatePowerShellCurl(webhookUrl, payload);
    navigator.clipboard.writeText(cmd);
    setCopiedRuleId(rule.id);
    setTimeout(() => setCopiedRuleId(null), 2500);
  };

  const handleApplyPreset = (code: number, name: string, body: any) => {
    setStatusCode(code);
    setRuleName(name);
    setResponseBodyText(JSON.stringify(body, null, 2));
  };

  const handlePrettify = () => {
    try {
      const parsed = JSON.parse(responseBodyText);
      setResponseBodyText(JSON.stringify(parsed, null, 2));
      setCreateError(null);
    } catch {
      setCreateError("Prettify yapılamadı. JSON formatı hatalı.");
    }
  };

  const handleCreateMock = (e: React.FormEvent) => {
    e.preventDefault();
    if (!ruleName.trim()) return;

    let parsedBody = {};
    try {
      parsedBody = JSON.parse(responseBodyText);
    } catch {
      setCreateError("Response Body geçerli bir JSON formatında olmalıdır.");
      return;
    }

    createMutation.mutate({
      name: ruleName.trim(),
      statusCode: Number(statusCode),
      delayMs: Number(delayMs),
      responseHeaders: { "Content-Type": "application/json", "X-ApiSentinel-Mock": "true" },
      responseBody: parsedBody,
      enabled: true,
    });
  };

  const mockRules = mocksData?.mocks || [];

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight">Mock Lab (Simulator Engine)</h1>
            <span className="flex items-center gap-1 rounded-full bg-purple-500/10 px-2.5 py-0.5 text-xs font-semibold text-purple-400">
              <Sparkles className="h-3 w-3" />
              Dynamic Simulator
            </span>
          </div>
          <p className="text-sm text-muted-foreground mt-1">
            Webhook sağlayıcılarını test etmek için özel HTTP yanıtları, hata durumları (503, 429) ve gecikmeler simüle edin
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          {endpoints.length > 0 && (
            <div className="flex items-center gap-2">
              <select
                value={activeEndpointId}
                onChange={(e) => setSelectedEndpointId(e.target.value)}
                className="rounded-xl border border-border bg-card px-3 py-2 text-xs font-semibold text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              >
                {endpoints.map((ep) => (
                  <option key={ep.id} value={ep.id}>
                    {ep.name} (/hook/{ep.slug})
                  </option>
                ))}
              </select>

              <button
                type="button"
                onClick={() => toggleModeMutation.mutate(isMockMode ? "PASS" : "MOCK")}
                disabled={toggleModeMutation.isPending || !activeEndpointId}
                className={`flex items-center gap-1.5 rounded-xl border px-3 py-2 text-xs font-bold transition shadow-sm cursor-pointer disabled:opacity-50 ${
                  isMockMode
                    ? "border-purple-500/30 bg-purple-500/15 text-purple-400 hover:bg-purple-500/25"
                    : "border-border bg-card text-muted-foreground hover:text-foreground hover:bg-muted/40"
                }`}
                title={isMockMode ? "Mock Modu devrede (Gelen istekler sahte yanıt alır). Kapatıp normal moda geçmek için tıklayın." : "Mock Modu kapalı (Normal mod). Simülasyonu başlatmak için tıklayın."}
              >
                {toggleModeMutation.isPending ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Radio className={`h-3.5 w-3.5 ${isMockMode ? "text-purple-400 animate-pulse" : "text-muted-foreground"}`} />
                )}
                <span>{isMockMode ? "MOCK MODU: AKTİF" : "MOCK MODU: KAPALI"}</span>
              </button>
            </div>
          )}

          <button
            onClick={() => setIsCreateOpen(!isCreateOpen)}
            disabled={!activeEndpointId}
            className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-xs font-bold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
          >
            <Plus className="h-4 w-4" />
            <span>Yeni Mock Kuralı</span>
          </button>
        </div>
      </div>

      {/* Target Endpoint & Webhook URL Bar */}
      {activeEndpoint && (
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-2xl border border-border bg-card/60 p-4 shadow-sm">
          <div className="flex items-center gap-3">
            <span className="rounded-lg bg-secondary px-2.5 py-1 text-xs font-mono font-bold text-foreground">
              POST
            </span>
            <div className="font-mono text-xs text-foreground truncate max-w-md md:max-w-xl">
              {webhookUrl}
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={handleCopyUrl}
              className="flex items-center gap-1.5 rounded-xl border border-border bg-secondary/50 px-3 py-1.5 text-xs font-medium text-foreground hover:bg-secondary transition"
            >
              {copiedUrl ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5 text-muted-foreground" />}
              <span>{copiedUrl ? "Kopyalandı!" : "URL Kopyala"}</span>
            </button>

            <Link
              href={`/requests?endpointId=${activeEndpoint.id}`}
              className="flex items-center gap-1.5 rounded-xl border border-border bg-secondary/50 px-3 py-1.5 text-xs font-medium text-foreground hover:bg-secondary transition"
            >
              <ExternalLink className="h-3.5 w-3.5 text-muted-foreground" />
              <span>Canlı İstekler</span>
            </Link>
          </div>
        </div>
      )}

      {/* Create Mock Modal / Drawer */}
      {isCreateOpen && (
        <div className="rounded-2xl border border-border bg-card p-6 shadow-md animate-in fade-in duration-200 space-y-4">
          <div className="flex items-center justify-between border-b border-border pb-4 mb-2">
            <h3 className="text-base font-bold">Yeni Mock Yanıt Kuralı Tanımla</h3>
            <button
              onClick={() => {
                setIsCreateOpen(false);
                setCreateError(null);
              }}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              Kapat
            </button>
          </div>

          {/* Quick Presets Bar */}
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-[10px] uppercase font-bold text-muted-foreground mr-1">Hazır Şablonlar:</span>
            <button
              type="button"
              onClick={() => handleApplyPreset(200, "200 OK Ödeme Başarılı", { status: "success", transaction_id: "tx_9981", code: 200 })}
              className="rounded-lg border border-border bg-secondary/50 px-2.5 py-1 text-xs font-semibold text-foreground hover:bg-secondary transition"
            >
              200 OK Başarılı
            </button>
            <button
              type="button"
              onClick={() => handleApplyPreset(400, "400 Bad Request Geçersiz Parametre", { error: "INVALID_PARAM", message: "Missing required signature header" })}
              className="rounded-lg border border-border bg-secondary/50 px-2.5 py-1 text-xs font-semibold text-foreground hover:bg-secondary transition"
            >
              400 Bad Request
            </button>
            <button
              type="button"
              onClick={() => handleApplyPreset(429, "429 Rate Limit Aşıldı", { error: "TOO_MANY_REQUESTS", retry_after_seconds: 60 })}
              className="rounded-lg border border-border bg-secondary/50 px-2.5 py-1 text-xs font-semibold text-foreground hover:bg-secondary transition"
            >
              429 Rate Limit
            </button>
            <button
              type="button"
              onClick={() => handleApplyPreset(503, "503 Servis Bakımda", { error: "SERVICE_UNAVAILABLE", message: "Gateway under maintenance" })}
              className="rounded-lg border border-border bg-secondary/50 px-2.5 py-1 text-xs font-semibold text-foreground hover:bg-secondary transition"
            >
              503 Bakım Modu
            </button>
          </div>

          {/* Preset Helper Notice */}
          <div className="rounded-xl border border-primary/20 bg-primary/5 p-3 text-xs text-muted-foreground flex items-center gap-2">
            <Sparkles className="h-4 w-4 text-primary shrink-0" />
            <span>
              Bu kural kaydedildiğinde, kural kartının altındaki <strong className="text-foreground">"PowerShell Komutunu Kopyala"</strong> veya <strong className="text-foreground">"Bu Kuralı Test Et (Tetikle)"</strong> butonuyla seçtiğiniz şablona uygun istek doğrudan hazır olacaktır.
            </span>
          </div>

          {createError && (
            <div className="flex items-center gap-2 rounded-xl border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive mb-4">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{createError}</span>
            </div>
          )}

          <form onSubmit={handleCreateMock} className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <div>
                <label className="block text-[10px] font-bold uppercase tracking-wider text-muted-foreground mb-1.5">
                  Kural Adı
                </label>
                <input
                  type="text"
                  required
                  value={ruleName}
                  onChange={(e) => setRuleName(e.target.value)}
                  placeholder="Örn: 503 Payment Gateway Outage"
                  className="w-full rounded-xl border border-input bg-background/50 px-3 py-2 text-sm focus:border-primary focus:outline-none"
                />
              </div>

              <div>
                <label className="block text-[10px] font-bold uppercase tracking-wider text-muted-foreground mb-1.5">
                  HTTP Status Code
                </label>
                <select
                  value={statusCode}
                  onChange={(e) => setStatusCode(Number(e.target.value))}
                  className="w-full rounded-xl border border-input bg-background/50 px-3 py-2 text-sm focus:border-primary focus:outline-none"
                >
                  <option value={200}>200 OK</option>
                  <option value={201}>201 Created</option>
                  <option value={400}>400 Bad Request</option>
                  <option value={401}>401 Unauthorized</option>
                  <option value={404}>404 Not Found</option>
                  <option value={429}>429 Too Many Requests</option>
                  <option value={500}>500 Internal Server Error</option>
                  <option value={503}>503 Service Unavailable</option>
                  <option value={504}>504 Gateway Timeout</option>
                </select>
              </div>

              <div>
                <label className="block text-[10px] font-bold uppercase tracking-wider text-muted-foreground mb-1.5">
                  Yapay Gecikme (Delay ms)
                </label>
                <input
                  type="number"
                  min="0"
                  max="10000"
                  step="50"
                  value={delayMs}
                  onChange={(e) => setDelayMs(Number(e.target.value))}
                  placeholder="0 ms"
                  className="w-full rounded-xl border border-input bg-background/50 px-3 py-2 text-sm focus:border-primary focus:outline-none"
                />
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between mb-1.5">
                <label className="block text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                  Dönülecek Mock JSON Gövdesi
                </label>
                <button
                  type="button"
                  onClick={handlePrettify}
                  className="flex items-center gap-1 text-xs text-primary hover:underline font-semibold"
                >
                  <AlignLeft className="h-3 w-3" />
                  <span>Formatla (Prettify)</span>
                </button>
              </div>
              <textarea
                rows={5}
                value={responseBodyText}
                onChange={(e) => setResponseBodyText(e.target.value)}
                className="w-full rounded-xl border border-input bg-background/50 p-3 font-mono text-xs focus:border-primary focus:outline-none leading-relaxed"
              />
            </div>

            <div className="flex justify-end pt-2">
              <button
                type="submit"
                disabled={createMutation.isPending}
                className="flex items-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-xs font-bold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
              >
                {createMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    <span>Kaydediliyor...</span>
                  </>
                ) : (
                  <>
                    <span>Mock Kuralını Başlat</span>
                    <Zap className="h-4 w-4" />
                  </>
                )}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Rules List Section */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-bold tracking-tight">Tanımlı Mock Yanıt Kuralları</h2>
          <span className="text-xs text-muted-foreground">{mockRules.length} kural kayıtlı</span>
        </div>

        <div className="grid grid-cols-1 gap-4">
          {isLoading ? (
            <div className="flex h-48 items-center justify-center">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
            </div>
          ) : mockRules.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-border py-16 text-center bg-card/40">
              <Sparkles className="h-8 w-8 mx-auto mb-2 text-muted-foreground/60" />
              <p className="text-sm font-semibold">Bu endpoint için henüz mock kuralı tanımlanmadı</p>
              <p className="text-xs text-muted-foreground mt-1 max-w-xs mx-auto">
                "Yeni Mock Kuralı" butonuyla sağlayıcınıza dönülecek sahte HTTP durum kodları ve yanıtlar tanımlayın.
              </p>
            </div>
          ) : (
            mockRules.map((rule) => {
              const isEnabled = rule.enabled;
              return (
                <div
                  key={rule.id}
                  className={`rounded-2xl border p-6 shadow-sm flex flex-col justify-between gap-4 transition ${
                    isEnabled
                      ? "border-emerald-500/30 bg-card glow-card"
                      : "border-border bg-card/40 opacity-70"
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <span
                        className={`rounded-lg px-2.5 py-1 text-xs font-mono font-bold ${
                          rule.statusCode < 400
                            ? "bg-emerald-500/15 text-emerald-400 border border-emerald-500/30"
                            : "bg-rose-500/15 text-rose-400 border border-rose-500/30"
                        }`}
                      >
                        HTTP {rule.statusCode}
                      </span>
                      <h3 className="text-sm font-bold text-foreground">{rule.name}</h3>

                      {isEnabled && (
                        <span className="rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 px-2 py-0.5 text-[10px] font-bold flex items-center gap-1">
                          <Zap className="h-2.5 w-2.5" />
                          Aktif Simülasyon Yanıtı
                        </span>
                      )}
                    </div>

                    <div className="flex items-center gap-2.5 text-xs text-muted-foreground">
                      {rule.delayMs > 0 && (
                        <span className="flex items-center gap-1 rounded-lg bg-secondary px-2.5 py-1 font-mono text-[11px]">
                          <Clock className="h-3 w-3 text-amber-400" />
                          {rule.delayMs} ms
                        </span>
                      )}

                      <button
                        type="button"
                        onClick={() => toggleRuleMutation.mutate(rule.id)}
                        disabled={toggleRuleMutation.isPending}
                        className={`flex items-center gap-1.5 rounded-xl border px-2.5 py-1 text-xs font-bold transition shadow-sm cursor-pointer disabled:opacity-50 ${
                          isEnabled
                            ? "border-emerald-500/30 bg-emerald-500/15 text-emerald-400 hover:bg-emerald-500/25"
                            : "border-border bg-secondary/50 text-muted-foreground hover:text-foreground"
                        }`}
                        title={isEnabled ? "Devre dışı bırakmak için tıklayın" : "Aktif etmek için tıklayın"}
                      >
                        <Power className="h-3 w-3" />
                        <span>{isEnabled ? "AKTİF" : "PASİF"}</span>
                      </button>

                      <button
                        type="button"
                        onClick={() => {
                          if (confirm(`"${rule.name}" kuralını silmek istediğinize emin misiniz?`)) {
                            deleteRuleMutation.mutate(rule.id);
                          }
                        }}
                        disabled={deleteRuleMutation.isPending}
                        className="p-1.5 text-muted-foreground hover:text-destructive transition rounded-lg hover:bg-destructive/10"
                        title="Kuralı Sil"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  </div>

                  {/* Mock Response Body */}
                  <div className="space-y-1.5">
                    <span className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                      Dönen Sahte Mock Yanıtı (Response Body):
                    </span>
                    <div className="rounded-xl bg-background p-3.5 font-mono text-xs border border-border overflow-x-auto text-muted-foreground leading-relaxed">
                      <pre>{JSON.stringify(rule.responseBody, null, 2)}</pre>
                    </div>
                  </div>

                  {/* Matching PowerShell Request Section */}
                  {activeEndpoint && (
                    <div className="rounded-xl border border-primary/20 bg-primary/5 p-4 space-y-3">
                      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5">
                        <div className="flex items-center gap-2">
                          <Terminal className="h-4 w-4 text-primary" />
                          <span className="text-xs font-bold text-foreground">
                            Bu Kuralı Tetikleyen İstek (PowerShell / cURL)
                          </span>
                        </div>

                        <div className="flex flex-wrap items-center gap-2">
                          <button
                            type="button"
                            onClick={() => handleCopyPowerShell(rule)}
                            className="flex items-center gap-1.5 rounded-xl border border-border bg-card px-3 py-1.5 text-xs font-semibold text-foreground hover:bg-secondary transition shadow-sm"
                          >
                            {copiedRuleId === rule.id ? (
                              <>
                                <Check className="h-3.5 w-3.5 text-emerald-400" />
                                <span className="text-emerald-400 font-bold">PowerShell Kopyalandı!</span>
                              </>
                            ) : (
                              <>
                                <Copy className="h-3.5 w-3.5 text-muted-foreground" />
                                <span>PowerShell Komutunu Kopyala</span>
                              </>
                            )}
                          </button>

                          <button
                            type="button"
                            onClick={() => handleRunRuleSimulation(rule)}
                            disabled={testingRuleId === rule.id}
                            className="flex items-center gap-1.5 rounded-xl bg-primary hover:bg-primary/90 text-primary-foreground px-3.5 py-1.5 text-xs font-bold transition shadow-sm disabled:opacity-50 cursor-pointer"
                          >
                            {testingRuleId === rule.id ? (
                              <>
                                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                                <span>Tetikleniyor...</span>
                              </>
                            ) : (
                              <>
                                <Send className="h-3.5 w-3.5" />
                                <span>Bu Kuralı Test Et (Tetikle)</span>
                              </>
                            )}
                          </button>
                        </div>
                      </div>

                      {/* PowerShell Command Preview Box */}
                      <div className="rounded-lg bg-background p-2.5 font-mono text-xs border border-border text-foreground overflow-x-auto flex items-center gap-2">
                        <span className="text-primary font-bold select-none">PS&gt;</span>
                        <code className="text-muted-foreground flex-1 truncate select-all">
                          {generatePowerShellCurl(webhookUrl, getRuleMatchingPayload(rule.name, rule.statusCode))}
                        </code>
                      </div>

                      {/* Live Inline Result of Triggering this Rule */}
                      {ruleTestResults[rule.id] && (
                        <div className="rounded-xl border border-border bg-card p-3.5 space-y-2.5 animate-in fade-in duration-200">
                          <div className="flex items-center justify-between text-xs border-b border-border pb-2">
                            <div className="flex items-center gap-2.5">
                              <span
                                className={`rounded-md px-2.5 py-0.5 text-xs font-mono font-bold ${
                                  ruleTestResults[rule.id].status < 400
                                    ? "bg-emerald-500/20 text-emerald-400 border border-emerald-500/30"
                                    : "bg-rose-500/20 text-rose-400 border border-rose-500/30"
                                }`}
                              >
                                HTTP {ruleTestResults[rule.id].status} {ruleTestResults[rule.id].statusText}
                              </span>
                              <span className="flex items-center gap-1 font-mono text-[11px] text-muted-foreground">
                                <Clock className="h-3 w-3 text-primary" />
                                {ruleTestResults[rule.id].latencyMs} ms
                              </span>
                              <span className="rounded-full bg-purple-500/10 text-purple-400 border border-purple-500/20 px-2 py-0.5 text-[10px] font-bold">
                                MOCK MOTORU DEVREDE
                              </span>
                            </div>
                            <span className="text-[10px] text-muted-foreground font-mono">
                              {ruleTestResults[rule.id].timestamp}
                            </span>
                          </div>

                          <div className="space-y-1">
                            <div className="text-[10px] font-bold uppercase text-muted-foreground">Dönen Yanıt:</div>
                            <pre className="rounded-lg bg-secondary/40 p-2.5 font-mono text-xs text-foreground overflow-x-auto">
                              {typeof ruleTestResults[rule.id].body === "object"
                                ? JSON.stringify(ruleTestResults[rule.id].body, null, 2)
                                : ruleTestResults[rule.id].body}
                            </pre>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              );
            })
          )}
        </div>
      </div>

      {/* Live Recent Mock Invocations Feed */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Activity className="h-4 w-4 text-primary" />
            <h2 className="text-base font-bold tracking-tight">Bu Endpoint İçin Yakalanan Son Mock İstekleri</h2>
          </div>

          <Link
            href={`/requests?endpointId=${activeEndpointId}`}
            className="flex items-center gap-1 text-xs text-primary hover:underline"
          >
            <span>Tüm Canlı İstekleri Gör</span>
            <ExternalLink className="h-3 w-3" />
          </Link>
        </div>

        {mockRequests.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-border py-8 text-center bg-card/40">
            <p className="text-xs text-muted-foreground">
              Henüz bu endpoint için yakalanmış mock isteği bulunmuyor. Yukarıdaki "Simülasyon İsteği Gönder" butonuyla test edebilirsiniz.
            </p>
          </div>
        ) : (
          <div className="rounded-2xl border border-border bg-card overflow-hidden shadow-sm">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="bg-secondary/40 text-muted-foreground uppercase text-[10px] tracking-wider border-b border-border">
                  <tr>
                    <th className="p-3">Metot</th>
                    <th className="p-3">Durum Kodu</th>
                    <th className="p-3">Request ID</th>
                    <th className="p-3">Zaman</th>
                    <th className="p-3">Dönen Mock Yanıtı</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {mockRequests.slice(0, 5).map((req) => (
                    <tr key={req.id} className="hover:bg-muted/20">
                      <td className="p-3 font-mono font-bold">{req.httpMethod}</td>
                      <td className="p-3">
                        <span className="rounded-lg px-2 py-0.5 font-mono text-[11px] font-bold bg-purple-500/20 text-purple-400 border border-purple-500/30">
                          HTTP {req.responseStatus || 200} (MOCKED)
                        </span>
                      </td>
                      <td className="p-3 font-mono text-muted-foreground truncate max-w-[160px]">{req.requestId}</td>
                      <td className="p-3 text-muted-foreground font-mono">
                        {new Date(req.createdAt).toLocaleTimeString("tr-TR")}
                      </td>
                      <td className="p-3 font-mono text-muted-foreground truncate max-w-xs">
                        {req.maskedBody || req.rawBody || "-"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

