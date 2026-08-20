"use client";

import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import { Project, Endpoint, CapturedRequest } from "@apisentinel/shared";
import {
  Radio,
  Clock,
  Globe,
  Filter,
  RefreshCw,
  Search,
  Code,
  FileJson,
  Layers,
  Shield,
  Loader2,
  CheckCircle2,
  XCircle,
  ChevronRight,
  ExternalLink,
} from "lucide-react";

export default function RequestsPage() {
  const { accessToken, organization } = useAuth();

  const [selectedProjectId, setSelectedProjectId] = useState<string>("");
  const [selectedRequestId, setSelectedRequestId] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<"payload" | "headers" | "query">("payload");
  const [searchFilter, setSearchFilter] = useState("");

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

  // Fetch requests for active project
  const { data: requestsData, isLoading, refetch, isRefetching } = useQuery({
    queryKey: ["requests", activeProjectId],
    queryFn: () =>
      apiFetch<{ requests: (CapturedRequest & { endpoint?: { name: string; slug: string } })[] }>(
        `/api/projects/${activeProjectId}/requests`,
        {
          token: accessToken,
          organizationId: organization?.id,
        }
      ),
    enabled: !!accessToken && !!activeProjectId,
    refetchInterval: 3000, // Polling fallback until SSE in Phase 3
  });

  const requests = requestsData?.requests || [];

  // Filter requests
  const filteredRequests = requests.filter((r) => {
    if (!searchFilter) return true;
    const query = searchFilter.toLowerCase();
    return (
      r.requestId.toLowerCase().includes(query) ||
      r.httpMethod.toLowerCase().includes(query) ||
      r.endpoint?.name.toLowerCase().includes(query) ||
      r.endpoint?.slug.toLowerCase().includes(query)
    );
  });

  // Active selected request detail
  const selectedRequest = requests.find((r) => r.id === selectedRequestId) || filteredRequests[0];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Canlı İstekler (Live Stream)</h1>
          <p className="text-sm text-muted-foreground">
            Webhook gateway üzerinden geçen istekleri inceleyin ve analiz edin
          </p>
        </div>

        <div className="flex items-center gap-3">
          {projects.length > 1 && (
            <select
              value={activeProjectId}
              onChange={(e) => {
                setSelectedProjectId(e.target.value);
                setSelectedRequestId(null);
              }}
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
            onClick={() => refetch()}
            disabled={isRefetching}
            className="flex items-center gap-1.5 rounded-lg border border-border bg-card px-3 py-2 text-xs font-semibold text-foreground transition hover:bg-secondary disabled:opacity-50"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${isRefetching ? "animate-spin text-primary" : ""}`} />
            <span>Yenile</span>
          </button>
        </div>
      </div>

      {/* Main 2-column layout: Left (List) & Right (Inspector) */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
        {/* Left Column: Request List (5 cols) */}
        <div className="space-y-3 lg:col-span-5">
          {/* Search bar */}
          <div className="relative flex items-center">
            <Search className="absolute left-3 h-4 w-4 text-muted-foreground" />
            <input
              type="text"
              value={searchFilter}
              onChange={(e) => setSearchFilter(e.target.value)}
              placeholder="İstek ID, method veya endpoint ara..."
              className="w-full rounded-lg border border-input bg-card py-2 pl-9 pr-3 text-xs placeholder:text-muted-foreground/60 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>

          {isLoading ? (
            <div className="flex h-64 items-center justify-center rounded-xl border border-border bg-card">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
            </div>
          ) : filteredRequests.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-16 text-center bg-card">
              <Radio className="h-6 w-6 text-muted-foreground mb-2" />
              <p className="text-sm font-medium">Henüz yakalanan istek yok</p>
              <p className="text-xs text-muted-foreground mt-1">
                Endpoint URL'lerinize webhook gönderildiğinde burada canlı listelenecektir.
              </p>
            </div>
          ) : (
            <div className="space-y-2 max-h-[calc(100vh-280px)] overflow-y-auto pr-1">
              {filteredRequests.map((req) => {
                const isSelected = selectedRequest?.id === req.id;
                const isSuccess = req.responseStatus && req.responseStatus < 400;

                return (
                  <div
                    key={req.id}
                    onClick={() => setSelectedRequestId(req.id)}
                    className={`cursor-pointer rounded-xl border p-4 transition ${
                      isSelected
                        ? "border-primary bg-primary/5 shadow-sm"
                        : "border-border bg-card hover:border-border/80 hover:bg-secondary/40"
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span
                          className={`rounded px-1.5 py-0.5 text-[10px] font-bold ${
                            req.httpMethod === "POST"
                              ? "bg-blue-500/15 text-blue-400"
                              : req.httpMethod === "GET"
                              ? "bg-emerald-500/15 text-emerald-400"
                              : "bg-amber-500/15 text-amber-400"
                          }`}
                        >
                          {req.httpMethod}
                        </span>

                        <span
                          className={`rounded px-1.5 py-0.5 text-[10px] font-semibold ${
                            isSuccess ? "bg-emerald-500/15 text-emerald-400" : "bg-rose-500/15 text-rose-400"
                          }`}
                        >
                          {req.responseStatus || 200}
                        </span>

                        <span className="text-xs font-semibold text-foreground truncate max-w-[140px]">
                          {req.endpoint?.name || "Endpoint"}
                        </span>
                      </div>

                      <span className="text-[10px] text-muted-foreground font-mono">
                        {new Date(req.createdAt).toLocaleTimeString("tr-TR")}
                      </span>
                    </div>

                    <div className="mt-2 flex items-center justify-between text-[11px] text-muted-foreground">
                      <span className="font-mono truncate max-w-[180px]">{req.requestId}</span>
                      <span>{req.clientIp || "127.0.0.1"}</span>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Right Column: Request Inspector (7 cols) */}
        <div className="lg:col-span-7">
          {selectedRequest ? (
            <div className="rounded-xl border border-border bg-card shadow-sm overflow-hidden flex flex-col h-[calc(100vh-230px)]">
              {/* Inspector Header */}
              <div className="border-b border-border p-4 bg-secondary/20 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <span
                    className={`rounded px-2 py-0.5 text-xs font-bold ${
                      selectedRequest.httpMethod === "POST"
                        ? "bg-blue-500/20 text-blue-400"
                        : "bg-emerald-500/20 text-emerald-400"
                    }`}
                  >
                    {selectedRequest.httpMethod}
                  </span>
                  <span className="font-mono text-xs font-semibold text-foreground">
                    {selectedRequest.requestId}
                  </span>
                </div>

                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <Clock className="h-3.5 w-3.5" />
                  <span>{new Date(selectedRequest.createdAt).toLocaleString("tr-TR")}</span>
                </div>
              </div>

              {/* Tabs Bar */}
              <div className="flex border-b border-border bg-secondary/10 px-4">
                <button
                  onClick={() => setActiveTab("payload")}
                  className={`flex items-center gap-2 border-b-2 px-3 py-2.5 text-xs font-semibold transition ${
                    activeTab === "payload"
                      ? "border-primary text-primary"
                      : "border-transparent text-muted-foreground hover:text-foreground"
                  }`}
                >
                  <FileJson className="h-3.5 w-3.5" />
                  <span>Payload Body</span>
                </button>

                <button
                  onClick={() => setActiveTab("headers")}
                  className={`flex items-center gap-2 border-b-2 px-3 py-2.5 text-xs font-semibold transition ${
                    activeTab === "headers"
                      ? "border-primary text-primary"
                      : "border-transparent text-muted-foreground hover:text-foreground"
                  }`}
                >
                  <Code className="h-3.5 w-3.5" />
                  <span>Headers ({Object.keys(selectedRequest.headers || {}).length})</span>
                </button>

                <button
                  onClick={() => setActiveTab("query")}
                  className={`flex items-center gap-2 border-b-2 px-3 py-2.5 text-xs font-semibold transition ${
                    activeTab === "query"
                      ? "border-primary text-primary"
                      : "border-transparent text-muted-foreground hover:text-foreground"
                  }`}
                >
                  <Layers className="h-3.5 w-3.5" />
                  <span>Query Params ({Object.keys(selectedRequest.queryParams || {}).length})</span>
                </button>
              </div>

              {/* Inspector Content */}
              <div className="flex-1 overflow-y-auto p-4 font-mono text-xs">
                {activeTab === "payload" && (
                  <pre className="rounded-lg bg-background p-4 text-foreground/90 overflow-x-auto leading-relaxed border border-border">
                    {selectedRequest.rawBody
                      ? (() => {
                          try {
                            return JSON.stringify(JSON.parse(selectedRequest.rawBody), null, 2);
                          } catch {
                            return selectedRequest.rawBody;
                          }
                        })()
                      : "// Boş istek gövdesi"}
                  </pre>
                )}

                {activeTab === "headers" && (
                  <div className="space-y-1.5">
                    {Object.entries(selectedRequest.headers || {}).map(([key, value]) => (
                      <div
                        key={key}
                        className="flex items-start gap-2 rounded bg-background p-2 border border-border"
                      >
                        <span className="font-semibold text-primary min-w-[160px] truncate">{key}:</span>
                        <span className="text-foreground break-all">{String(value)}</span>
                      </div>
                    ))}
                  </div>
                )}

                {activeTab === "query" && (
                  <div className="space-y-1.5">
                    {Object.keys(selectedRequest.queryParams || {}).length === 0 ? (
                      <p className="text-muted-foreground italic text-center py-8">
                        Query parametresi bulunmuyor
                      </p>
                    ) : (
                      Object.entries(selectedRequest.queryParams || {}).map(([key, value]) => (
                        <div
                          key={key}
                          className="flex items-start gap-2 rounded bg-background p-2 border border-border"
                        >
                          <span className="font-semibold text-primary min-w-[120px] truncate">{key}:</span>
                          <span className="text-foreground break-all">{String(value)}</span>
                        </div>
                      ))
                    )}
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="flex h-full items-center justify-center rounded-xl border border-dashed border-border bg-card p-12 text-center text-muted-foreground">
              Detaylarını incelemek için soldaki listeden bir istek seçin
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
