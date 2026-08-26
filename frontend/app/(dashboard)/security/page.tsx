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
} from "lucide-react";

interface Finding {
  id: string;
  requestId: string;
  reqDisplayId: string;
  endpointName: string;
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
}

export default function SecurityFindingsPage() {
  const { accessToken, organization } = useAuth();
  const { projects, activeProjectId, setActiveProjectId } = useActiveProject();

  const [activeFinding, setActiveFinding] = useState<Finding | null>(null);
  const [severityFilter, setSeverityFilter] = useState<"ALL" | "CRITICAL" | "HIGH" | "MEDIUM">("ALL");
  const [copiedCode, setCopiedCode] = useState(false);

  // 1. Fetch Real Security Findings from DB
  const {
    data: findingsData,
    isLoading: isFindingsLoading,
    refetch: refetchFindings,
  } = useQuery({
    queryKey: ["findings", activeProjectId],
    queryFn: () =>
      apiFetch<{ findings: Finding[] }>(`/api/projects/${activeProjectId}/findings`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
  });

  // 3. Fetch Real Finding Statistics
  const { data: statsData, refetch: refetchStats } = useQuery({
    queryKey: ["findingStats", activeProjectId],
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

  // 4. AI Explain Mutation
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

  const handleExplain = (finding: Finding) => {
    setActiveFinding(finding);
    explainMutation.mutate(finding);
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

  // Filtered findings
  const filteredFindings = findings.filter((f) => {
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
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
              <ShieldAlert className="h-6 w-6 text-rose-500" />
              Güvenlik Bulguları & Tehdit Analizi
            </h1>
            <span className="flex items-center gap-1 rounded-full bg-purple-500/10 px-2.5 py-0.5 text-xs font-semibold text-purple-400">
              <Bot className="h-3 w-3" />
              AI Remediation Active
            </span>
          </div>
          <p className="text-sm text-muted-foreground mt-1">
            API ve Webhook trafiğinde tespit edilip PostgreSQL'e kaydedilen gerçek PII, Secret, SQLi ve Politika ihlalleri
          </p>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={handleRefresh}
            className="flex items-center gap-1.5 rounded-xl border border-border bg-card px-3 py-2 text-xs font-semibold text-foreground hover:bg-secondary transition"
          >
            <RotateCw className="h-3.5 w-3.5" />
            <span>Yenile</span>
          </button>

          {projects.length > 0 && (
            <select
              value={activeProjectId}
              onChange={(e) => setActiveProjectId(e.target.value)}
              className="rounded-xl border border-border bg-card px-3 py-2 text-xs font-semibold text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
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

      {/* Summary KPI Cards */}
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        <div className="rounded-2xl border border-rose-500/20 bg-rose-500/5 p-5">
          <div className="flex items-center justify-between text-xs font-semibold text-rose-400">
            <span>Kritik İhlaller</span>
            <AlertOctagon className="h-4 w-4" />
          </div>
          <p className="text-3xl font-bold text-rose-400 mt-2">{stats.criticalCount}</p>
          <p className="text-[11px] text-muted-foreground mt-1">API Keys & SQL Injection</p>
        </div>

        <div className="rounded-2xl border border-amber-500/20 bg-amber-500/5 p-5">
          <div className="flex items-center justify-between text-xs font-semibold text-amber-400">
            <span>Yüksek Seviye</span>
            <AlertTriangle className="h-4 w-4" />
          </div>
          <p className="text-3xl font-bold text-amber-400 mt-2">{stats.highCount}</p>
          <p className="text-[11px] text-muted-foreground mt-1">TCKN, IBAN & Özel Anahtarlar</p>
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

      {/* Filter Tabs Bar */}
      <div className="flex items-center gap-2">
        <SlidersHorizontal className="h-4 w-4 text-muted-foreground" />
        <div className="flex items-center gap-1.5 text-xs">
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

      {/* Main Grid: Real Findings List + AI Remediation Drawer */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
        {/* Left: Findings Cards (7 cols) */}
        <div className="space-y-4 lg:col-span-7">
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
                Seçili filtre veya projede engellenen bir tehdit bulunmuyor. Webhook gönderildiğinde bulgular anlık burada listelenecektir.
              </p>
            </div>
          ) : (
            filteredFindings.map((f) => (
              <div
                key={f.id}
                className={`rounded-2xl border p-5 shadow-sm transition flex flex-col justify-between gap-4 ${
                  activeFinding?.id === f.id
                    ? "border-primary bg-primary/5 shadow-md ring-1 ring-primary/40"
                    : "border-border bg-card hover:border-border/80 hover:bg-secondary/20"
                }`}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2.5">
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
                    <span className="text-xs text-muted-foreground">• {f.endpointName}</span>
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

                <div className="space-y-2 text-xs">
                  <p className="text-foreground leading-relaxed">{f.message}</p>
                  {f.evidenceMasked && (
                    <div className="flex items-center gap-2 font-mono text-muted-foreground">
                      <span>Maskeli Kanıt:</span>
                      <span className="rounded-lg bg-background px-2.5 py-1 text-foreground border border-border">
                        {f.evidenceMasked}
                      </span>
                    </div>
                  )}
                </div>

                <div className="flex items-center justify-between pt-3 border-t border-border">
                  <span className="text-[11px] text-muted-foreground font-mono">
                    {new Date(f.createdAt).toLocaleString("tr-TR")}
                  </span>

                  <button
                    onClick={() => handleExplain(f)}
                    className="flex items-center gap-1.5 rounded-xl bg-secondary px-3.5 py-1.5 text-xs font-semibold text-foreground transition hover:bg-primary hover:text-primary-foreground shadow-sm"
                  >
                    <Sparkles className="h-3.5 w-3.5 text-purple-400" />
                    <span>🤖 AI ile Açıkla & Çöz</span>
                  </button>
                </div>
              </div>
            ))
          )}
        </div>

        {/* Right: AI Explanation & Remediation Panel (5 cols) */}
        <div className="lg:col-span-5">
          {explainMutation.isPending ? (
            <div className="flex flex-col items-center justify-center rounded-2xl border border-border bg-card p-12 text-center h-[460px]">
              <Loader2 className="h-8 w-8 animate-spin text-primary mb-3" />
              <p className="text-sm font-semibold">AI Güvenlik Analizi Yapılıyor...</p>
              <p className="text-xs text-muted-foreground mt-1 max-w-xs">
                Maskeli kanıtlar inceleniyor, kök neden analizi ve güvenli kod bloğu üretiliyor.
              </p>
            </div>
          ) : explainMutation.data ? (
            <div className="rounded-2xl border border-purple-500/30 bg-card p-6 shadow-sm space-y-4 animate-in fade-in duration-200 sticky top-20">
              <div className="flex items-center justify-between border-b border-border pb-3">
                <div className="flex items-center gap-2">
                  <Bot className="h-5 w-5 text-purple-400" />
                  <h3 className="text-sm font-bold text-foreground">{explainMutation.data.title}</h3>
                </div>
                <span className="rounded-lg bg-purple-500/10 px-2 py-0.5 text-xs font-mono font-bold text-purple-400">
                  Güven: %{(explainMutation.data.confidenceScore * 100).toFixed(0)}
                </span>
              </div>

              <div>
                <h4 className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground mb-1">
                  Kök Neden (Root Cause)
                </h4>
                <p className="text-xs text-foreground leading-relaxed">
                  {explainMutation.data.rootCause}
                </p>
              </div>

              <div>
                <h4 className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground mb-1">
                  Güvenlik & İş Riski (Impact)
                </h4>
                <p className="text-xs text-rose-400 leading-relaxed">
                  {explainMutation.data.impact}
                </p>
              </div>

              <div>
                <h4 className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground mb-1">
                  Önerilen Çözüm Adımları (Remediation)
                </h4>
                <ul className="space-y-1.5 text-xs text-muted-foreground">
                  {explainMutation.data.remediationSteps.map((step, idx) => (
                    <li key={idx} className="flex items-start gap-1.5 text-foreground">
                      <span className="text-primary font-bold">•</span>
                      <span>{step}</span>
                    </li>
                  ))}
                </ul>
              </div>

              <div>
                <div className="flex items-center justify-between mb-1">
                  <h4 className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                    Örnek Güvenli Kod Bloğu
                  </h4>
                  <button
                    onClick={() => handleCopyCode(explainMutation.data!.codeSnippet)}
                    className="flex items-center gap-1 text-[11px] text-primary hover:underline font-semibold"
                  >
                    {copiedCode ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
                    <span>{copiedCode ? "Kopyalandı!" : "Kodu Kopyala"}</span>
                  </button>
                </div>
                <pre className="rounded-xl bg-background p-3.5 text-[11px] font-mono text-foreground border border-border overflow-x-auto leading-relaxed">
                  {explainMutation.data.codeSnippet}
                </pre>
              </div>
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border bg-card/40 p-12 text-center text-muted-foreground h-[460px]">
              <Bot className="h-10 w-10 mx-auto mb-3 text-muted-foreground/60" />
              <p className="text-sm font-semibold text-foreground">AI Güvenlik & Çözüm Danışmanı</p>
              <p className="text-xs text-muted-foreground mt-1 max-w-xs">
                Herhangi bir bulgunun yanındaki "🤖 AI ile Açıkla & Çöz" butonuna basarak kök neden analizi ve doğrudan uygulanabilir kod bloğunu görüntüleyin.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

