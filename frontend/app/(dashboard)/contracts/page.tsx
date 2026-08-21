"use client";

import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
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
  Plus,
  Zap,
} from "lucide-react";

export default function ContractsPage() {
  const { accessToken, organization } = useAuth();

  const [selectedProjectId, setSelectedProjectId] = useState<string>("");
  const [selectedEndpointId, setSelectedEndpointId] = useState<string>("");

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

  // Fetch endpoints
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

  const sampleSchema = {
    $schema: "https://json-schema.org/draft/2020-12/schema",
    title: "PaymentWebhookPayload",
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

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-bold tracking-tight">API & Webhook Sözleşmeleri (Contract Testing)</h1>
            <span className="flex items-center gap-1 rounded-full bg-blue-500/10 px-2.5 py-0.5 text-xs font-semibold text-blue-400">
              <FileCode className="h-3 w-3" />
              JSON Schema Draft 2020-12
            </span>
          </div>
          <p className="text-sm text-muted-foreground">
            Entegrasyon sağlayan servislerin (Stripe, GitHub, ERP) payload şemalarını denetleyin ve breaking change'leri önleyin
          </p>
        </div>

        <div className="flex items-center gap-3">
          {endpoints.length > 0 && (
            <select
              value={activeEndpointId}
              onChange={(e) => setSelectedEndpointId(e.target.value)}
              className="rounded-lg border border-input bg-card px-3 py-2 text-sm font-medium focus:border-primary focus:outline-none"
            >
              {endpoints.map((ep) => (
                <option key={ep.id} value={ep.id}>
                  {ep.name} (/hook/{ep.slug})
                </option>
              ))}
            </select>
          )}

          <button
            disabled={!activeEndpointId}
            className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
          >
            <Plus className="h-4 w-4" />
            <span>Yeni Şema Tanımla</span>
          </button>
        </div>
      </div>

      {/* Contract Details Card */}
      <div className="rounded-xl border border-border bg-card p-6 shadow-sm space-y-6">
        <div className="flex items-center justify-between border-b border-border pb-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10 text-blue-400">
              <FileCode className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-base font-bold text-foreground">PaymentWebhookPayload (v1.2)</h3>
              <p className="text-xs text-muted-foreground">Aktif Endpoint: Stripe Production Hook</p>
            </div>
          </div>

          <span className="flex items-center gap-1.5 rounded-full bg-emerald-500/10 text-emerald-400 px-3 py-1 text-xs font-semibold">
            <CheckCircle2 className="h-3.5 w-3.5" />
            ŞEMA AKTİF & DENETLENİYOR
          </span>
        </div>

        <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
          <div className="lg:col-span-8 space-y-2">
            <h4 className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
              JSON Schema Sözleşme Tanımı
            </h4>
            <pre className="rounded-lg bg-background p-4 text-xs font-mono text-foreground border border-border overflow-x-auto max-h-[380px]">
              {JSON.stringify(sampleSchema, null, 2)}
            </pre>
          </div>

          <div className="lg:col-span-4 space-y-4">
            <div className="rounded-lg border border-border bg-secondary/20 p-4 space-y-3">
              <h4 className="text-xs font-bold uppercase tracking-wider text-muted-foreground">
                Sözleşme Sağlığı
              </h4>
              <div className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground">Doğrulanan İstek:</span>
                <span className="font-mono font-bold text-foreground">1,248</span>
              </div>
              <div className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground">Şema Uyum Oranı:</span>
                <span className="font-mono font-bold text-emerald-400">%99.8</span>
              </div>
              <div className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground">Breaking Change:</span>
                <span className="font-mono font-bold text-foreground">0</span>
              </div>
            </div>

            <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/5 p-4 text-xs space-y-1.5">
              <div className="flex items-center gap-1.5 font-bold text-emerald-400">
                <CheckCircle2 className="h-4 w-4" />
                <span>Otomatik Doğrulama Açık</span>
              </div>
              <p className="text-muted-foreground">
                Gelen her istek bu JSON Schema sözleşmesine göre otomatik denetlenir. Uyumsuz alanlar Güvenlik Bulguları paneline yansıtılır.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
