"use client";

import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import { Project } from "@apisentinel/shared";
import {
  BellRing,
  Plus,
  Send,
  Trash2,
  CheckCircle2,
  AlertTriangle,
  Loader2,
  Radio,
  ExternalLink,
  ShieldAlert,
  Lock,
} from "lucide-react";
import { useActiveProject } from "../../../contexts/ProjectContext";

interface AlertChannel {
  id: string;
  project_id: string;
  name: string;
  channel_type: "SLACK" | "DISCORD" | "TELEGRAM" | "WEBHOOK";
  webhook_url: string;
  min_severity: string;
  is_enabled: boolean;
  created_at: string;
}

export default function AlertsPage() {
  const queryClient = useQueryClient();
  const { accessToken, organization } = useAuth();
  const { projects, activeProjectId, setActiveProjectId } = useActiveProject();

  const isViewer = organization?.role === "VIEWER";

  const [isCreating, setIsCreating] = useState(false);
  const [testingId, setTestingId] = useState<string | null>(null);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  // New Channel Form
  const [name, setName] = useState("");
  const [channelType, setChannelType] = useState<"SLACK" | "DISCORD" | "TELEGRAM" | "WEBHOOK">("SLACK");
  const [webhookUrl, setWebhookUrl] = useState("");
  const [minSeverity, setMinSeverity] = useState("HIGH");

  // Fetch alert channels
  const { data: channelsData, isLoading } = useQuery({
    queryKey: ["alertChannels", activeProjectId],
    queryFn: () =>
      apiFetch<AlertChannel[]>(`/api/projects/${activeProjectId}/alerts`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
  });

  const channels = channelsData || [];

  // 3. Create channel mutation
  const createMutation = useMutation({
    mutationFn: (data: { name: string; channelType: string; webhookUrl: string; minSeverity: string }) =>
      apiFetch(`/api/projects/${activeProjectId}/alerts`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify(data),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alertChannels", activeProjectId] });
      setName("");
      setWebhookUrl("");
      setIsCreating(false);
      setMessage({ type: "success", text: "Bildirim kanalı başarıyla eklendi!" });
      setTimeout(() => setMessage(null), 4000);
    },
    onError: (err: any) => {
      setMessage({ type: "error", text: err.message || "Kanal eklenemedi." });
    },
  });

  // 4. Delete channel mutation
  const deleteMutation = useMutation({
    mutationFn: (channelId: string) =>
      apiFetch(`/api/alerts/${channelId}`, {
        method: "DELETE",
        token: accessToken,
        organizationId: organization?.id,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alertChannels", activeProjectId] });
      setMessage({ type: "success", text: "Bildirim kanalı silindi." });
      setTimeout(() => setMessage(null), 4000);
    },
  });

  // 5. Test alert mutation
  const testMutation = useMutation({
    mutationFn: (channelId: string) =>
      apiFetch(`/api/alerts/${channelId}/test`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
      }),
    onSuccess: () => {
      setTestingId(null);
      setMessage({ type: "success", text: "Test güvenlik bildirimi başarıyla gönderildi!" });
      setTimeout(() => setMessage(null), 4000);
    },
    onError: (err: any) => {
      setTestingId(null);
      setMessage({ type: "error", text: "Test bildirimi gönderilemedi: " + (err.message || "Bilinmeyen hata") });
    },
  });

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !webhookUrl.trim() || !activeProjectId) return;
    createMutation.mutate({
      name: name.trim(),
      channelType,
      webhookUrl: webhookUrl.trim(),
      minSeverity,
    });
  };

  const handleTest = (channelId: string) => {
    setTestingId(channelId);
    testMutation.mutate(channelId);
  };

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <BellRing className="h-6 w-6 text-primary" />
            Çok Kanallı Güvenlik Alarmları
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            Kritik PII sızıntıları veya yetkisiz erişimlerde Slack, Discord, Telegram veya Webhook ile anında haberdar olun. Webhook URL'leri AES-256-GCM ile şifrelenir.
          </p>
        </div>

        <div className="flex items-center gap-3">
          {projects.length > 1 && (
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

          {!isViewer && (
            <button
              onClick={() => setIsCreating(true)}
              disabled={!activeProjectId}
              className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
            >
              <Plus className="h-4 w-4" />
              <span>Kanal Ekle</span>
            </button>
          )}
        </div>
      </div>

      {message && (
        <div
          className={`flex items-center gap-2 rounded-lg p-4 text-sm font-medium ${
            message.type === "success"
              ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20"
              : "bg-destructive/10 text-destructive border border-destructive/20"
          }`}
        >
          {message.type === "success" ? <CheckCircle2 className="h-5 w-5" /> : <AlertTriangle className="h-5 w-5" />}
          <span>{message.text}</span>
        </div>
      )}

      {/* Create Modal / Form */}
      {isCreating && (
        <div className="rounded-xl border border-border bg-card p-6 shadow-sm animate-in fade-in duration-200">
          <div className="flex items-center justify-between border-b border-border pb-4 mb-4">
            <h3 className="text-base font-semibold text-foreground">Yeni Alarm Kanalı Ekle</h3>
            <button
              onClick={() => setIsCreating(false)}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              Vazgeç
            </button>
          </div>

          <form onSubmit={handleCreate} className="space-y-4">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                  Kanal Adı
                </label>
                <input
                  type="text"
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="Örn: Security Ops Slack"
                  className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground/60 focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                  Kanal Türü
                </label>
                <select
                  value={channelType}
                  onChange={(e) => setChannelType(e.target.value as any)}
                  className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                >
                  <option value="SLACK">Slack Webhook</option>
                  <option value="DISCORD">Discord Webhook</option>
                  <option value="TELEGRAM">Telegram Bot</option>
                  <option value="WEBHOOK">Genel HTTP Webhook</option>
                </select>
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                Webhook URL (AES-256-GCM ile Şifrelenir)
              </label>
              <input
                type="url"
                required
                value={webhookUrl}
                onChange={(e) => setWebhookUrl(e.target.value)}
                placeholder="https://hooks.slack.com/services/... veya https://discord.com/api/webhooks/..."
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground/60 focus:outline-none focus:ring-2 focus:ring-primary"
              />
              <p className="text-[11px] text-muted-foreground mt-1">
                Webhook anahtarı veritabanına yazılmadan önce şifrelenir ve arayüzde asla açıkça gösterilmez.
              </p>
            </div>

            <div>
              <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                Minimum Bildirim Seviyesi (Severity)
              </label>
              <select
                value={minSeverity}
                onChange={(e) => setMinSeverity(e.target.value)}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              >
                <option value="LOW">Tüm Seviyeler (LOW, MEDIUM, HIGH, CRITICAL)</option>
                <option value="MEDIUM">MEDIUM ve Üzeri</option>
                <option value="HIGH">HIGH ve CRITICAL (Önerilen)</option>
                <option value="CRITICAL">Yalnızca CRITICAL Tehditler</option>
              </select>
            </div>

            <div className="flex justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={() => setIsCreating(false)}
                className="rounded-lg border border-border px-4 py-2 text-sm font-medium text-muted-foreground hover:bg-secondary transition"
              >
                İptal
              </button>
              <button
                type="submit"
                disabled={createMutation.isPending}
                className="flex items-center gap-2 rounded-lg bg-primary px-5 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
              >
                {createMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
                <span>Kanalı Kaydet</span>
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Channels List */}
      {isLoading ? (
        <div className="flex h-48 items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      ) : channels.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border py-16 text-center bg-card/40">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-secondary text-muted-foreground mb-4">
            <Radio className="h-6 w-6" />
          </div>
          <h3 className="text-base font-semibold text-foreground">Henüz Alarm Kanalı Tanımlanmadı</h3>
          <p className="mt-1 text-sm text-muted-foreground max-w-sm">
            Kritik güvenlik açıklarında ekibinize anlık bildirim gitmesi için Slack veya Discord webhook'u bağlayın.
          </p>
          {!isViewer && (
            <button
              onClick={() => setIsCreating(true)}
              className="mt-6 flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90"
            >
              <Plus className="h-4 w-4" />
              <span>İlk Bildirim Kanalını Ekle</span>
            </button>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {channels.map((channel) => (
            <div
              key={channel.id}
              className="flex flex-col justify-between rounded-xl border border-border bg-card p-5 shadow-sm space-y-4 hover:border-border/80 transition"
            >
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="rounded-md bg-primary/10 px-2 py-0.5 text-xs font-bold text-primary">
                    {channel.channel_type}
                  </span>
                  <div className="flex items-center gap-2">
                    <span className="flex items-center gap-1 text-[10px] text-emerald-400 font-mono bg-emerald-500/10 px-1.5 py-0.5 rounded border border-emerald-500/20">
                      <Lock className="h-2.5 w-2.5" />
                      <span>AES-256</span>
                    </span>
                    <span className="text-[11px] font-mono text-muted-foreground">
                      Min: {channel.min_severity}
                    </span>
                  </div>
                </div>
                <h3 className="font-bold text-foreground text-base">{channel.name}</h3>
                <p className="text-xs text-muted-foreground truncate font-mono">{channel.webhook_url}</p>
              </div>

              <div className="flex items-center justify-between pt-3 border-t border-border">
                <button
                  onClick={() => handleTest(channel.id)}
                  disabled={testingId === channel.id || isViewer}
                  className="flex items-center gap-1.5 rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-foreground hover:bg-secondary transition disabled:opacity-50"
                >
                  {testingId === channel.id ? (
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <Send className="h-3.5 w-3.5 text-primary" />
                  )}
                  <span>Test Bildirimi</span>
                </button>

                {!isViewer && (
                  <button
                    onClick={() => {
                      if (confirm(`"${channel.name}" kanalını silmek istediğinize emin misiniz?`)) {
                        deleteMutation.mutate(channel.id);
                      }
                    }}
                    className="flex items-center gap-1 rounded-lg border border-destructive/30 px-2.5 py-1.5 text-xs font-semibold text-destructive hover:bg-destructive/10 transition"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                    <span>Sil</span>
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
