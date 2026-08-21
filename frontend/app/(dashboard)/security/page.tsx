"use client";

import React, { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import { Project, CapturedRequest } from "@apisentinel/shared";
import {
  ShieldAlert,
  ShieldCheck,
  Bot,
  Sparkles,
  Search,
  Code,
  FileJson,
  Layers,
  Loader2,
  CheckCircle2,
  AlertTriangle,
  ArrowRight,
  Zap,
} from "lucide-react";

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

  const [selectedProjectId, setSelectedProjectId] = useState<string>("");
  const [activeFinding, setActiveFinding] = useState<any | null>(null);

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

  // AI Explain Mutation
  const explainMutation = useMutation({
    mutationFn: (finding: any) =>
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

  // Example demo findings based on security scanner rule engine
  const demoFindings = [
    {
      id: "f-1",
      category: "SECRET",
      type: "AWS_KEY",
      severity: "CRITICAL",
      action: "BLOCK",
      endpointName: "Stripe Production Hook",
      message: "AWS Access Key ID detected in webhook payload",
      evidenceMasked: "AKIA****1234",
      createdAt: new Date().toISOString(),
    },
    {
      id: "f-2",
      category: "PII",
      type: "CREDIT_CARD",
      severity: "CRITICAL",
      action: "BLOCK",
      endpointName: "Payment Webhook Gateway",
      message: "Valid credit card number detected in raw request",
      evidenceMasked: "************0366",
      createdAt: new Date(Date.now() - 3600000).toISOString(),
    },
    {
      id: "f-3",
      category: "PII",
      type: "TCKN",
      severity: "HIGH",
      action: "BLOCK",
      endpointName: "Kyc Verification Hook",
      message: "Verified Turkish National Identity (TCKN) detected",
      evidenceMasked: "*********46",
      createdAt: new Date(Date.now() - 7200000).toISOString(),
    },
    {
      id: "f-4",
      category: "PII",
      type: "EMAIL",
      severity: "INFO",
      action: "MASK",
      endpointName: "Customer Support Hook",
      message: "Personal email address detected, applied masking",
      evidenceMasked: "a***z@example.com",
      createdAt: new Date(Date.now() - 10800000).toISOString(),
    },
  ];

  const handleExplain = (finding: any) => {
    setActiveFinding(finding);
    explainMutation.mutate(finding);
  };

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-bold tracking-tight">Güvenlik Bulguları (Security Engine)</h1>
            <span className="flex items-center gap-1 rounded-full bg-purple-500/10 px-2.5 py-0.5 text-xs font-semibold text-purple-400">
              <Bot className="h-3 w-3" />
              AI Remediation Active
            </span>
          </div>
          <p className="text-sm text-muted-foreground">
            API ve Webhook trafiğinde tespit edilen PII, Secret, Enjeksiyon ve Şema ihlallerini inceleyin
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
        </div>
      </div>

      {/* Main Grid: Findings List + AI Remediation Drawer */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
        {/* Left: Findings Cards (7 cols) */}
        <div className="space-y-4 lg:col-span-7">
          {demoFindings.map((f) => (
            <div
              key={f.id}
              className={`rounded-xl border p-5 shadow-sm transition flex flex-col justify-between gap-4 ${
                activeFinding?.id === f.id
                  ? "border-primary bg-primary/5 shadow-md"
                  : "border-border bg-card hover:border-border/80"
              }`}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <span
                    className={`rounded px-2 py-0.5 text-xs font-bold ${
                      f.severity === "CRITICAL"
                        ? "bg-rose-500/20 text-rose-400"
                        : f.severity === "HIGH"
                        ? "bg-amber-500/20 text-amber-400"
                        : "bg-blue-500/20 text-blue-400"
                    }`}
                  >
                    {f.severity}
                  </span>
                  <h3 className="text-sm font-bold text-foreground">{f.type}</h3>
                  <span className="text-xs text-muted-foreground">• {f.endpointName}</span>
                </div>

                <span
                  className={`rounded-full px-2.5 py-0.5 text-[11px] font-semibold ${
                    f.action === "BLOCK"
                      ? "bg-rose-500/10 text-rose-400"
                      : "bg-emerald-500/10 text-emerald-400"
                  }`}
                >
                  POLİTİKA: {f.action}
                </span>
              </div>

              <div className="space-y-1 text-xs">
                <p className="text-foreground">{f.message}</p>
                <div className="flex items-center gap-2 font-mono text-muted-foreground">
                  <span>Maskeli Kanıt:</span>
                  <span className="rounded bg-background px-2 py-0.5 text-foreground border border-border">
                    {f.evidenceMasked}
                  </span>
                </div>
              </div>

              <div className="flex items-center justify-between pt-2 border-t border-border">
                <span className="text-[11px] text-muted-foreground font-mono">
                  {new Date(f.createdAt).toLocaleTimeString("tr-TR")}
                </span>

                <button
                  onClick={() => handleExplain(f)}
                  className="flex items-center gap-1.5 rounded-lg bg-secondary px-3 py-1.5 text-xs font-semibold text-foreground transition hover:bg-primary hover:text-primary-foreground"
                >
                  <Sparkles className="h-3.5 w-3.5 text-purple-400" />
                  <span>🤖 AI ile Açıkla & Çöz</span>
                </button>
              </div>
            </div>
          ))}
        </div>

        {/* Right: AI Explanation & Remediation Panel (5 cols) */}
        <div className="lg:col-span-5">
          {explainMutation.isPending ? (
            <div className="flex flex-col items-center justify-center rounded-xl border border-border bg-card p-12 text-center h-[420px]">
              <Loader2 className="h-8 w-8 animate-spin text-primary mb-3" />
              <p className="text-sm font-semibold">AI Güvenlik Analizi Yapılıyor...</p>
              <p className="text-xs text-muted-foreground mt-1">
                Maskeli kanıtlar inceleniyor, kök neden ve çözüm adımları üretiliyor.
              </p>
            </div>
          ) : explainMutation.data ? (
            <div className="rounded-xl border border-purple-500/30 bg-card p-6 shadow-sm space-y-4 animate-in fade-in duration-200">
              <div className="flex items-center justify-between border-b border-border pb-3">
                <div className="flex items-center gap-2">
                  <Bot className="h-5 w-5 text-purple-400" />
                  <h3 className="text-sm font-bold text-foreground">{explainMutation.data.title}</h3>
                </div>
                <span className="rounded bg-purple-500/10 px-2 py-0.5 text-xs font-mono font-bold text-purple-400">
                  Güven: %{(explainMutation.data.confidenceScore * 100).toFixed(0)}
                </span>
              </div>

              <div>
                <h4 className="text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1">
                  Kök Neden (Root Cause)
                </h4>
                <p className="text-xs text-foreground leading-relaxed">
                  {explainMutation.data.rootCause}
                </p>
              </div>

              <div>
                <h4 className="text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1">
                  Güvenlik & İş Riski (Impact)
                </h4>
                <p className="text-xs text-rose-400 leading-relaxed">
                  {explainMutation.data.impact}
                </p>
              </div>

              <div>
                <h4 className="text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1">
                  Önerilen Çözüm Adımları (Remediation)
                </h4>
                <ul className="space-y-1 text-xs text-muted-foreground">
                  {explainMutation.data.remediationSteps.map((step, idx) => (
                    <li key={idx} className="flex items-start gap-1.5 text-foreground">
                      <span>•</span>
                      <span>{step}</span>
                    </li>
                  ))}
                </ul>
              </div>

              <div>
                <h4 className="text-xs font-bold uppercase tracking-wider text-muted-foreground mb-1">
                  Örnek Güvenli Kod Bloğu
                </h4>
                <pre className="rounded bg-background p-3 text-[11px] font-mono text-foreground border border-border overflow-x-auto">
                  {explainMutation.data.codeSnippet}
                </pre>
              </div>
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border bg-card p-12 text-center text-muted-foreground h-[420px]">
              <Bot className="h-8 w-8 mx-auto mb-2 text-muted-foreground/60" />
              <p className="text-sm font-medium">AI Çözüm Danışmanı</p>
              <p className="text-xs text-muted-foreground mt-1 max-w-xs">
                Herhangi bir bulgunun yanındaki "🤖 AI ile Açıkla & Çöz" butonuna basarak kök neden analizi ve çözüm kodunu görüntüleyin.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
