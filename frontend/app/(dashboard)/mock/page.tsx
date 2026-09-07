"use client";

import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import { Project, Endpoint } from "@apisentinel/shared";
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

        <div className="flex items-center gap-3">
          {endpoints.length > 0 && (
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

      {/* Rules List */}
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
          mockRules.map((rule) => (
            <div
              key={rule.id}
              className="rounded-2xl border border-border bg-card p-6 shadow-sm flex flex-col justify-between gap-4 glow-card"
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
                </div>

                <div className="flex items-center gap-3 text-xs text-muted-foreground">
                  {rule.delayMs > 0 && (
                    <span className="flex items-center gap-1 rounded-lg bg-secondary px-2.5 py-1 font-mono text-[11px]">
                      <Clock className="h-3 w-3 text-amber-400" />
                      {rule.delayMs} ms gecikme
                    </span>
                  )}
                  <span className="rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 px-2.5 py-0.5 text-[11px] font-bold">
                    AKTİF
                  </span>
                </div>
              </div>

              <div className="rounded-xl bg-background p-3.5 font-mono text-xs border border-border overflow-x-auto text-muted-foreground leading-relaxed">
                <pre>{JSON.stringify(rule.responseBody, null, 2)}</pre>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

