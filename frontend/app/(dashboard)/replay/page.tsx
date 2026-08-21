"use client";

import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import { Project, CapturedRequest } from "@apisentinel/shared";
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
} from "lucide-react";

interface ReplayJobItem {
  id: string;
  sourceRequestId: string;
  requestId: string;
  httpMethod: string;
  endpointName: string;
  targetUrl: string;
  status: string;
  responseStatus: number;
  responseBody: string;
  createdAt: string;
}

export default function ReplayPage() {
  const queryClient = useQueryClient();
  const { accessToken, organization } = useAuth();

  const [selectedProjectId, setSelectedProjectId] = useState<string>("");
  const [isNewReplayOpen, setIsNewReplayOpen] = useState(false);
  const [selectedRequestId, setSelectedRequestId] = useState<string>("");
  const [targetUrl, setTargetUrl] = useState<string>("https://httpbin.org/post");
  const [replayError, setReplayError] = useState<string | null>(null);
  const [lastReplayResult, setLastReplayResult] = useState<any | null>(null);

  // Fetch projects
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

  // Fetch captured requests for replay selection
  const { data: requestsData } = useQuery({
    queryKey: ["requests", activeProjectId],
    queryFn: () =>
      apiFetch<{ requests: CapturedRequest[] }>(`/api/projects/${activeProjectId}/requests`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeProjectId,
  });

  // Fetch past replay jobs
  const { data: replaysData, isLoading } = useQuery({
    queryKey: ["replays", activeProjectId],
    queryFn: () =>
      apiFetch<{ replays: ReplayJobItem[] }>(`/api/projects/${activeProjectId}/replays`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeProjectId,
    refetchInterval: 5000,
  });

  // Replay mutation
  const replayMutation = useMutation({
    mutationFn: (vars: { requestId: string; targetUrl: string }) =>
      apiFetch<any>(`/api/requests/${vars.requestId}/replay`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify({ targetUrl: vars.targetUrl }),
      }),
    onSuccess: (data) => {
      setLastReplayResult(data);
      queryClient.invalidateQueries({ queryKey: ["replays", activeProjectId] });
      setReplayError(null);
    },
    onError: (err: any) => {
      setReplayError(err.message || "Replay işlemi başarısız oldu.");
    },
  });

  const handleExecuteReplay = (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedRequestId || !targetUrl.trim()) return;
    setLastReplayResult(null);
    replayMutation.mutate({
      requestId: selectedRequestId,
      targetUrl: targetUrl.trim(),
    });
  };

  const requests = requestsData?.requests || [];
  const replays = replaysData?.replays || [];

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-bold tracking-tight">Replay Lab</h1>
            <span className="flex items-center gap-1 rounded-full bg-emerald-500/10 px-2.5 py-0.5 text-xs font-semibold text-emerald-400">
              <ShieldCheck className="h-3 w-3" />
              SSRF Guard Active
            </span>
          </div>
          <p className="text-sm text-muted-foreground">
            Yakalanan webhook ve API isteklerini hedef sunuculara veya yerel ortama güvenle tekrar fırlatın
          </p>
        </div>

        <div className="flex items-center gap-3">
          {projects.length > 1 && (
            <select
              value={activeProjectId}
              onChange={(e) => setSelectedProjectId(e.target.value)}
              className="rounded-lg border border-input bg-card px-3 py-2 text-sm font-medium focus:border-primary focus:outline-none"
            >
              {projects.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          )}

          <button
            onClick={() => {
              setIsNewReplayOpen(!isNewReplayOpen);
              if (!selectedRequestId && requests[0]) {
                setSelectedRequestId(requests[0].id);
              }
            }}
            disabled={!activeProjectId || requests.length === 0}
            className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
          >
            <Play className="h-4 w-4" />
            <span>Yeni Replay Ateşle</span>
          </button>
        </div>
      </div>

      {/* New Replay Drawer / Form */}
      {isNewReplayOpen && (
        <div className="rounded-xl border border-border bg-card p-6 shadow-sm animate-in fade-in duration-200">
          <div className="flex items-center justify-between border-b border-border pb-4 mb-4">
            <div className="flex items-center gap-2">
              <Repeat className="h-5 w-5 text-primary" />
              <h3 className="text-base font-semibold">Hedefe İstek Tekrarı (Replay) Başlat</h3>
            </div>
            <button
              onClick={() => {
                setIsNewReplayOpen(false);
                setLastReplayResult(null);
                setReplayError(null);
              }}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              Kapat
            </button>
          </div>

          {replayError && (
            <div className="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive mb-4">
              <ShieldAlert className="h-4 w-4 shrink-0" />
              <span>{replayError}</span>
            </div>
          )}

          <form onSubmit={handleExecuteReplay} className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                  Tekrar Edilecek İstek (Source Request)
                </label>
                <select
                  value={selectedRequestId}
                  onChange={(e) => setSelectedRequestId(e.target.value)}
                  className="w-full rounded-lg border border-input bg-background/50 px-3 py-2 text-sm focus:border-primary focus:outline-none"
                >
                  {requests.map((r) => (
                    <option key={r.id} value={r.id}>
                      [{r.httpMethod}] {r.requestId} ({new Date(r.createdAt).toLocaleTimeString("tr-TR")})
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                  Hedef URL (Target Destination)
                </label>
                <input
                  type="url"
                  required
                  value={targetUrl}
                  onChange={(e) => setTargetUrl(e.target.value)}
                  placeholder="https://httpbin.org/post veya https://api.mycompany.com/webhook"
                  className="w-full rounded-lg border border-input bg-background/50 px-3 py-2 text-sm focus:border-primary focus:outline-none"
                />
              </div>
            </div>

            <div className="flex items-center justify-between pt-2">
              <p className="text-xs text-muted-foreground">
                🛡️ SSRF Guard: Özel IP'ler (`127.0.0.1`, `10.0.0.0/8`) ve AWS metadata (`169.254.169.254`) otomatik engellenir.
              </p>

              <button
                type="submit"
                disabled={replayMutation.isPending}
                className="flex items-center gap-2 rounded-lg bg-primary px-5 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
              >
                {replayMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    <span>Gönderiliyor...</span>
                  </>
                ) : (
                  <>
                    <span>Replay Gönder</span>
                    <Zap className="h-4 w-4" />
                  </>
                )}
              </button>
            </div>
          </form>

          {/* Last Replay Result Box */}
          {lastReplayResult && (
            <div className="mt-6 rounded-lg border border-emerald-500/30 bg-emerald-500/5 p-4 animate-in fade-in duration-200">
              <div className="flex items-center justify-between border-b border-emerald-500/20 pb-2 mb-3">
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-emerald-400" />
                  <span className="text-sm font-bold text-foreground">Replay Başarılı!</span>
                  <span className="rounded bg-emerald-500/20 px-2 py-0.5 text-xs font-mono font-bold text-emerald-400">
                    HTTP {lastReplayResult.responseStatus}
                  </span>
                </div>
                <span className="text-xs text-muted-foreground font-mono">
                  Gecikme: {lastReplayResult.latencyMs} ms
                </span>
              </div>

              <div className="space-y-1">
                <span className="text-xs text-muted-foreground font-semibold">Hedef Yanıtı:</span>
                <pre className="max-h-48 overflow-y-auto rounded bg-background p-3 text-xs font-mono text-foreground border border-border">
                  {lastReplayResult.responseBody || "// Boş yanıt"}
                </pre>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Past Replays Table */}
      <div className="rounded-xl border border-border bg-card shadow-sm overflow-hidden">
        <div className="border-b border-border p-4 bg-secondary/20 flex items-center justify-between">
          <h3 className="text-sm font-bold tracking-tight">Geçmiş Replay Görevleri ({replays.length})</h3>
        </div>

        {isLoading ? (
          <div className="flex h-48 items-center justify-center">
            <Loader2 className="h-6 w-6 animate-spin text-primary" />
          </div>
        ) : replays.length === 0 ? (
          <div className="py-16 text-center text-muted-foreground">
            <Repeat className="h-8 w-8 mx-auto mb-2 text-muted-foreground/60" />
            <p className="text-sm font-medium">Henüz replay görevi çalıştırılmadı</p>
            <p className="text-xs text-muted-foreground mt-1">
              Yukarıdaki "Yeni Replay Ateşle" butonu ile yakalanan istekleri dilediğiniz hedefe yönlendirebilirsiniz.
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="border-b border-border bg-secondary/30 text-muted-foreground font-semibold">
                <tr>
                  <th className="p-3.5">Method</th>
                  <th className="p-3.5">Orijinal İstek</th>
                  <th className="p-3.5">Endpoint</th>
                  <th className="p-3.5">Hedef URL</th>
                  <th className="p-3.5">Durum</th>
                  <th className="p-3.5">Yanıt Kodu</th>
                  <th className="p-3.5">Zaman</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border font-mono">
                {replays.map((j) => (
                  <tr key={j.id} className="hover:bg-secondary/20 transition">
                    <td className="p-3.5">
                      <span className="rounded bg-blue-500/15 text-blue-400 px-1.5 py-0.5 font-bold">
                        {j.httpMethod}
                      </span>
                    </td>
                    <td className="p-3.5 font-semibold text-foreground">{j.requestId}</td>
                    <td className="p-3.5 text-muted-foreground font-sans">{j.endpointName}</td>
                    <td className="p-3.5 text-foreground truncate max-w-xs">{j.targetUrl}</td>
                    <td className="p-3.5">
                      <span
                        className={`rounded px-2 py-0.5 text-[11px] font-semibold font-sans ${
                          j.status === "COMPLETED"
                            ? "bg-emerald-500/15 text-emerald-400"
                            : "bg-rose-500/15 text-rose-400"
                        }`}
                      >
                        {j.status}
                      </span>
                    </td>
                    <td className="p-3.5">
                      <span
                        className={`font-bold ${
                          j.responseStatus >= 200 && j.responseStatus < 400
                            ? "text-emerald-400"
                            : "text-rose-400"
                        }`}
                      >
                        {j.responseStatus || "-"}
                      </span>
                    </td>
                    <td className="p-3.5 text-muted-foreground">
                      {new Date(j.createdAt).toLocaleTimeString("tr-TR")}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
