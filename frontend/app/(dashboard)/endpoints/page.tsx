"use client";

import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import { Project, Endpoint, EndpointMode } from "@apisentinel/shared";
import {
  Radio,
  Plus,
  Copy,
  Check,
  Globe,
  ShieldAlert,
  ArrowRight,
  Loader2,
  AlertCircle,
  FolderGit2,
  ExternalLink,
  Send,
  Zap,
} from "lucide-react";
import Link from "next/link";

export default function EndpointsPage() {
  const queryClient = useQueryClient();
  const { accessToken, organization } = useAuth();

  const [selectedProjectId, setSelectedProjectId] = useState<string>("");
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [copiedSlug, setCopiedSlug] = useState<string | null>(null);
  const [testStatus, setTestStatus] = useState<Record<string, string>>({});

  // Form state
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [mode, setMode] = useState<EndpointMode>(EndpointMode.PASS);
  const [upstreamUrl, setUpstreamUrl] = useState("");
  const [createError, setCreateError] = useState<string | null>(null);

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

  // Fetch endpoints for active project
  const { data: endpointsData, isLoading } = useQuery({
    queryKey: ["endpoints", activeProjectId],
    queryFn: () =>
      apiFetch<{ endpoints: (Endpoint & { requestCount: number })[] }>(
        `/api/projects/${activeProjectId}/endpoints`,
        {
          token: accessToken,
          organizationId: organization?.id,
        }
      ),
    enabled: !!accessToken && !!activeProjectId,
  });

  // Create endpoint mutation
  const createMutation = useMutation({
    mutationFn: (input: { name: string; slug?: string; mode: EndpointMode; upstreamUrl?: string }) =>
      apiFetch<Endpoint>(`/api/projects/${activeProjectId}/endpoints`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["endpoints", activeProjectId] });
      setName("");
      setSlug("");
      setUpstreamUrl("");
      setMode(EndpointMode.PASS);
      setIsCreateOpen(false);
      setCreateError(null);
    },
    onError: (err: any) => {
      setCreateError(err.message || "Endpoint oluşturulamadı.");
    },
  });

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    createMutation.mutate({
      name: name.trim(),
      slug: slug.trim() || undefined,
      mode,
      upstreamUrl: upstreamUrl.trim() || undefined,
    });
  };

  const copyWebhookUrl = (slug: string) => {
    const backendUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3001";
    const fullUrl = `${backendUrl}/hook/${slug}`;
    navigator.clipboard.writeText(fullUrl);
    setCopiedSlug(slug);
    setTimeout(() => setCopiedSlug(null), 2000);
  };

  const sendTestWebhook = async (slug: string) => {
    const backendUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3001";
    setTestStatus((prev) => ({ ...prev, [slug]: "sending" }));

    try {
      const res = await fetch(`${backendUrl}/hook/${slug}`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "x-test-source": "apisentinel-console" },
        body: JSON.stringify({
          event: "payment.succeeded",
          amount: 2500,
          currency: "try",
          customer: {
            name: "Ahmet Yılmaz",
            email: "ahmet@example.com",
          },
          timestamp: new Date().toISOString(),
        }),
      });

      if (res.ok) {
        setTestStatus((prev) => ({ ...prev, [slug]: "success" }));
        queryClient.invalidateQueries({ queryKey: ["endpoints", activeProjectId] });
      } else {
        setTestStatus((prev) => ({ ...prev, [slug]: "error" }));
      }
    } catch {
      setTestStatus((prev) => ({ ...prev, [slug]: "error" }));
    } finally {
      setTimeout(() => {
        setTestStatus((prev) => {
          const next = { ...prev };
          delete next[slug];
          return next;
        });
      }, 3000);
    }
  };

  const endpointsList = endpointsData?.endpoints || [];

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Webhook Endpoint'leri</h1>
          <p className="text-sm text-muted-foreground">
            Dış servislerden gelen API & Webhook trafiğini karşılayın ve yönetin
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
            onClick={() => setIsCreateOpen(true)}
            disabled={!activeProjectId}
            className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
          >
            <Plus className="h-4 w-4" />
            <span>Yeni Endpoint</span>
          </button>
        </div>
      </div>

      {/* Create Modal / Form */}
      {isCreateOpen && (
        <div className="rounded-xl border border-border bg-card p-6 shadow-sm animate-in fade-in duration-200">
          <div className="flex items-center justify-between border-b border-border pb-4 mb-4">
            <h3 className="text-base font-semibold">Yeni Webhook Endpoint Tanımla</h3>
            <button
              onClick={() => {
                setIsCreateOpen(false);
                setCreateError(null);
              }}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              Vazgeç
            </button>
          </div>

          {createError && (
            <div className="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive mb-4">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{createError}</span>
            </div>
          )}

          <form onSubmit={handleCreate} className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                  Endpoint Adı
                </label>
                <input
                  type="text"
                  required
                  value={name}
                  onChange={(e) => {
                    setName(e.target.value);
                    if (!slug) {
                      setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9]/g, "-"));
                    }
                  }}
                  placeholder="Örn: Stripe Payments Production"
                  className="w-full rounded-lg border border-input bg-background/50 px-3 py-2 text-sm placeholder:text-muted-foreground/60 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                  Slug (URL Yolu)
                </label>
                <div className="flex items-center rounded-lg border border-input bg-background/50 px-3 py-2 text-sm focus-within:border-primary focus-within:ring-1 focus-within:ring-primary">
                  <span className="text-muted-foreground mr-1">/hook/</span>
                  <input
                    type="text"
                    value={slug}
                    onChange={(e) => setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""))}
                    placeholder="stripe-prod"
                    className="w-full bg-transparent focus:outline-none"
                  />
                </div>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                  Çalışma Modu
                </label>
                <select
                  value={mode}
                  onChange={(e) => setMode(e.target.value as EndpointMode)}
                  className="w-full rounded-lg border border-input bg-background/50 px-3 py-2 text-sm focus:border-primary focus:outline-none"
                >
                  <option value={EndpointMode.PASS}>PASS (Normal Geçir ve Kaydet)</option>
                  <option value={EndpointMode.CAPTURE_ONLY}>CAPTURE_ONLY (Sadece Kaydet, İletme)</option>
                  <option value={EndpointMode.BLOCK}>BLOCK (Gelen Tüm İstekleri 403 ile Reddet)</option>
                  <option value={EndpointMode.MOCK}>MOCK (Sahte Yanıt Dön)</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                  Upstream URL (Opsiyonel İletim Hedefi)
                </label>
                <input
                  type="url"
                  value={upstreamUrl}
                  onChange={(e) => setUpstreamUrl(e.target.value)}
                  placeholder="https://api.mycompany.com/webhooks/stripe"
                  className="w-full rounded-lg border border-input bg-background/50 px-3 py-2 text-sm placeholder:text-muted-foreground/60 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                />
              </div>
            </div>

            <div className="flex justify-end pt-2">
              <button
                type="submit"
                disabled={createMutation.isPending}
                className="flex items-center gap-2 rounded-lg bg-primary px-5 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
              >
                {createMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    <span>Kaydediliyor...</span>
                  </>
                ) : (
                  <>
                    <span>Endpoint'i Başlat</span>
                    <Zap className="h-4 w-4" />
                  </>
                )}
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Endpoints List */}
      {projects.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border py-16 text-center bg-card/40">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-amber-500/10 text-amber-400 mb-4">
            <FolderGit2 className="h-6 w-6" />
          </div>
          <h3 className="text-base font-semibold text-foreground">Henüz Oluşturulmuş Bir Projeniz Yok</h3>
          <p className="mt-1 text-sm text-muted-foreground max-w-sm">
            Webhook endpoint'i tanımlayabilmek için öncelikle en az bir Proje (örn: E-Ticaret, Ödeme Geçidi) oluşturmalısınız.
          </p>
          <Link
            href="/projects"
            className="mt-6 flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90"
          >
            <Plus className="h-4 w-4" />
            <span>Önce Bir Proje Oluştur</span>
          </Link>
        </div>
      ) : isLoading ? (
        <div className="flex h-48 items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      ) : endpointsList.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border py-16 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-secondary text-muted-foreground mb-4">
            <Radio className="h-6 w-6" />
          </div>
          <h3 className="text-base font-semibold">Bu projede henüz endpoint tanımlanmadı</h3>
          <p className="mt-1 text-sm text-muted-foreground max-w-sm">
            Webhook sağlayıcılarınızı (Stripe, GitHub, iyzico vb.) yönlendirebileceğiniz bir endpoint oluşturun.
          </p>
          <button
            onClick={() => setIsCreateOpen(true)}
            className="mt-6 flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90"
          >
            <Plus className="h-4 w-4" />
            <span>İlk Webhook Endpoint'ini Oluştur</span>
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-6">
          {endpointsList.map((endpoint) => {
            const isCopied = copiedSlug === endpoint.slug;
            const status = testStatus[endpoint.slug];

            return (
              <div
                key={endpoint.id}
                className="flex flex-col justify-between rounded-xl border border-border bg-card p-6 shadow-sm transition hover:border-border/80"
              >
                <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="space-y-1">
                    <div className="flex items-center gap-3">
                      <h3 className="text-base font-bold text-foreground">{endpoint.name}</h3>
                      <span
                        className={`rounded-full px-2.5 py-0.5 text-[11px] font-semibold ${
                          endpoint.mode === EndpointMode.PASS
                            ? "bg-emerald-500/10 text-emerald-400"
                            : endpoint.mode === EndpointMode.BLOCK
                            ? "bg-rose-500/10 text-rose-400"
                            : "bg-blue-500/10 text-blue-400"
                        }`}
                      >
                        {endpoint.mode}
                      </span>
                    </div>

                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <span>Yakalanan İstek:</span>
                      <span className="font-bold text-foreground">{endpoint.requestCount}</span>
                      {endpoint.upstreamUrl && (
                        <>
                          <span>•</span>
                          <span className="truncate max-w-xs">İletim: {endpoint.upstreamUrl}</span>
                        </>
                      )}
                    </div>
                  </div>

                  {/* Webhook URL Box */}
                  <div className="flex items-center gap-2 rounded-lg border border-border bg-background/50 p-1.5 pl-3">
                    <code className="text-xs font-mono text-foreground select-all">
                      http://localhost:3001/hook/{endpoint.slug}
                    </code>
                    <button
                      onClick={() => copyWebhookUrl(endpoint.slug)}
                      className="flex h-7 w-7 items-center justify-center rounded bg-secondary text-muted-foreground transition hover:text-foreground"
                      title="URL'i Kopyala"
                    >
                      {isCopied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
                    </button>
                  </div>
                </div>

                <div className="mt-6 pt-4 border-t border-border flex flex-wrap items-center justify-between gap-3">
                  <button
                    onClick={() => sendTestWebhook(endpoint.slug)}
                    disabled={status === "sending"}
                    className="flex items-center gap-1.5 rounded-lg border border-border bg-secondary/50 px-3 py-1.5 text-xs font-semibold text-foreground transition hover:bg-secondary disabled:opacity-50"
                  >
                    {status === "sending" ? (
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    ) : status === "success" ? (
                      <Check className="h-3.5 w-3.5 text-emerald-400" />
                    ) : (
                      <Send className="h-3.5 w-3.5 text-primary" />
                    )}
                    <span>
                      {status === "sending"
                        ? "Gönderiliyor..."
                        : status === "success"
                        ? "İstek Yakalandı! (200 OK)"
                        : "Test Webhook'u Gönder"}
                    </span>
                  </button>

                  <Link
                    href={`/requests?endpointId=${endpoint.id}`}
                    className="flex items-center gap-1.5 text-xs font-semibold text-primary hover:underline"
                  >
                    <span>İstekleri İncele ({endpoint.requestCount})</span>
                    <ArrowRight className="h-3.5 w-3.5" />
                  </Link>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
