"use client";

import React, { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import { Project, Endpoint } from "@apisentinel/shared";
import {
  FileCode,
  CheckCircle2,
  AlertTriangle,
  Code,
  FileJson,
  Layers,
  Save,
  Trash2,
  Loader2,
  ShieldCheck,
  Sparkles,
  Check,
  AlignLeft,
  XCircle,
} from "lucide-react";

const PRESET_STRIPE = {
  $schema: "https://json-schema.org/draft/2020-12/schema",
  title: "StripeWebhookPayload",
  type: "object",
  properties: {
    event: { type: "string", enum: ["payment.success", "payment.failed", "refund.created"] },
    amount: { type: "integer", minimum: 1 },
    currency: { type: "string", minLength: 3, maxLength: 3 },
    customer: {
      type: "object",
      properties: {
        id: { type: "string" },
        email: { type: "string", format: "email" },
      },
      required: ["id", "email"],
    },
  },
  required: ["event", "amount", "currency", "customer"],
};

const PRESET_GITHUB = {
  $schema: "https://json-schema.org/draft/2020-12/schema",
  title: "GitHubWebhookPayload",
  type: "object",
  properties: {
    action: { type: "string" },
    repository: {
      type: "object",
      properties: {
        name: { type: "string" },
        full_name: { type: "string" },
      },
      required: ["name"],
    },
    sender: {
      type: "object",
      properties: {
        login: { type: "string" },
      },
      required: ["login"],
    },
  },
  required: ["repository", "sender"],
};

const PRESET_GENERIC = {
  $schema: "https://json-schema.org/draft/2020-12/schema",
  title: "GenericApiPayload",
  type: "object",
  properties: {
    event: { type: "string" },
    timestamp: { type: "string" },
    data: { type: "object" },
  },
  required: ["event", "data"],
};

export default function ContractsPage() {
  const queryClient = useQueryClient();
  const { accessToken, organization } = useAuth();

  const [selectedProjectId, setSelectedProjectId] = useState<string>("");
  const [selectedEndpointId, setSelectedEndpointId] = useState<string>("");
  const [schemaText, setSchemaText] = useState<string>(JSON.stringify(PRESET_STRIPE, null, 2));
  const [syntaxValid, setSyntaxValid] = useState(true);
  const [statusMessage, setStatusMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  // 1. Fetch projects
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

  // 2. Fetch endpoints for active project
  const { data: endpointsData } = useQuery({
    queryKey: ["endpoints", activeProjectId],
    queryFn: () =>
      apiFetch<{ endpoints: Endpoint[] }>(`/api/projects/${activeProjectId}/endpoints`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeProjectId,
  });

  const endpoints = endpointsData?.endpoints || [];
  const activeEndpointId = selectedEndpointId || (endpoints[0]?.id ?? "");

  // 3. Fetch existing schema for active endpoint
  const { data: schemaData, isLoading: isSchemaLoading } = useQuery({
    queryKey: ["endpointSchema", activeEndpointId],
    queryFn: async () => {
      try {
        return await apiFetch<any>(`/api/endpoints/${activeEndpointId}/schema`, {
          token: accessToken,
          organizationId: organization?.id,
        });
      } catch {
        return null;
      }
    },
    enabled: !!accessToken && !!activeEndpointId && !!organization?.id,
  });

  useEffect(() => {
    if (schemaData?.json_schema) {
      try {
        const parsed = typeof schemaData.json_schema === "string" 
          ? JSON.parse(schemaData.json_schema) 
          : schemaData.json_schema;
        setSchemaText(JSON.stringify(parsed, null, 2));
      } catch {
        setSchemaText(JSON.stringify(schemaData.json_schema, null, 2));
      }
    } else {
      setSchemaText(JSON.stringify(PRESET_STRIPE, null, 2));
    }
  }, [schemaData, activeEndpointId]);

  // Validate syntax on schemaText change
  useEffect(() => {
    try {
      JSON.parse(schemaText);
      setSyntaxValid(true);
    } catch {
      setSyntaxValid(false);
    }
  }, [schemaText]);

  // 4. Save Schema Mutation
  const saveMutation = useMutation({
    mutationFn: async (jsonContent: string) => {
      const parsed = JSON.parse(jsonContent);
      return apiFetch(`/api/endpoints/${activeEndpointId}/schema`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify(parsed),
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["endpointSchema", activeEndpointId] });
      setStatusMessage({ type: "success", text: "JSON Schema sözleşmesi başarıyla kaydedildi ve aktif edildi!" });
      setTimeout(() => setStatusMessage(null), 4000);
    },
    onError: (err: any) => {
      setStatusMessage({ type: "error", text: err.message || "Geçersiz JSON formatı veya sözleşme kaydedilemedi." });
    },
  });

  // 5. Delete Schema Mutation
  const deleteMutation = useMutation({
    mutationFn: () =>
      apiFetch(`/api/endpoints/${activeEndpointId}/schema`, {
        method: "DELETE",
        token: accessToken,
        organizationId: organization?.id,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["endpointSchema", activeEndpointId] });
      setStatusMessage({ type: "success", text: "Sözleşme kaldırıldı. Bu endpoint artık serbest şemayla çalışıyor." });
      setTimeout(() => setStatusMessage(null), 4000);
    },
  });

  const handlePrettify = () => {
    try {
      const parsed = JSON.parse(schemaText);
      setSchemaText(JSON.stringify(parsed, null, 2));
      setSyntaxValid(true);
    } catch (e: any) {
      setStatusMessage({ type: "error", text: "Prettify yapılamadı. JSON sözdizimi hatalı: " + e.message });
      setTimeout(() => setStatusMessage(null), 3000);
    }
  };

  const handleSave = () => {
    try {
      JSON.parse(schemaText);
      saveMutation.mutate(schemaText);
    } catch (e: any) {
      setStatusMessage({ type: "error", text: "Geçersiz JSON formatı: " + e.message });
    }
  };

  const activeEndpoint = endpoints.find((e) => e.id === activeEndpointId);

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div>
          <div className="flex items-center gap-2.5">
            <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
              <FileCode className="h-6 w-6 text-primary" />
              API & Webhook Sözleşmeleri (Contract Guard)
            </h1>
            <span className="flex items-center gap-1 rounded-full bg-blue-500/10 px-2.5 py-0.5 text-xs font-semibold text-blue-400">
              Draft 2020-12
            </span>
          </div>
          <p className="text-sm text-muted-foreground mt-1">
            Gelen webhook isteklerini JSON Schema ile doğrulayın; eksik veya bozuk alanları anında tespit edin.
          </p>
        </div>

        {/* Project & Endpoint Selectors */}
        <div className="flex items-center gap-3">
          {projects.length > 0 && (
            <select
              value={activeProjectId}
              onChange={(e) => {
                setSelectedProjectId(e.target.value);
                setSelectedEndpointId("");
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

          {endpoints.length > 0 && (
            <select
              value={activeEndpointId}
              onChange={(e) => setSelectedEndpointId(e.target.value)}
              className="rounded-xl border border-border bg-card px-3 py-2 text-xs font-semibold text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            >
              {endpoints.map((ep) => (
                <option key={ep.id} value={ep.id}>
                  {ep.name} (/{ep.slug})
                </option>
              ))}
            </select>
          )}
        </div>
      </div>

      {statusMessage && (
        <div
          className={`flex items-center gap-2 rounded-xl p-4 text-sm font-medium ${
            statusMessage.type === "success"
              ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20"
              : "bg-destructive/10 text-destructive border border-destructive/20"
          }`}
        >
          {statusMessage.type === "success" ? (
            <CheckCircle2 className="h-5 w-5" />
          ) : (
            <AlertTriangle className="h-5 w-5" />
          )}
          <span>{statusMessage.text}</span>
        </div>
      )}

      {/* Contract Editor Card */}
      <div className="rounded-2xl border border-border bg-card p-6 shadow-sm space-y-6">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between border-b border-border pb-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-blue-500/10 text-blue-400 border border-blue-500/20">
              <FileJson className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-base font-bold text-foreground">
                {activeEndpoint ? activeEndpoint.name : "Endpoint Seçin"} Sözleşmesi
              </h3>
              <p className="text-xs text-muted-foreground font-mono">
                {activeEndpoint ? `/hook/${activeEndpoint.slug}` : "Lütfen bir endpoint seçin"}
              </p>
            </div>
          </div>

          {/* Template Presets Bar */}
          <div className="flex flex-wrap items-center gap-2">
            {schemaData?.json_schema && (
              <span className="flex items-center gap-1.5 rounded-full bg-emerald-500/10 text-emerald-400 px-3 py-1 text-xs font-semibold border border-emerald-500/20">
                <CheckCircle2 className="h-3.5 w-3.5" />
                ŞEMA AKTİF
              </span>
            )}

            <button
              onClick={() => setSchemaText(JSON.stringify(PRESET_STRIPE, null, 2))}
              className="rounded-xl border border-border bg-secondary/50 px-2.5 py-1.5 text-xs font-semibold text-foreground hover:bg-secondary transition"
            >
              Stripe
            </button>
            <button
              onClick={() => setSchemaText(JSON.stringify(PRESET_GITHUB, null, 2))}
              className="rounded-xl border border-border bg-secondary/50 px-2.5 py-1.5 text-xs font-semibold text-foreground hover:bg-secondary transition"
            >
              GitHub
            </button>
            <button
              onClick={() => setSchemaText(JSON.stringify(PRESET_GENERIC, null, 2))}
              className="rounded-xl border border-border bg-secondary/50 px-2.5 py-1.5 text-xs font-semibold text-foreground hover:bg-secondary transition"
            >
              Genel REST
            </button>
          </div>
        </div>

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
          {/* JSON Schema Code Editor Area */}
          <div className="lg:col-span-8 space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <h4 className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">
                  JSON Schema Sözleşme Tanımı (Düzenlenebilir)
                </h4>
                {syntaxValid ? (
                  <span className="flex items-center gap-1 text-[10px] text-emerald-400 font-semibold">
                    <CheckCircle2 className="h-3 w-3" /> JSON Geçerli
                  </span>
                ) : (
                  <span className="flex items-center gap-1 text-[10px] text-rose-400 font-semibold">
                    <XCircle className="h-3 w-3" /> Sözdizimi Hatası
                  </span>
                )}
              </div>

              <button
                onClick={handlePrettify}
                className="flex items-center gap-1 rounded-lg bg-secondary px-2.5 py-1 text-xs font-medium text-foreground hover:bg-primary hover:text-primary-foreground transition"
              >
                <AlignLeft className="h-3 w-3" />
                <span>Formatla (Prettify)</span>
              </button>
            </div>

            <textarea
              value={schemaText}
              onChange={(e) => setSchemaText(e.target.value)}
              rows={16}
              spellCheck={false}
              className={`w-full rounded-xl bg-background p-4 text-xs font-mono text-foreground border leading-relaxed focus:outline-none ${
                syntaxValid ? "border-border focus:ring-2 focus:ring-primary" : "border-rose-500/60 focus:ring-2 focus:ring-rose-500"
              }`}
            />

            <div className="flex items-center justify-between pt-2">
              {schemaData?.json_schema && (
                <button
                  onClick={() => deleteMutation.mutate()}
                  disabled={deleteMutation.isPending}
                  className="flex items-center gap-1.5 rounded-xl border border-destructive/30 px-3.5 py-2 text-xs font-semibold text-destructive hover:bg-destructive/10 transition"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                  <span>Sözleşmeyi Kaldır</span>
                </button>
              )}

              <div className="ml-auto">
                <button
                  onClick={handleSave}
                  disabled={saveMutation.isPending || !activeEndpointId || !syntaxValid}
                  className="flex items-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-xs font-bold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
                >
                  {saveMutation.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="h-4 w-4" />
                  )}
                  <span>Sözleşmeyi Kaydet & Uygula</span>
                </button>
              </div>
            </div>
          </div>

          {/* Right Info & Contract Health */}
          <div className="lg:col-span-4 space-y-4">
            <div className="rounded-xl border border-border bg-secondary/20 p-4 space-y-3">
              <h4 className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
                Sözleşme Doğrulama Mantığı
              </h4>
              <div className="text-xs text-muted-foreground space-y-2 leading-relaxed">
                <p>
                  • Gelen webhook JSON gövdesi, kaydedilen bu JSON Schema kurallarına uymak zorundadır.
                </p>
                <p>
                  • Zorunlu alanlar (örneğin <span className="font-mono text-foreground">event, amount, customer</span>) eksik veya yanlış tipte ise istek politika ihlali olarak işaretlenir.
                </p>
                <p>
                  • İhlaller <span className="font-semibold text-primary">Güvenlik Bulguları</span> sayfasında listelenir ve Slack/Discord üzerinden uyarılır.
                </p>
              </div>
            </div>

            <div className="rounded-xl border border-emerald-500/30 bg-emerald-500/5 p-4 text-xs space-y-1.5">
              <div className="flex items-center gap-1.5 font-bold text-emerald-400">
                <ShieldCheck className="h-4 w-4" />
                <span>Gerçek Zamanlı Denetim</span>
              </div>
              <p className="text-muted-foreground leading-relaxed">
                Şema kaydedildiği andan itibaren Go ingestion motoru her gelen webhook'u mikrosaniyeler içinde denetler.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

