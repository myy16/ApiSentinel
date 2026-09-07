"use client";

import React, { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import { Project, Endpoint, SchemaBaseline, SchemaDriftEvent, DriftReport } from "@apisentinel/shared";
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
  UploadCloud,
  Wand2,
  FileUp,
  History,
  Radio,
  CheckCircle,
  Copy,
  ExternalLink,
  ChevronRight,
  Plus,
  GitBranch,
  ArrowRight,
  Eye,
  AlertOctagon,
  RefreshCw,
  Edit3,
  ToggleLeft,
  ToggleRight,
} from "lucide-react";
import { useActiveProject } from "../../../contexts/ProjectContext";
import { useSearchParams } from "next/navigation";

const PRESET_STRIPE = {
  $schema: "https://json-schema.org/draft/2020-12/schema",
  title: "StripeWebhookPayload",
  type: "object",
  properties: {
    id: { type: "string" },
    object: { type: "string" },
    type: { type: "string", enum: ["payment_intent.succeeded", "payment_intent.payment_failed", "charge.refunded"] },
    created: { type: "integer" },
    data: {
      type: "object",
      properties: {
        object: {
          type: "object",
          properties: {
            id: { type: "string" },
            amount: { type: "integer", minimum: 1 },
            currency: { type: "string", minLength: 3, maxLength: 3 },
            status: { type: "string" },
          },
          required: ["id", "amount", "currency", "status"],
        },
      },
      required: ["object"],
    },
  },
  required: ["id", "type", "created", "data"],
};

const PRESET_IYZICO = {
  $schema: "https://json-schema.org/draft/2020-12/schema",
  title: "IyzicoWebhookPayload",
  type: "object",
  properties: {
    iyziEventType: { type: "string" },
    iyziEventTime: { type: "integer" },
    paymentId: { type: "string" },
    status: { type: "string", enum: ["SUCCESS", "FAILURE"] },
    price: { type: "string" },
    paidPrice: { type: "string" },
    currency: { type: "string" },
    conversationId: { type: "string" },
  },
  required: ["iyziEventType", "paymentId", "status", "price", "currency"],
};

const PRESET_GITHUB = {
  $schema: "https://json-schema.org/draft/2020-12/schema",
  title: "GitHubWebhookPayload",
  type: "object",
  properties: {
    ref: { type: "string" },
    repository: {
      type: "object",
      properties: {
        name: { type: "string" },
        full_name: { type: "string" },
      },
      required: ["name"],
    },
    pusher: {
      type: "object",
      properties: {
        name: { type: "string" },
        email: { type: "string", format: "email" },
      },
      required: ["name"],
    },
  },
  required: ["ref", "repository"],
};

function decodeBase64ToUTF8(str: string): string {
  try {
    if (typeof window !== "undefined") {
      const binary = atob(str);
      const bytes = new Uint8Array(binary.length);
      for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i);
      }
      return new TextDecoder("utf-8").decode(bytes);
    }
    return Buffer.from(str, "base64").toString("utf-8");
  } catch {
    return str;
  }
}

function parseJsonOrBase64<T = any>(raw: any): T | null {
  if (!raw) return null;
  if (typeof raw === "object") return raw as T;
  if (typeof raw === "string") {
    try {
      return JSON.parse(raw) as T;
    } catch {
      try {
        const decoded = decodeBase64ToUTF8(raw);
        return JSON.parse(decoded) as T;
      } catch {
        return null;
      }
    }
  }
  return null;
}

function decodeAndFormatSchema(raw: any): string {
  if (!raw) return "";
  const parsed = parseJsonOrBase64(raw);
  if (parsed) {
    return JSON.stringify(parsed, null, 2);
  }
  return typeof raw === "string" ? raw : "";
}

export default function ContractsPage() {
  const queryClient = useQueryClient();
  const searchParams = useSearchParams();
  const endpointParam = searchParams.get("endpointId");

  const { accessToken, organization } = useAuth();
  const { projects, activeProjectId, setActiveProjectId } = useActiveProject();

  const [activeTab, setActiveTab] = useState<"EDITOR" | "DRIFTS">("EDITOR");
  const [selectedEndpointId, setSelectedEndpointId] = useState<string>("");
  const [schemaText, setSchemaText] = useState<string>(JSON.stringify(PRESET_STRIPE, null, 2));
  const [syntaxValid, setSyntaxValid] = useState(true);
  const [statusMessage, setStatusMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);
  const [editingBaseline, setEditingBaseline] = useState<SchemaBaseline | null>(null);

  // Modals state
  const [isInferModalOpen, setIsInferModalOpen] = useState(false);
  const [samplePayloadText, setSamplePayloadText] = useState("");
  const [isOpenAPIModalOpen, setIsOpenAPIModalOpen] = useState(false);
  const [openAPISpecText, setOpenAPISpecText] = useState("");
  const [openAPIOperationPath, setOpenAPIOperationPath] = useState("");

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

  useEffect(() => {
    if (endpointParam && endpoints.some((e) => e.id === endpointParam)) {
      setSelectedEndpointId(endpointParam);
    } else if (!selectedEndpointId && endpoints.length > 0) {
      setSelectedEndpointId(endpoints[0].id);
    }
  }, [endpoints, endpointParam, selectedEndpointId]);

  const activeEndpointId = selectedEndpointId || (endpoints[0]?.id ?? "");

  // 1. Fetch Schema Baselines for active endpoint
  const { data: baselinesData, isLoading: isBaselinesLoading } = useQuery({
    queryKey: ["schemaBaselines", activeEndpointId],
    queryFn: () =>
      apiFetch<{ baselines: SchemaBaseline[] }>(`/api/endpoints/${activeEndpointId}/schemas`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeEndpointId && !!organization?.id,
  });

  const baselines = baselinesData?.baselines || [];
  const activeBaseline = baselines.find((b: any) => Boolean(b.isActive ?? b.is_active));

  // 2. Fetch Schema Drift Events for active endpoint
  const { data: driftsData, isLoading: isDriftsLoading } = useQuery({
    queryKey: ["schemaDrifts", activeEndpointId],
    queryFn: () =>
      apiFetch<{ drifts: SchemaDriftEvent[] }>(`/api/endpoints/${activeEndpointId}/drifts`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeEndpointId && !!organization?.id,
    refetchInterval: 5000,
  });

  const drifts = driftsData?.drifts || [];
  const unacknowledgedDrifts = drifts.filter((d: any) => !Boolean(d.isAcknowledged ?? d.is_acknowledged));

  useEffect(() => {
    if (activeBaseline) {
      const rawSchema = (activeBaseline as any).schemaJson ?? (activeBaseline as any).schema_json;
      const formatted = decodeAndFormatSchema(rawSchema);
      setSchemaText(formatted);
      setSyntaxValid(true);
    }
  }, [activeBaseline]);

  const handleSchemaChange = (text: string) => {
    setSchemaText(text);
    try {
      JSON.parse(text);
      setSyntaxValid(true);
    } catch {
      setSyntaxValid(false);
    }
  };

  // 1. Save Manual Schema Mutation
  const saveManualMutation = useMutation({
    mutationFn: (jsonString: string) =>
      apiFetch<SchemaBaseline>(`/api/endpoints/${activeEndpointId}/schemas`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify({ schemaJson: jsonString, activate: true }),
      }),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["schemaBaselines", activeEndpointId] });
      setStatusMessage({ type: "success", text: `Sözleşme v${data.version} başarıyla kaydedildi ve aktif baseline yapıldı.` });
      setTimeout(() => setStatusMessage(null), 4000);
    },
    onError: (err: any) => {
      setStatusMessage({ type: "error", text: err.message || "Sözleşme kaydedilemedi." });
    },
  });

  // 2. Infer Schema Mutation
  const inferMutation = useMutation({
    mutationFn: (payload: string) =>
      apiFetch<SchemaBaseline>(`/api/endpoints/${activeEndpointId}/schemas/infer`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify({ payload, activate: true }),
      }),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["schemaBaselines", activeEndpointId] });
      setIsInferModalOpen(false);
      setSamplePayloadText("");
      setStatusMessage({ type: "success", text: `Örnek payload'dan v${data.version} şeması otomatik türetildi ve aktifleştirildi.` });
      setTimeout(() => setStatusMessage(null), 4000);
    },
    onError: (err: any) => {
      setStatusMessage({ type: "error", text: err.message || "Şema türetilemedi." });
    },
  });

  // 3. Import OpenAPI Mutation
  const openAPIMutation = useMutation({
    mutationFn: (input: { spec: string; operationPath: string }) =>
      apiFetch<SchemaBaseline>(`/api/endpoints/${activeEndpointId}/schemas/openapi`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify({ spec: input.spec, operationPath: input.operationPath, activate: true }),
      }),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["schemaBaselines", activeEndpointId] });
      setIsOpenAPIModalOpen(false);
      setOpenAPISpecText("");
      setStatusMessage({ type: "success", text: `OpenAPI spesifikasyonundan v${data.version} şeması başarıyla çıkarıldı.` });
      setTimeout(() => setStatusMessage(null), 4000);
    },
    onError: (err: any) => {
      setStatusMessage({ type: "error", text: err.message || "OpenAPI şeması çıkarılamadı." });
    },
  });

  // 4. Activate Version Mutation with Instant Optimistic UI Update
  const activateMutation = useMutation({
    mutationFn: (schemaId: string) =>
      apiFetch<SchemaBaseline>(`/api/endpoints/${activeEndpointId}/schemas/${schemaId}/activate`, {
        method: "PUT",
        token: accessToken,
        organizationId: organization?.id,
      }),
    onMutate: async (schemaId: string) => {
      await queryClient.cancelQueries({ queryKey: ["schemaBaselines", activeEndpointId] });
      const previousData = queryClient.getQueryData<{ baselines: any[] }>(["schemaBaselines", activeEndpointId]);
      if (previousData) {
        queryClient.setQueryData(["schemaBaselines", activeEndpointId], {
          baselines: previousData.baselines.map((b) => ({
            ...b,
            isActive: b.id === schemaId,
            is_active: b.id === schemaId,
          })),
        });
      }
      return { previousData };
    },
    onSuccess: (data, schemaId) => {
      queryClient.setQueryData(["schemaBaselines", activeEndpointId], (old: any) => {
        if (!old?.baselines) return old;
        return {
          ...old,
          baselines: old.baselines.map((b: any) => ({
            ...b,
            isActive: b.id === schemaId,
            is_active: b.id === schemaId,
          })),
        };
      });
      queryClient.invalidateQueries({ queryKey: ["schemaBaselines", activeEndpointId] });
      setStatusMessage({ type: "success", text: `Sözleşme v${data.version} aktif baseline olarak ayarlandı.` });
      setTimeout(() => setStatusMessage(null), 3000);
    },
    onError: (err: any, _vars, context: any) => {
      if (context?.previousData) {
        queryClient.setQueryData(["schemaBaselines", activeEndpointId], context.previousData);
      }
      setStatusMessage({ type: "error", text: err.message || "Aktifleştirilemedi." });
    },
  });

  // 4a2. Deactivate Baseline Mutation
  const deactivateMutation = useMutation({
    mutationFn: (schemaId: string) =>
      apiFetch<{ message: string }>(`/api/endpoints/${activeEndpointId}/schemas/${schemaId}/deactivate`, {
        method: "PUT",
        token: accessToken,
        organizationId: organization?.id,
      }),
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: ["schemaBaselines", activeEndpointId] });
      const previousData = queryClient.getQueryData<{ baselines: any[] }>(["schemaBaselines", activeEndpointId]);
      if (previousData) {
        queryClient.setQueryData(["schemaBaselines", activeEndpointId], {
          baselines: previousData.baselines.map((b) => ({
            ...b,
            isActive: false,
            is_active: false,
          })),
        });
      }
      return { previousData };
    },
    onSuccess: (data) => {
      queryClient.setQueryData(["schemaBaselines", activeEndpointId], (old: any) => {
        if (!old?.baselines) return old;
        return {
          ...old,
          baselines: old.baselines.map((b: any) => ({
            ...b,
            isActive: false,
            is_active: false,
          })),
        };
      });
      queryClient.invalidateQueries({ queryKey: ["schemaBaselines", activeEndpointId] });
      setStatusMessage({ type: "success", text: data.message || "Sözleşme pasife alındı." });
      setTimeout(() => setStatusMessage(null), 3000);
    },
    onError: (err: any, _vars, context: any) => {
      if (context?.previousData) {
        queryClient.setQueryData(["schemaBaselines", activeEndpointId], context.previousData);
      }
      setStatusMessage({ type: "error", text: err.message || "Pasife alınamadı." });
    },
  });

  // 4b. Update Existing Baseline Mutation
  const updateMutation = useMutation({
    mutationFn: (input: { schemaId: string; jsonString: string }) =>
      apiFetch<SchemaBaseline>(`/api/endpoints/${activeEndpointId}/schemas/${input.schemaId}`, {
        method: "PUT",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify({ schemaJson: input.jsonString }),
      }),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["schemaBaselines", activeEndpointId] });
      setStatusMessage({ type: "success", text: `Sözleşme v${data.version} içeriği başarıyla güncellendi.` });
      setTimeout(() => setStatusMessage(null), 4000);
    },
    onError: (err: any) => {
      setStatusMessage({ type: "error", text: err.message || "Sözleşme güncellenemedi." });
    },
  });

  // 4c. Delete Baseline Mutation
  const deleteMutation = useMutation({
    mutationFn: (schemaId: string) =>
      apiFetch(`/api/endpoints/${activeEndpointId}/schemas/${schemaId}`, {
        method: "DELETE",
        token: accessToken,
        organizationId: organization?.id,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["schemaBaselines", activeEndpointId] });
      if (editingBaseline) setEditingBaseline(null);
      setStatusMessage({ type: "success", text: "Sözleşme sürümü silindi." });
      setTimeout(() => setStatusMessage(null), 3000);
    },
    onError: (err: any) => {
      setStatusMessage({ type: "error", text: err.message || "Sözleşme silinemedi." });
    },
  });

  // 5. Accept Drift Mutation
  const acceptDriftMutation = useMutation({
    mutationFn: (driftId: string) =>
      apiFetch<{ message: string; baseline: SchemaBaseline }>(`/api/endpoints/${activeEndpointId}/drifts/${driftId}/accept`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
      }),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["schemaBaselines", activeEndpointId] });
      queryClient.invalidateQueries({ queryKey: ["schemaDrifts", activeEndpointId] });
      setStatusMessage({ type: "success", text: data.message });
      setTimeout(() => setStatusMessage(null), 4000);
    },
    onError: (err: any) => {
      setStatusMessage({ type: "error", text: err.message || "Sapma kabul edilemedi." });
    },
  });

  // 6. Dismiss Drift Mutation
  const dismissDriftMutation = useMutation({
    mutationFn: (driftId: string) =>
      apiFetch(`/api/endpoints/${activeEndpointId}/drifts/${driftId}/dismiss`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["schemaDrifts", activeEndpointId] });
      setStatusMessage({ type: "success", text: "Şema sapması uyarısı kapatıldı." });
      setTimeout(() => setStatusMessage(null), 3000);
    },
    onError: (err: any) => {
      setStatusMessage({ type: "error", text: err.message || "İşlem başarısız." });
    },
  });

  return (
    <div className="space-y-6 animate-in fade-in duration-300">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div className="flex items-center gap-2">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-blue-500/10 text-blue-400 border border-blue-500/20">
              <FileCode className="h-5 w-5" />
            </div>
            <h1 className="text-xl font-bold tracking-tight text-foreground">
              Şema Sözleşmeleri & Drift Tespiti (Contracts)
            </h1>
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            Webhook JSON yapılarını versiyonlayın, OpenAPI baseline'ları yönetin ve canlı trafik sapmalarını tespit edin
          </p>
        </div>

        {/* Action Buttons */}
        <div className="flex flex-wrap items-center gap-2">
          <button
            onClick={() => setIsInferModalOpen(true)}
            disabled={!activeEndpointId}
            className="flex items-center gap-1.5 px-3 py-2 rounded-xl bg-purple-500/10 hover:bg-purple-500/20 text-purple-400 border border-purple-500/20 text-xs font-bold transition disabled:opacity-50"
          >
            <Wand2 className="h-3.5 w-3.5" />
            <span>Payload'dan Türet</span>
          </button>

          <button
            onClick={() => setIsOpenAPIModalOpen(true)}
            disabled={!activeEndpointId}
            className="flex items-center gap-1.5 px-3 py-2 rounded-xl bg-blue-500/10 hover:bg-blue-500/20 text-blue-400 border border-blue-500/20 text-xs font-bold transition disabled:opacity-50"
          >
            <FileUp className="h-3.5 w-3.5" />
            <span>OpenAPI 3.0 İçe Aktar</span>
          </button>
        </div>
      </div>

      {/* Endpoint Selector Bar & Tab Switcher */}
      <div className="flex flex-wrap items-center justify-between gap-3 p-3.5 rounded-2xl border border-border bg-card">
        <div className="flex items-center gap-3">
          <Layers className="h-4 w-4 text-muted-foreground" />
          <span className="text-xs font-bold text-foreground">Hedef Endpoint:</span>
          {endpoints.length === 0 ? (
            <span className="text-xs text-muted-foreground italic">Henüz tanımlı endpoint yok.</span>
          ) : (
            <select
              value={activeEndpointId}
              onChange={(e) => setSelectedEndpointId(e.target.value)}
              className="rounded-xl border border-border bg-background px-3 py-1.5 text-xs font-semibold text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            >
              {endpoints.map((ep) => (
                <option key={ep.id} value={ep.id}>
                  {ep.name} (/hook/{ep.slug})
                </option>
              ))}
            </select>
          )}
        </div>

        {/* View Tabs */}
        <div className="flex items-center gap-1 bg-muted/40 p-1 rounded-xl">
          <button
            onClick={() => setActiveTab("EDITOR")}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-bold transition ${
              activeTab === "EDITOR"
                ? "bg-card text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            <Code className="h-3.5 w-3.5" />
            <span>Sözleşme & Sürümler</span>
          </button>
          <button
            onClick={() => setActiveTab("DRIFTS")}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-bold transition ${
              activeTab === "DRIFTS"
                ? "bg-card text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            <GitBranch className="h-3.5 w-3.5" />
            <span>Şema Sapmaları (Drift)</span>
            {unacknowledgedDrifts.length > 0 && (
              <span className="px-1.5 py-0.2 rounded-full bg-amber-500 text-black text-[10px] font-extrabold">
                {unacknowledgedDrifts.length}
              </span>
            )}
          </button>
        </div>
      </div>

      {/* Status Message */}
      {statusMessage && (
        <div
          className={`flex items-center gap-2 rounded-xl p-3 text-xs font-medium border ${
            statusMessage.type === "success"
              ? "bg-emerald-500/10 border-emerald-500/30 text-emerald-400"
              : "bg-destructive/10 border-destructive/30 text-destructive"
          }`}
        >
          {statusMessage.type === "success" ? (
            <CheckCircle2 className="h-4 w-4 shrink-0" />
          ) : (
            <AlertTriangle className="h-4 w-4 shrink-0" />
          )}
          <span>{statusMessage.text}</span>
        </div>
      )}

      {/* TAB 1: Schema Editor & Sürüm Geçmişi */}
      {activeTab === "EDITOR" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-12 animate-in fade-in duration-200">
          {/* Left: JSON Schema Editor */}
          <div className="lg:col-span-7 space-y-4">
            <div className="rounded-2xl border border-border bg-card p-5 space-y-4 shadow-sm">
              {/* Code Editor Header with Editing State & Presets */}
              <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border pb-3">
                <div className="flex items-center gap-2">
                  <Code className="h-4 w-4 text-primary" />
                  <span className="text-xs font-bold text-foreground">
                    {editingBaseline ? `Sözleşme v${editingBaseline.version} Düzenleniyor` : "JSON Schema (Draft 2020-12)"}
                  </span>
                  {editingBaseline && (
                    <button
                      type="button"
                      onClick={() => setEditingBaseline(null)}
                      className="px-2 py-0.5 rounded bg-muted hover:bg-muted/80 text-[10px] font-bold text-muted-foreground transition"
                    >
                      İptal Et
                    </button>
                  )}
                </div>

                {/* Presets */}
                <div className="flex items-center gap-1.5">
                  <span className="text-[10px] uppercase font-bold text-muted-foreground mr-1">Şablon:</span>
                  <button
                    type="button"
                    onClick={() => {
                      setEditingBaseline(null);
                      handleSchemaChange(JSON.stringify(PRESET_STRIPE, null, 2));
                    }}
                    className="px-2 py-0.5 rounded-md bg-secondary hover:bg-muted text-[11px] font-semibold text-foreground transition"
                  >
                    Stripe
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setEditingBaseline(null);
                      handleSchemaChange(JSON.stringify(PRESET_IYZICO, null, 2));
                    }}
                    className="px-2 py-0.5 rounded-md bg-secondary hover:bg-muted text-[11px] font-semibold text-foreground transition"
                  >
                    iyzico
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setEditingBaseline(null);
                      handleSchemaChange(JSON.stringify(PRESET_GITHUB, null, 2));
                    }}
                    className="px-2 py-0.5 rounded-md bg-secondary hover:bg-muted text-[11px] font-semibold text-foreground transition"
                  >
                    GitHub
                  </button>
                </div>
              </div>

              {/* Code Editor Area */}
              <div className="relative">
                <textarea
                  value={schemaText}
                  onChange={(e) => handleSchemaChange(e.target.value)}
                  rows={18}
                  spellCheck={false}
                  placeholder="JSON Schema definition..."
                  className="w-full rounded-xl border border-input bg-background/70 p-4 font-mono text-xs leading-relaxed text-foreground placeholder:text-muted-foreground/60 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary select-text"
                />
                <div className="absolute right-3 bottom-3">
                  {syntaxValid ? (
                    <span className="flex items-center gap-1 rounded-md bg-emerald-500/10 px-2 py-1 text-[10px] font-bold text-emerald-400 border border-emerald-500/20">
                      <CheckCircle2 className="h-3 w-3" />
                      <span>Geçerli JSON</span>
                    </span>
                  ) : (
                    <span className="flex items-center gap-1 rounded-md bg-destructive/10 px-2 py-1 text-[10px] font-bold text-destructive border border-destructive/20">
                      <XCircle className="h-3 w-3" />
                      <span>Sözdizimi Hatası</span>
                    </span>
                  )}
                </div>
              </div>

              {/* Save & Update Buttons */}
              <div className="flex items-center justify-end gap-2 pt-2">
                {editingBaseline && (
                  <button
                    type="button"
                    onClick={() =>
                      updateMutation.mutate({
                        schemaId: editingBaseline.id,
                        jsonString: schemaText,
                      })
                    }
                    disabled={!syntaxValid || updateMutation.isPending}
                    className="flex items-center gap-2 rounded-xl bg-emerald-600 px-4 py-2.5 text-xs font-bold text-white shadow-sm transition hover:bg-emerald-700 disabled:opacity-50"
                  >
                    {updateMutation.isPending ? (
                      <>
                        <Loader2 className="h-4 w-4 animate-spin" />
                        <span>Güncelleniyor...</span>
                      </>
                    ) : (
                      <>
                        <Check className="h-4 w-4" />
                        <span>v{editingBaseline.version} Sürümünü Güncelle</span>
                      </>
                    )}
                  </button>
                )}

                <button
                  type="button"
                  onClick={() => saveManualMutation.mutate(schemaText)}
                  disabled={!syntaxValid || saveManualMutation.isPending || !activeEndpointId}
                  className="flex items-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-xs font-bold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
                >
                  {saveManualMutation.isPending ? (
                    <>
                      <Loader2 className="h-4 w-4 animate-spin" />
                      <span>Kaydediliyor...</span>
                    </>
                  ) : (
                    <>
                      <Save className="h-4 w-4" />
                      <span>Yeni Sürüm Olarak Kaydet & Aktifleştir</span>
                    </>
                  )}
                </button>
              </div>
            </div>
          </div>

          {/* Right: Version History & Active Baseline (5 cols) */}
          <div className="lg:col-span-5 space-y-4">
            <div className="rounded-2xl border border-border bg-card p-5 space-y-4 shadow-sm">
              <div className="flex items-center justify-between border-b border-border pb-3">
                <div className="flex items-center gap-2">
                  <History className="h-4 w-4 text-primary" />
                  <span className="text-xs font-bold text-foreground">Sözleşme Sürüm Geçmişi</span>
                </div>
                <span className="text-[11px] text-muted-foreground">{baselines.length} sürüm</span>
              </div>

              {isBaselinesLoading ? (
                <div className="py-8 text-center text-muted-foreground flex flex-col items-center gap-2">
                  <Loader2 className="h-5 w-5 animate-spin text-primary" />
                  <span className="text-xs">Sürümler yükleniyor...</span>
                </div>
              ) : baselines.length === 0 ? (
                <div className="py-8 text-center text-muted-foreground space-y-2">
                  <FileCode className="h-8 w-8 mx-auto opacity-30 text-muted-foreground" />
                  <p className="text-xs font-semibold">Henüz kayıtlı şema baseline'ı yok.</p>
                  <p className="text-[11px] text-muted-foreground max-w-xs mx-auto">
                    Soldaki editörden kaydedebilir, örnek bir payload yapıştırarak türetebilir veya OpenAPI dosyanızı yükleyebilirsiniz.
                  </p>
                </div>
              ) : (
                <div className="space-y-2.5">
                  {baselines.map((b: any) => {
                    const isActive = Boolean(b.isActive ?? b.is_active);
                    return (
                      <div
                        key={b.id}
                        className={`p-3.5 rounded-xl border transition flex items-center justify-between gap-2 ${
                          isActive
                            ? "border-emerald-500/40 bg-emerald-500/5 ring-1 ring-emerald-500/30"
                            : "border-border bg-card/60 hover:bg-muted/30"
                        }`}
                      >
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2 flex-wrap">
                            <span className="text-xs font-bold text-foreground">Sürüm v{b.version}</span>
                            <span
                              className={`text-[10px] font-bold px-1.5 py-0.5 rounded ${
                                b.source === "OPENAPI"
                                  ? "bg-blue-500/10 text-blue-400 border border-blue-500/30"
                                  : b.source === "INFERRED_PAYLOAD"
                                  ? "bg-purple-500/10 text-purple-400 border border-purple-500/30"
                                  : "bg-muted text-muted-foreground"
                              }`}
                            >
                              {b.source}
                            </span>
                            {isActive ? (
                              <span className="text-[10px] font-bold text-emerald-400 bg-emerald-500/20 px-1.5 py-0.5 rounded border border-emerald-500/30">
                                AKTİF
                              </span>
                            ) : (
                              <span className="text-[10px] font-bold text-rose-400 bg-rose-500/10 px-1.5 py-0.5 rounded border border-rose-500/20">
                                PASİF
                              </span>
                            )}
                          </div>
                          <div className="text-[11px] text-muted-foreground mt-1">
                            {new Date(b.createdAt || b.created_at).toLocaleString()}
                          </div>
                        </div>

                        <div className="flex items-center gap-1.5 shrink-0">
                          {/* Edit Button */}
                          <button
                            type="button"
                            title="Bu sürümü editöre yükle ve düzenle"
                            onClick={() => {
                              setEditingBaseline(b);
                              const rawSchema = b.schemaJson ?? b.schema_json;
                              const formatted = decodeAndFormatSchema(rawSchema);
                              setSchemaText(formatted);
                              setSyntaxValid(true);
                            }}
                            className={`p-1.5 rounded-lg border text-xs font-semibold transition flex items-center gap-1 ${
                              editingBaseline?.id === b.id
                                ? "bg-primary text-primary-foreground border-primary"
                                : "border-border bg-secondary hover:bg-muted text-foreground"
                            }`}
                          >
                            <Edit3 className="h-3.5 w-3.5" />
                            <span className="hidden sm:inline text-[11px]">Düzenle</span>
                          </button>

                          {/* Active/Inactive Toggle Switch */}
                          <button
                            type="button"
                            title={
                              isActive
                                ? "Bu sözleşmeyi pasife almak (kapatmak) için tıklayın"
                                : "Bu sözleşmeyi aktif baseline yapmak için tıklayın"
                            }
                            disabled={activateMutation.isPending || deactivateMutation.isPending}
                            onClick={() => {
                              if (isActive) {
                                deactivateMutation.mutate(b.id);
                              } else {
                                activateMutation.mutate(b.id);
                              }
                            }}
                            className={`p-1 rounded-lg transition-all flex items-center cursor-pointer hover:scale-110 ${
                              isActive
                                ? "text-emerald-400 hover:text-rose-400 drop-shadow-[0_0_8px_rgba(52,211,153,0.35)]"
                                : "text-rose-400/80 hover:text-emerald-400"
                            }`}
                          >
                            {isActive ? (
                              <ToggleRight className="h-6 w-6 text-emerald-400 fill-emerald-500/20" />
                            ) : (
                              <ToggleLeft className="h-6 w-6 text-rose-400/80 hover:text-emerald-400" />
                            )}
                          </button>

                          {/* Delete Button */}
                          <button
                            type="button"
                            title="Bu sürümü sil"
                            disabled={deleteMutation.isPending}
                            onClick={() => {
                              if (window.confirm(`v${b.version} sözleşme sürümünü silmek istediğinize emin misiniz?`)) {
                                deleteMutation.mutate(b.id);
                              }
                            }}
                            className="p-1.5 rounded-lg text-muted-foreground hover:text-rose-400 hover:bg-rose-500/10 transition"
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* TAB 2: Şema Sapmaları (Drift Events) */}
      {activeTab === "DRIFTS" && (
        <div className="space-y-4 animate-in fade-in duration-200">
          <div className="rounded-2xl border border-border bg-card p-5 space-y-4 shadow-sm">
            <div className="flex items-center justify-between border-b border-border pb-3">
              <div className="flex items-center gap-2">
                <GitBranch className="h-4 w-4 text-primary" />
                <h2 className="text-sm font-bold text-foreground">Canlı Webhook Trafiğinde Tespit Edilen Şema Sapmaları</h2>
              </div>
              <span className="text-xs text-muted-foreground">{drifts.length} sapma kaydı</span>
            </div>

            {isDriftsLoading ? (
              <div className="py-12 text-center text-muted-foreground flex flex-col items-center gap-2">
                <Loader2 className="h-6 w-6 animate-spin text-primary" />
                <span className="text-xs">Sapma kayıtları taranıyor...</span>
              </div>
            ) : drifts.length === 0 ? (
              <div className="py-12 text-center text-muted-foreground space-y-2">
                <CheckCircle2 className="h-10 w-10 mx-auto text-emerald-400 opacity-60" />
                <p className="text-xs font-bold text-foreground">Şema sapması bulunamadı!</p>
                <p className="text-xs text-muted-foreground max-w-sm mx-auto">
                  Gelen tüm canlı webhook'lar aktif baseline sözleşmesiyle tam uyumlu çalışmaktadır.
                </p>
              </div>
            ) : (
              <div className="space-y-4">
                {drifts.map((d: any) => {
                  const rawDiff = d.diffJson ?? d.diff_json;
                  const report: DriftReport =
                    parseJsonOrBase64<DriftReport>(rawDiff) || { hasDrift: false, severity: "NONE", changes: [], summary: "Şema sapması" };

                  const driftType = d.driftType || d.drift_type || report.severity || "NON_BREAKING";
                  const isBreaking = driftType === "BREAKING";
                  const isAcknowledged = Boolean(d.isAcknowledged ?? d.is_acknowledged);
                  const monotonicId = d.monotonicRequestId || d.monotonic_request_id || d.request_id || (d.requestId ? String(d.requestId).slice(0, 8) : "webhook");
                  const rawDate = d.createdAt || d.created_at;
                  const formattedTime = rawDate && !isNaN(new Date(rawDate).getTime()) ? new Date(rawDate).toLocaleTimeString() : "";

                  return (
                    <div
                      key={d.id}
                      className={`rounded-2xl border p-4 space-y-3 transition ${
                        isAcknowledged
                          ? "border-border bg-card/40 opacity-70"
                          : isBreaking
                          ? "border-rose-500/40 bg-rose-500/5 ring-1 ring-rose-500/20"
                          : "border-amber-500/40 bg-amber-500/5 ring-1 ring-amber-500/20"
                      }`}
                    >
                      {/* Header */}
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <div className="flex items-center gap-2">
                          <span
                            className={`px-2 py-0.5 rounded-full text-[10px] font-extrabold border ${
                              isBreaking
                                ? "bg-rose-500/20 text-rose-300 border-rose-500/40"
                                : "bg-amber-500/20 text-amber-300 border-amber-500/40"
                            }`}
                          >
                            {isBreaking ? "🔴 KIRICI DEĞİŞİKLİK (BREAKING)" : "🟡 GERİYE UYUMLU EKLEME"}
                          </span>
                          <span className="text-xs font-mono text-muted-foreground">
                            İstek: #{monotonicId}
                          </span>
                        </div>

                        <div className="flex items-center gap-2">
                          {formattedTime && (
                            <span className="text-[11px] text-muted-foreground">
                              {formattedTime}
                            </span>
                          )}

                          {!isAcknowledged && (
                            <>
                              <button
                                onClick={() => dismissDriftMutation.mutate(d.id)}
                                disabled={dismissDriftMutation.isPending}
                                className="px-2.5 py-1 rounded-lg bg-secondary hover:bg-muted text-xs font-semibold text-muted-foreground hover:text-foreground transition disabled:opacity-50"
                              >
                                Yok Say
                              </button>
                              <button
                                onClick={() => acceptDriftMutation.mutate(d.id)}
                                disabled={acceptDriftMutation.isPending}
                                className="flex items-center gap-1 px-3 py-1 rounded-lg bg-primary hover:bg-primary/90 text-primary-foreground text-xs font-bold transition disabled:opacity-50"
                              >
                                {acceptDriftMutation.isPending ? (
                                  <Loader2 className="h-3 w-3 animate-spin" />
                                ) : (
                                  <Sparkles className="h-3 w-3" />
                                )}
                                <span>Değişiklikleri Kabul Et & Güncelle</span>
                              </button>
                            </>
                          )}
                        </div>
                      </div>

                      {/* Summary */}
                      <p className="text-xs text-foreground font-medium">{report.summary}</p>

                      {/* Diff Items Table */}
                      {report.changes && report.changes.length > 0 && (
                        <div className="rounded-xl border border-border bg-card/80 overflow-hidden text-xs">
                          <table className="w-full text-left">
                            <thead className="bg-muted/40 border-b border-border text-[10px] uppercase font-bold text-muted-foreground">
                              <tr>
                                <th className="px-3 py-2">Alan / Path</th>
                                <th className="px-3 py-2">Sapma Türü</th>
                                <th className="px-3 py-2">Açıklama</th>
                              </tr>
                            </thead>
                            <tbody className="divide-y divide-border">
                              {report.changes.map((ch, idx) => (
                                <tr key={idx} className="hover:bg-muted/20">
                                  <td className="px-3 py-2 font-mono text-primary font-semibold">{ch.path}</td>
                                  <td className="px-3 py-2">
                                    <span
                                      className={`px-1.5 py-0.5 rounded text-[10px] font-bold ${
                                        ch.changeType === "FIELD_ADDED"
                                          ? "bg-emerald-500/10 text-emerald-400"
                                          : ch.changeType === "FIELD_MISSING"
                                          ? "bg-rose-500/10 text-rose-400"
                                          : "bg-amber-500/10 text-amber-400"
                                      }`}
                                    >
                                      {ch.changeType}
                                    </span>
                                  </td>
                                  <td className="px-3 py-2 text-muted-foreground">{ch.description}</td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Infer from Payload Modal */}
      {isInferModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-in fade-in duration-200">
          <div className="w-full max-w-xl rounded-2xl border border-border bg-card p-6 shadow-xl space-y-4">
            <div className="flex items-center justify-between border-b border-border pb-3">
              <div className="flex items-center gap-2">
                <Wand2 className="h-5 w-5 text-purple-400" />
                <h3 className="text-sm font-bold text-foreground">Örnek Payload'dan Şema Türet</h3>
              </div>
              <button
                onClick={() => setIsInferModalOpen(false)}
                className="text-xs text-muted-foreground hover:text-foreground"
              >
                Kapat
              </button>
            </div>

            <p className="text-xs text-muted-foreground">
              Herhangi bir geçerli webhook JSON payload'unu yapıştırın. ApiSentinel veri tiplerini, string formatlarını (UUID, email, date-time, uri) ve zorunlu alanları otomatik çıkararak JSON Schema baseline'ı oluşturur.
            </p>

            <textarea
              value={samplePayloadText}
              onChange={(e) => setSamplePayloadText(e.target.value)}
              rows={10}
              placeholder={`{\n  "event_id": "evt_12345",\n  "amount": 2500,\n  "customer": {\n    "email": "user@example.com"\n  }\n}`}
              className="w-full rounded-xl border border-input bg-background/60 p-3 font-mono text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
            />

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setIsInferModalOpen(false)}
                className="px-4 py-2 rounded-xl text-xs font-semibold text-muted-foreground hover:text-foreground"
              >
                Vazgeç
              </button>
              <button
                type="button"
                onClick={() => inferMutation.mutate(samplePayloadText)}
                disabled={!samplePayloadText.trim() || inferMutation.isPending}
                className="flex items-center gap-1.5 px-4 py-2 rounded-xl bg-purple-600 hover:bg-purple-500 text-white text-xs font-bold transition disabled:opacity-50"
              >
                {inferMutation.isPending ? (
                  <>
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    <span>Türetiliyor...</span>
                  </>
                ) : (
                  <>
                    <Wand2 className="h-3.5 w-3.5" />
                    <span>Şemayı Türet & Aktifleştir</span>
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* OpenAPI Import Modal */}
      {isOpenAPIModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4 animate-in fade-in duration-200">
          <div className="w-full max-w-2xl rounded-2xl border border-border bg-card p-6 shadow-xl space-y-4">
            <div className="flex items-center justify-between border-b border-border pb-3">
              <div className="flex items-center gap-2">
                <FileUp className="h-5 w-5 text-blue-400" />
                <h3 className="text-sm font-bold text-foreground">OpenAPI 3.0 / 3.1 İçe Aktar</h3>
              </div>
              <button
                onClick={() => setIsOpenAPIModalOpen(false)}
                className="text-xs text-muted-foreground hover:text-foreground"
              >
                Kapat
              </button>
            </div>

            <p className="text-xs text-muted-foreground">
              OpenAPI 3.0 veya 3.1 spesifikasyonunuzun (JSON veya YAML) içeriğini yapıştırın. Webhook veya istek şeması otomatik çıkarılacaktır.
            </p>

            <div>
              <label className="block text-[10px] font-bold uppercase tracking-wider text-muted-foreground mb-1">
                Hedef Rota / Path (Opsiyonel):
              </label>
              <input
                type="text"
                value={openAPIOperationPath}
                onChange={(e) => setOpenAPIOperationPath(e.target.value)}
                placeholder="Örn: /webhooks/stripe veya /api/v1/payments (Boş bırakılırsa ilk bulunan şema alınır)"
                className="w-full rounded-xl border border-input bg-background/60 px-3 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>

            <textarea
              value={openAPISpecText}
              onChange={(e) => setOpenAPISpecText(e.target.value)}
              rows={10}
              placeholder="openapi: 3.0.3\ninfo:\n  title: Webhook API\npaths:\n  /webhook: ..."
              className="w-full rounded-xl border border-input bg-background/60 p-3 font-mono text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
            />

            <div className="flex justify-end gap-2 pt-2">
              <button
                type="button"
                onClick={() => setIsOpenAPIModalOpen(false)}
                className="px-4 py-2 rounded-xl text-xs font-semibold text-muted-foreground hover:text-foreground"
              >
                Vazgeç
              </button>
              <button
                type="button"
                onClick={() => openAPIMutation.mutate({ spec: openAPISpecText, operationPath: openAPIOperationPath })}
                disabled={!openAPISpecText.trim() || openAPIMutation.isPending}
                className="flex items-center gap-1.5 px-4 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white text-xs font-bold transition disabled:opacity-50"
              >
                {openAPIMutation.isPending ? (
                  <>
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    <span>İçe Aktarılıyor...</span>
                  </>
                ) : (
                  <>
                    <UploadCloud className="h-3.5 w-3.5" />
                    <span>Şemayı Çıkar & Aktifleştir</span>
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
