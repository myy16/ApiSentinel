"use client";

import React, { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { useActiveProject } from "../../../contexts/ProjectContext";
import { apiFetch } from "../../../lib/api";
import { Project } from "@apisentinel/shared";
import {
  ShieldAlert,
  ShieldCheck,
  Bot,
  Sparkles,
  Loader2,
  AlertOctagon,
  AlertTriangle,
  Info,
  Shield,
  RotateCw,
  Copy,
  Check,
  CheckCircle2,
  SlidersHorizontal,
  ChevronDown,
  ChevronUp,
  X,
  GitBranch,
  FileCode,
  Globe,
  Terminal,
} from "lucide-react";

interface Finding {
  id: string;
  requestId?: string;
  reqDisplayId?: string;
  endpointName?: string;
  sourceType?: string;
  repository?: string;
  filePath?: string;
  lineNumber?: number;
  commitHash?: string;
  category: string;
  type: string;
  severity: "CRITICAL" | "HIGH" | "MEDIUM" | "INFO";
  action: "BLOCK" | "MASK" | "ALLOW";
  fieldPath?: string;
  message: string;
  evidenceMasked?: string;
  confidence: number;
  createdAt: string;
}

interface FindingStats {
  criticalCount: number;
  highCount: number;
  mediumCount: number;
  infoCount: number;
  totalCount: number;
}

interface AIExplanation {
  findingType: string;
  severity: string;
  title: string;
  rootCause: string;
  impact: string;
  remediationSteps: string[];
  codeSnippet: string;
  confidenceScore: number;
  provider?: string;
}

export default function SecurityFindingsPage() {
  const { accessToken, organization } = useAuth();
  const { projects, activeProjectId, setActiveProjectId } = useActiveProject();

  const [activeFinding, setActiveFinding] = useState<Finding | null>(null);
  const [severityFilter, setSeverityFilter] = useState<"ALL" | "CRITICAL" | "HIGH" | "MEDIUM">("ALL");
  const [sourceFilter, setSourceFilter] = useState<"ALL" | "WEBHOOK" | "AGENT_GIT">("ALL");
  const [copiedCode, setCopiedCode] = useState(false);

  // 1. Fetch Real Security Findings from DB
  const {
    data: findingsData,
    isLoading: isFindingsLoading,
    refetch: refetchFindings,
    isRefetching: isRefetchingFindings,
  } = useQuery({
    queryKey: ["findings", activeProjectId],
    queryFn: () =>
      apiFetch<{ findings: Finding[] }>(`/api/projects/${activeProjectId}/findings`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
  });

  // 2. Fetch Aggregated Statistics from DB
  const {
    data: statsData,
    isLoading: isStatsLoading,
    refetch: refetchStats,
  } = useQuery({
    queryKey: ["findings", "stats", activeProjectId],
    queryFn: () =>
      apiFetch<FindingStats>(`/api/projects/${activeProjectId}/findings/stats`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
  });

  const findings = findingsData?.findings || [];
  const stats = statsData || {
    criticalCount: 0,
    highCount: 0,
    mediumCount: 0,
    infoCount: 0,
    totalCount: 0,
  };

  // 3. AI Explain Mutation
  const explainMutation = useMutation({
    mutationFn: (finding: Finding) =>
      apiFetch<AIExplanation>("/api/ai/explain", {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify({
          category: finding.category || "SECRET",
          findingType: finding.type,
          severity: finding.severity,
          maskedEvidence: finding.evidenceMasked || "********",
          message: finding.message,
        }),
      }),
  });

  const handleToggleExplain = (finding: Finding) => {
    if (activeFinding?.id === finding.id) {
      // Toggle off if already open
      setActiveFinding(null);
    } else {
      setActiveFinding(finding);
      explainMutation.mutate(finding);
    }
  };

  const handleRefresh = () => {
    refetchFindings();
    refetchStats();
  };

  const handleCopyCode = (snippet: string) => {
    navigator.clipboard.writeText(snippet);
    setCopiedCode(true);
    setTimeout(() => setCopiedCode(false), 2000);
  };

  // Filter findings based on severity tab and source filter
  const filteredFindings = findings.filter((f) => {
    if (sourceFilter !== "ALL") {
      const src = f.sourceType || "WEBHOOK";
      if (src !== sourceFilter) return false;
    }
    if (severityFilter === "ALL") return true;
    if (severityFilter === "CRITICAL") return f.severity === "CRITICAL";
    if (severityFilter === "HIGH") return f.severity === "HIGH";
    if (severityFilter === "MEDIUM") return f.severity === "MEDIUM" || f.severity === "INFO";
    return true;
  });

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Güvenlik Bulguları & Tehdit Analizi</h1>
          <p className="text-sm text-muted-foreground">
            Yakalanan API trafiğinde tespit edilen PII, Secret, Injection ve anomali tehditleri
          </p>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={handleRefresh}
            disabled={isRefetchingFindings}
            className="flex items-center gap-2 rounded-xl border border-border bg-card px-3.5 py-2 text-xs font-semibold hover:bg-secondary transition disabled:opacity-50"
          >
            <RotateCw className={`h-3.5 w-3.5 ${isRefetchingFindings ? "animate-spin" : ""}`} />
            <span>Yenile</span>
          </button>

          {projects && projects.length > 0 && (
            <select
              value={activeProjectId || ""}
              onChange={(e) => setActiveProjectId(e.target.value)}
              className="rounded-xl border border-border bg-card px-3.5 py-2 text-xs font-semibold focus:outline-none focus:ring-2 focus:ring-primary"
            >
              {projects.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          )}
        </div>
      </div>

      {/* 4 Analytics Metrics Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-2xl border border-rose-500/20 bg-rose-500/5 p-5">
          <div className="flex items-center justify-between text-xs font-semibold text-rose-400">
            <span>Kritik Tehditler</span>
            <AlertOctagon className="h-4 w-4" />
          </div>
          <p className="text-3xl font-bold text-rose-400 mt-2">{stats.criticalCount}</p>
          <p className="text-[11px] text-muted-foreground mt-1">AWS Key & SQLi (Engellendi)</p>
        </div>

        <div className="rounded-2xl border border-amber-500/20 bg-amber-500/5 p-5">
          <div className="flex items-center justify-between text-xs font-semibold text-amber-400">
            <span>Yüksek Risk (PII/TCKN)</span>
            <AlertTriangle className="h-4 w-4" />
          </div>
          <p className="text-3xl font-bold text-amber-400 mt-2">{stats.highCount}</p>
          <p className="text-[11px] text-muted-foreground mt-1">TCKN, Kredi Kartı & XSS</p>
        </div>

        <div className="rounded-2xl border border-blue-500/20 bg-blue-500/5 p-5">
          <div className="flex items-center justify-between text-xs font-semibold text-blue-400">
            <span>Orta / Bilgi</span>
            <Info className="h-4 w-4" />
          </div>
          <p className="text-3xl font-bold text-blue-400 mt-2">{stats.mediumCount + stats.infoCount}</p>
          <p className="text-[11px] text-muted-foreground mt-1">E-posta Maskeleme & JWT</p>
        </div>

        <div className="rounded-2xl border border-border bg-card p-5">
          <div className="flex items-center justify-between text-xs font-semibold text-foreground">
            <span>Toplam Tehdit</span>
            <Shield className="h-4 w-4 text-primary" />
          </div>
          <p className="text-3xl font-bold text-foreground mt-2">{stats.totalCount}</p>
          <p className="text-[11px] text-muted-foreground mt-1">Veritabanına kalıcı kaydedilen</p>
        </div>
      </div>

      {/* Filter Tabs Bar (Severity + Source) */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <SlidersHorizontal className="h-4 w-4 text-muted-foreground" />
          <div className="flex flex-wrap items-center gap-1.5 text-xs">
            <button
              onClick={() => setSeverityFilter("ALL")}
              className={`rounded-lg px-3 py-1.5 font-semibold transition ${
                severityFilter === "ALL"
                  ? "bg-primary text-primary-foreground"
                  : "bg-secondary text-muted-foreground hover:text-foreground"
              }`}
            >
              Tüm Bulgular ({findings.length})
            </button>
            <button
              onClick={() => setSeverityFilter("CRITICAL")}
              className={`rounded-lg px-3 py-1.5 font-semibold transition ${
                severityFilter === "CRITICAL"
                  ? "bg-rose-500 text-white"
                  : "bg-secondary text-muted-foreground hover:text-rose-400"
              }`}
            >
              Kritik ({stats.criticalCount})
            </button>
            <button
              onClick={() => setSeverityFilter("HIGH")}
              className={`rounded-lg px-3 py-1.5 font-semibold transition ${
                severityFilter === "HIGH"
                  ? "bg-amber-500 text-white"
                  : "bg-secondary text-muted-foreground hover:text-amber-400"
              }`}
            >
              Yüksek ({stats.highCount})
            </button>
            <button
              onClick={() => setSeverityFilter("MEDIUM")}
              className={`rounded-lg px-3 py-1.5 font-semibold transition ${
                severityFilter === "MEDIUM"
                  ? "bg-blue-500 text-white"
                  : "bg-secondary text-muted-foreground hover:text-blue-400"
              }`}
            >
              Orta & Bilgi ({stats.mediumCount + stats.infoCount})
            </button>
          </div>
        </div>

        {/* Source Filter (All vs Webhook vs Git/CLI Agent) */}
        <div className="flex items-center gap-1.5 bg-secondary/50 p-1 rounded-xl border border-border text-xs">
          <button
            onClick={() => setSourceFilter("ALL")}
            className={`px-3 py-1 rounded-lg font-semibold transition ${
              sourceFilter === "ALL"
                ? "bg-card text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            Tüm Kaynaklar
          </button>
          <button
            onClick={() => setSourceFilter("WEBHOOK")}
            className={`flex items-center gap-1.5 px-3 py-1 rounded-lg font-semibold transition ${
              sourceFilter === "WEBHOOK"
                ? "bg-card text-emerald-400 shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            <Globe className="h-3 w-3" />
            <span>Webhook</span>
          </button>
          <button
            onClick={() => setSourceFilter("AGENT_GIT")}
            className={`flex items-center gap-1.5 px-3 py-1 rounded-lg font-semibold transition ${
              sourceFilter === "AGENT_GIT"
                ? "bg-card text-purple-400 shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            <Terminal className="h-3 w-3" />
            <span>Git / CLI Ajanı</span>
          </button>
        </div>
      </div>

      {/* Findings List with In-Place (Inline) AI Remediation Drawer */}
      <div className="space-y-4">
        {isFindingsLoading ? (
          <div className="flex flex-col items-center justify-center rounded-2xl border border-border bg-card p-12 text-center h-[300px]">
            <Loader2 className="h-8 w-8 animate-spin text-primary mb-3" />
            <p className="text-sm font-semibold">Veritabanından Güvenlik Bulguları Çekiliyor...</p>
          </div>
        ) : filteredFindings.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border bg-card/50 p-12 text-center">
            <ShieldCheck className="h-12 w-12 text-emerald-400 mb-3" />
            <p className="text-base font-bold text-foreground">Tertemiz! Güvenlik İhlali Yok</p>
            <p className="text-xs text-muted-foreground mt-1 max-w-sm">
              Seçili filtre veya projede engellenen bir tehdit bulunmuyor. Webhook gönderildiğinde veya git taraması yapıldığında bulgular anlık burada listelenecektir.
            </p>
          </div>
        ) : (
          filteredFindings.map((f) => {
            const isExpanded = activeFinding?.id === f.id;
            const isGitFinding = f.sourceType === "AGENT_GIT" || !!f.filePath || !!f.repository;

            return (
              <div
                key={f.id}
                className={`rounded-2xl border p-5 shadow-sm transition flex flex-col gap-4 ${
                  isExpanded
                    ? "border-purple-500/50 bg-card shadow-md ring-1 ring-purple-500/30"
                    : "border-border bg-card hover:border-border/80 hover:bg-secondary/20"
                }`}
              >
                {/* Header Row */}
                <div className="flex items-center justify-between">
                  <div className="flex flex-wrap items-center gap-2.5">
                    <span
                      className={`rounded-lg px-2.5 py-1 text-xs font-bold ${
                        f.severity === "CRITICAL"
                          ? "bg-rose-500/20 text-rose-400 border border-rose-500/30"
                          : f.severity === "HIGH"
                          ? "bg-amber-500/20 text-amber-400 border border-amber-500/30"
                          : "bg-blue-500/20 text-blue-400 border border-blue-500/30"
                      }`}
                    >
                      {f.severity}
                    </span>
                    <h3 className="text-sm font-bold text-foreground">{f.type}</h3>

                    {/* Source Tag (Git vs Webhook) */}
                    {isGitFinding ? (
                      <span className="flex items-center gap-1 text-xs font-medium text-purple-400 bg-purple-500/10 px-2 py-0.5 rounded-md border border-purple-500/20">
                        <GitBranch className="h-3 w-3" />
                        <span>{f.repository || "Git Tarama"}</span>
                      </span>
                    ) : (
                      f.endpointName && (
                        <span className="flex items-center gap-1 text-xs font-medium text-muted-foreground bg-secondary px-2 py-0.5 rounded-md">
                          <Globe className="h-3 w-3 text-emerald-400" />
                          <span>{f.endpointName}</span>
                        </span>
                      )
                    )}
                  </div>

                  <span
                    className={`rounded-full px-2.5 py-0.5 text-[11px] font-bold ${
                      f.action === "BLOCK"
                        ? "bg-rose-500/15 text-rose-400 border border-rose-500/30"
                        : "bg-emerald-500/15 text-emerald-400 border border-emerald-500/30"
                    }`}
                  >
                    POLİTİKA: {f.action}
                  </span>
                </div>

                {/* Finding Details */}
                <div className="space-y-2 text-xs">
                  <p className="text-foreground leading-relaxed">{f.message}</p>

                  {/* File Path & Line Info for Git findings */}
                  {f.filePath && (
                    <div className="flex items-center gap-2 font-mono text-muted-foreground">
                      <span className="flex items-center gap-1 text-purple-400">
                        <FileCode className="h-3.5 w-3.5" />
                        <span>Dosya / Konum:</span>
                      </span>
                      <span className="rounded-lg bg-background px-2.5 py-1 text-foreground border border-border">
                        {f.filePath}
                        {f.lineNumber ? `:${f.lineNumber}` : ""}
                      </span>
                    </div>
                  )}

                  {f.evidenceMasked && (
                    <div className="flex items-center gap-2 font-mono text-muted-foreground">
                      <span>Maskeli Kanıt:</span>
                      <span className="rounded-lg bg-background px-2.5 py-1 text-foreground border border-border">
                        {f.evidenceMasked}
                      </span>
                    </div>
                  )}
                </div>

                {/* Card Footer: Timestamp & Inline AI Button */}
                <div className="flex items-center justify-between pt-3 border-t border-border">
                  <span className="text-[11px] text-muted-foreground font-mono">
                    {new Date(f.createdAt).toLocaleString("tr-TR")}
                  </span>

                  <button
                    onClick={() => handleToggleExplain(f)}
                    className={`flex items-center gap-1.5 rounded-xl px-3.5 py-1.5 text-xs font-semibold transition shadow-sm ${
                      isExpanded
                        ? "bg-purple-600 text-white hover:bg-purple-700"
                        : "bg-secondary text-foreground hover:bg-primary hover:text-primary-foreground"
                    }`}
                  >
                    <Sparkles className={`h-3.5 w-3.5 ${isExpanded ? "text-white" : "text-purple-400"}`} />
                    <span>{isExpanded ? "Rehberi Gizle" : "🤖 AI Çözüm Rehberi"}</span>
                    {isExpanded ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
                  </button>
                </div>

                {/* INLINE EXPANDABLE AI REMEDIATION DRAWER (Opens right below this card!) */}
                {isExpanded && (
                  <div className="mt-2 rounded-xl border border-purple-500/30 bg-purple-500/5 p-5 space-y-4 animate-in fade-in slide-in-from-top-2 duration-200">
                    {explainMutation.isPending ? (
                      <div className="flex flex-col items-center justify-center py-8 text-center space-y-2">
                        <Loader2 className="h-6 w-6 animate-spin text-purple-400" />
                        <p className="text-xs font-semibold text-foreground">AI Güvenlik Analizi Yapılıyor...</p>
                        <p className="text-[11px] text-muted-foreground">
                          Maskeli kanıt inceleniyor, kök neden analizi ve güvenli kod bloğu üretiliyor.
                        </p>
                      </div>
                    ) : explainMutation.data ? (
                      <>
                        {/* Title & Model Tag */}
                        <div className="flex items-center justify-between border-b border-purple-500/20 pb-3">
                          <div className="flex items-center gap-2">
                            <Bot className="h-5 w-5 text-purple-400" />
                            <div>
                              <h3 className="text-sm font-bold text-foreground">
                                {explainMutation.data.title}
                              </h3>
                              {explainMutation.data.provider && (
                                <span className="text-[10px] font-mono text-purple-400/90">
                                  Model: {explainMutation.data.provider}
                                </span>
                              )}
                            </div>
                          </div>

                          <div className="flex items-center gap-3">
                            <span className="rounded-lg bg-purple-500/15 px-2.5 py-0.5 text-xs font-mono font-bold text-purple-400">
                              Güven: %{(explainMutation.data.confidenceScore * 100).toFixed(0)}
                            </span>
                            <button
                              onClick={() => setActiveFinding(null)}
                              className="rounded-lg p-1 text-muted-foreground hover:text-foreground hover:bg-background transition"
                              title="Kapat"
                            >
                              <X className="h-4 w-4" />
                            </button>
                          </div>
                        </div>

                        {/* Root Cause & Impact Grid */}
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
                          <div className="rounded-lg bg-card p-3 border border-border space-y-1">
                            <h4 className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                              Kök Neden (Root Cause)
                            </h4>
                            <p className="text-foreground leading-relaxed">
                              {explainMutation.data.rootCause}
                            </p>
                          </div>

                          <div className="rounded-lg bg-card p-3 border border-border space-y-1">
                            <h4 className="text-[10px] font-bold uppercase tracking-wider text-rose-400">
                              Güvenlik & İş Riski (Impact)
                            </h4>
                            <p className="text-rose-400/90 leading-relaxed">
                              {explainMutation.data.impact}
                            </p>
                          </div>
                        </div>

                        {/* Remediation Steps */}
                        <div className="rounded-lg bg-card p-3.5 border border-border space-y-2 text-xs">
                          <h4 className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                            Önerilen Çözüm Adımları (Remediation)
                          </h4>
                          <ul className="space-y-1.5 text-foreground">
                            {explainMutation.data.remediationSteps.map((step, idx) => (
                              <li key={idx} className="flex items-start gap-2">
                                <span className="text-purple-400 font-bold">•</span>
                                <span>{step}</span>
                              </li>
                            ))}
                          </ul>
                        </div>

                        {/* Code Snippet with Copy Button */}
                        <div className="space-y-1.5">
                          <div className="flex items-center justify-between">
                            <h4 className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                              Örnek Güvenli Kod Bloğu
                            </h4>
                            <button
                              onClick={() => handleCopyCode(explainMutation.data!.codeSnippet)}
                              className="flex items-center gap-1 text-[11px] text-purple-400 hover:text-purple-300 font-semibold"
                            >
                              {copiedCode ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
                              <span>{copiedCode ? "Kopyalandı!" : "Kodu Kopyala"}</span>
                            </button>
                          </div>
                          <pre className="rounded-xl bg-background p-4 text-[11px] font-mono text-foreground border border-border overflow-x-auto leading-relaxed">
                            {explainMutation.data.codeSnippet}
                          </pre>
                        </div>
                      </>
                    ) : null}
                  </div>
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
