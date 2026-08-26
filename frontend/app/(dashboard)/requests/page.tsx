"use client";

import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { useActiveProject } from "../../../contexts/ProjectContext";
import { apiFetch } from "../../../lib/api";
import { Project, CapturedRequest } from "@apisentinel/shared";
import {
  Radio,
  Clock,
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
  Copy,
  Check,
  Terminal,
  ArrowRight,
  ShieldAlert,
  SlidersHorizontal,
} from "lucide-react";

export default function RequestsPage() {
  const { accessToken, organization } = useAuth();
  const { projects, activeProjectId, setActiveProjectId } = useActiveProject();

  const [selectedRequestId, setSelectedRequestId] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<"payload" | "headers" | "query">("payload");
  const [searchFilter, setSearchFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState<"ALL" | "SUCCESS" | "BLOCKED" | "ERROR">("ALL");
  const [copiedPayload, setCopiedPayload] = useState(false);
  const [copiedCurl, setCopiedCurl] = useState(false);

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
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
    refetchInterval: 3000,
  });

  const requests = requestsData?.requests || [];

  // Filter requests
  const filteredRequests = requests.filter((r) => {
    // 1. Search text filter
    if (searchFilter) {
      const query = searchFilter.toLowerCase();
      const matchSearch =
        r.requestId.toLowerCase().includes(query) ||
        r.httpMethod.toLowerCase().includes(query) ||
        (r.endpoint?.name && r.endpoint.name.toLowerCase().includes(query)) ||
        (r.endpoint?.slug && r.endpoint.slug.toLowerCase().includes(query));
      if (!matchSearch) return false;
    }

    // 2. Status code filter
    if (statusFilter === "SUCCESS") {
      return (r.responseStatus ?? 0) >= 200 && (r.responseStatus ?? 0) < 300;
    }
    if (statusFilter === "BLOCKED") {
      return r.responseStatus === 403;
    }
    if (statusFilter === "ERROR") {
      return (r.responseStatus ?? 0) >= 400 && r.responseStatus !== 403;
    }

    return true;
  });

  // Active selected request detail
  const selectedRequest = requests.find((r) => r.id === selectedRequestId) || filteredRequests[0];

  const getFormattedBody = (req?: CapturedRequest) => {
    if (!req) return "";
    const raw = req.maskedBody || req.rawBody || "";
    if (!raw) return "// Boş istek gövdesi";
    try {
      return JSON.stringify(JSON.parse(raw), null, 2);
    } catch {
      return raw;
    }
  };

  const handleCopyPayload = () => {
    if (!selectedRequest) return;
    const body = getFormattedBody(selectedRequest);
    navigator.clipboard.writeText(body);
    setCopiedPayload(true);
    setTimeout(() => setCopiedPayload(false), 2000);
  };

  const handleCopyCurl = () => {
    if (!selectedRequest) return;
    const backendUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3001";
    const slug = selectedRequest.endpoint?.slug || "endpoint";
    const body = selectedRequest.maskedBody || selectedRequest.rawBody || "";

    const curlCmd = `curl -X ${selectedRequest.httpMethod} "${backendUrl}/hook/${slug}" \\
  -H "Content-Type: application/json" \\
  -d '${body.replace(/'/g, "'\\''")}'`;

    navigator.clipboard.writeText(curlCmd);
    setCopiedCurl(true);
    setTimeout(() => setCopiedCurl(false), 2000);
  };

  const getMethodBadgeClass = (method: string) => {
    switch (method.toUpperCase()) {
      case "POST":
        return "bg-blue-500/15 text-blue-400 border border-blue-500/30";
      case "GET":
        return "bg-emerald-500/15 text-emerald-400 border border-emerald-500/30";
      case "PUT":
      case "PATCH":
        return "bg-amber-500/15 text-amber-400 border border-amber-500/30";
      case "DELETE":
        return "bg-rose-500/15 text-rose-400 border border-rose-500/30";
      default:
        return "bg-secondary text-muted-foreground border border-border";
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight">Canlı İstek Akışı (Live Stream)</h1>
            <span className="flex items-center gap-1.5 rounded-full bg-emerald-500/10 px-2.5 py-0.5 text-xs font-semibold text-emerald-400">
              <span className="h-2 w-2 rounded-full bg-emerald-400 animate-pulse" />
              Canlı Dinleniyor (3s Polling)
            </span>
          </div>
          <p className="text-sm text-muted-foreground mt-1">
            Webhook gateway üzerinden geçen tüm istekleri, şifrelenmiş başlıkları ve gövdeleri anlık inceleyin
          </p>
        </div>

        <div className="flex items-center gap-3">
          {projects.length > 1 && (
            <select
              value={activeProjectId}
              onChange={(e) => {
                setActiveProjectId(e.target.value);
                setSelectedRequestId(null);
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

          <button
            onClick={() => refetch()}
            disabled={isRefetching}
            className="flex items-center gap-1.5 rounded-xl border border-border bg-card px-3.5 py-2 text-xs font-semibold text-foreground transition hover:bg-secondary disabled:opacity-50"
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
          {/* Search bar & Status Filter */}
          <div className="space-y-2">
            <div className="relative flex items-center">
              <Search className="absolute left-3 h-4 w-4 text-muted-foreground" />
              <input
                type="text"
                value={searchFilter}
                onChange={(e) => setSearchFilter(e.target.value)}
                placeholder="İstek ID, method veya endpoint ara..."
                className="w-full rounded-xl border border-border bg-card py-2 pl-9 pr-3 text-xs placeholder:text-muted-foreground/60 focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>

            {/* Filter Pills */}
            <div className="flex items-center gap-1.5 overflow-x-auto pb-1 text-xs">
              <button
                onClick={() => setStatusFilter("ALL")}
                className={`rounded-lg px-2.5 py-1 font-semibold transition ${
                  statusFilter === "ALL"
                    ? "bg-primary text-primary-foreground"
                    : "bg-secondary/60 text-muted-foreground hover:text-foreground"
                }`}
              >
                Tümü ({requests.length})
              </button>
              <button
                onClick={() => setStatusFilter("SUCCESS")}
                className={`rounded-lg px-2.5 py-1 font-semibold transition ${
                  statusFilter === "SUCCESS"
                    ? "bg-emerald-500 text-white"
                    : "bg-secondary/60 text-muted-foreground hover:text-emerald-400"
                }`}
              >
                2xx Başarılı
              </button>
              <button
                onClick={() => setStatusFilter("BLOCKED")}
                className={`rounded-lg px-2.5 py-1 font-semibold transition ${
                  statusFilter === "BLOCKED"
                    ? "bg-rose-500 text-white"
                    : "bg-secondary/60 text-muted-foreground hover:text-rose-400"
                }`}
              >
                403 Engellendi
              </button>
              <button
                onClick={() => setStatusFilter("ERROR")}
                className={`rounded-lg px-2.5 py-1 font-semibold transition ${
                  statusFilter === "ERROR"
                    ? "bg-amber-500 text-white"
                    : "bg-secondary/60 text-muted-foreground hover:text-amber-400"
                }`}
              >
                Hatalar
              </button>
            </div>
          </div>

          {isLoading ? (
            <div className="flex h-64 items-center justify-center rounded-2xl border border-border bg-card">
              <Loader2 className="h-6 w-6 animate-spin text-primary" />
            </div>
          ) : filteredRequests.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border py-16 text-center bg-card/50">
              <Radio className="h-8 w-8 text-muted-foreground mb-2" />
              <p className="text-sm font-semibold">Henüz yakalanan istek yok</p>
              <p className="text-xs text-muted-foreground mt-1 max-w-xs">
                Endpoint URL'lerinize webhook gönderildiğinde veya test yapıldığında burada anlık listelenecektir.
              </p>
            </div>
          ) : (
            <div className="space-y-2 max-h-[calc(100vh-270px)] overflow-y-auto pr-1">
              {filteredRequests.map((req) => {
                const isSelected = selectedRequest?.id === req.id;
                const isSuccess = req.responseStatus && req.responseStatus < 400 && req.responseStatus !== 403;
                const isBlocked = req.responseStatus === 403;

                return (
                  <div
                    key={req.id}
                    onClick={() => setSelectedRequestId(req.id)}
                    className={`cursor-pointer rounded-2xl border p-4 transition ${
                      isSelected
                        ? "border-primary bg-primary/5 shadow-sm ring-1 ring-primary/40"
                        : "border-border bg-card hover:border-border/80 hover:bg-secondary/30"
                    }`}
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className={`rounded-lg px-2 py-0.5 text-[10px] font-bold ${getMethodBadgeClass(req.httpMethod)}`}>
                          {req.httpMethod}
                        </span>

                        <span
                          className={`rounded-lg px-2 py-0.5 text-[10px] font-bold ${
                            isBlocked
                              ? "bg-rose-500/20 text-rose-400 border border-rose-500/30"
                              : isSuccess
                              ? "bg-emerald-500/20 text-emerald-400 border border-emerald-500/30"
                              : "bg-amber-500/20 text-amber-400 border border-amber-500/30"
                          }`}
                        >
                          {req.responseStatus || 200}
                        </span>

                        <span className="text-xs font-semibold text-foreground truncate max-w-[130px]">
                          {req.endpoint?.name || "Endpoint"}
                        </span>
                      </div>

                      <span className="text-[10px] text-muted-foreground font-mono">
                        {new Date(req.createdAt).toLocaleTimeString("tr-TR")}
                      </span>
                    </div>

                    <div className="mt-2.5 flex items-center justify-between text-[11px] text-muted-foreground">
                      <span className="font-mono truncate max-w-[180px]">{req.requestId}</span>
                      <span className="font-mono text-[10px]">{req.clientIp || "127.0.0.1"}</span>
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
            <div className="rounded-2xl border border-border bg-card shadow-sm overflow-hidden flex flex-col h-[calc(100vh-210px)]">
              {/* Inspector Header */}
              <div className="border-b border-border p-4 bg-secondary/30 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <span className={`rounded-lg px-2.5 py-1 text-xs font-bold ${getMethodBadgeClass(selectedRequest.httpMethod)}`}>
                    {selectedRequest.httpMethod}
                  </span>
                  <div className="flex flex-col">
                    <span className="font-mono text-xs font-bold text-foreground">
                      {selectedRequest.requestId}
                    </span>
                    <span className="text-[11px] text-muted-foreground">
                      Endpoint: {selectedRequest.endpoint?.name} (/{selectedRequest.endpoint?.slug})
                    </span>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <button
                    onClick={handleCopyCurl}
                    className="flex items-center gap-1 rounded-lg border border-border bg-background px-2.5 py-1.5 text-xs font-semibold text-foreground hover:bg-secondary transition"
                    title="Terminal için cURL komutunu kopyala"
                  >
                    {copiedCurl ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Terminal className="h-3.5 w-3.5 text-primary" />}
                    <span>{copiedCurl ? "cURL Kopyalandı" : "cURL Kopyala"}</span>
                  </button>

                  <div className="flex items-center gap-1.5 text-xs text-muted-foreground font-mono pl-2 border-l border-border">
                    <Clock className="h-3.5 w-3.5" />
                    <span>{new Date(selectedRequest.createdAt).toLocaleTimeString("tr-TR")}</span>
                  </div>
                </div>
              </div>

              {/* Tabs Bar */}
              <div className="flex items-center justify-between border-b border-border bg-secondary/10 px-4">
                <div className="flex">
                  <button
                    onClick={() => setActiveTab("payload")}
                    className={`flex items-center gap-2 border-b-2 px-4 py-3 text-xs font-semibold transition ${
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
                    className={`flex items-center gap-2 border-b-2 px-4 py-3 text-xs font-semibold transition ${
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
                    className={`flex items-center gap-2 border-b-2 px-4 py-3 text-xs font-semibold transition ${
                      activeTab === "query"
                        ? "border-primary text-primary"
                        : "border-transparent text-muted-foreground hover:text-foreground"
                    }`}
                  >
                    <Layers className="h-3.5 w-3.5" />
                    <span>Query Params ({Object.keys(selectedRequest.queryParams || {}).length})</span>
                  </button>
                </div>

                {activeTab === "payload" && (
                  <button
                    onClick={handleCopyPayload}
                    className="flex items-center gap-1 rounded-lg bg-secondary px-2.5 py-1 text-xs font-medium text-foreground hover:bg-primary hover:text-primary-foreground transition"
                  >
                    {copiedPayload ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
                    <span>{copiedPayload ? "Kopyalandı" : "JSON Kopyala"}</span>
                  </button>
                )}
              </div>

              {/* Inspector Content */}
              <div className="flex-1 overflow-y-auto p-4 font-mono text-xs">
                {activeTab === "payload" && (
                  <pre className="rounded-xl bg-background p-4 text-foreground/90 overflow-x-auto leading-relaxed border border-border">
                    {getFormattedBody(selectedRequest)}
                  </pre>
                )}

                {activeTab === "headers" && (
                  <div className="space-y-2">
                    {Object.entries(selectedRequest.headers || {}).map(([key, value]) => (
                      <div
                        key={key}
                        className="flex items-start gap-2 rounded-xl bg-background p-3 border border-border"
                      >
                        <span className="font-semibold text-primary min-w-[180px] truncate">{key}:</span>
                        <span className="text-foreground break-all">{String(value)}</span>
                      </div>
                    ))}
                  </div>
                )}

                {activeTab === "query" && (
                  <div className="space-y-2">
                    {Object.keys(selectedRequest.queryParams || {}).length === 0 ? (
                      <p className="text-muted-foreground italic text-center py-12">
                        Query parametresi bulunmuyor
                      </p>
                    ) : (
                      Object.entries(selectedRequest.queryParams || {}).map(([key, value]) => (
                        <div
                          key={key}
                          className="flex items-start gap-2 rounded-xl bg-background p-3 border border-border"
                        >
                          <span className="font-semibold text-primary min-w-[140px] truncate">{key}:</span>
                          <span className="text-foreground break-all">{String(value)}</span>
                        </div>
                      ))
                    )}
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="flex h-full items-center justify-center rounded-2xl border border-dashed border-border bg-card/40 p-12 text-center text-muted-foreground">
              Detaylarını incelemek için soldaki listeden bir istek seçin
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

