"use client";

import React, { useState, useEffect } from "react";
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
} from "lucide-react";
import { useAuth } from "../../../hooks/useAuth";

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
  const { organization } = useAuth();
  const [channels, setChannels] = useState<AlertChannel[]>([]);
  const [projects, setProjects] = useState<{ id: string; name: string }[]>([]);
  const [selectedProjectId, setSelectedProjectId] = useState<string>("");
  const [isLoading, setIsLoading] = useState(true);
  const [isCreating, setIsCreating] = useState(false);
  const [testingId, setTestingId] = useState<string | null>(null);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  // New Channel Form
  const [name, setName] = useState("");
  const [channelType, setChannelType] = useState<"SLACK" | "DISCORD" | "TELEGRAM" | "WEBHOOK">("SLACK");
  const [webhookUrl, setWebhookUrl] = useState("");
  const [minSeverity, setMinSeverity] = useState("HIGH");

  const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3001";

  useEffect(() => {
    fetchProjects();
  }, []);

  useEffect(() => {
    if (selectedProjectId) {
      fetchChannels(selectedProjectId);
    }
  }, [selectedProjectId]);

  const fetchProjects = async () => {
    try {
      const token = localStorage.getItem("accessToken");
      const orgId = localStorage.getItem("organizationId");
      const res = await fetch(`${apiUrl}/api/projects`, {
        headers: {
          Authorization: `Bearer ${token}`,
          "x-organization-id": orgId || "",
        },
      });
      if (res.ok) {
        const data = await res.json();
        setProjects(data || []);
        if (data && data.length > 0) {
          setSelectedProjectId(data[0].id);
        }
      }
    } catch (err) {
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  const fetchChannels = async (projId: string) => {
    try {
      const token = localStorage.getItem("accessToken");
      const orgId = localStorage.getItem("organizationId");
      const res = await fetch(`${apiUrl}/api/projects/${projId}/alerts`, {
        headers: {
          Authorization: `Bearer ${token}`,
          "x-organization-id": orgId || "",
        },
      });
      if (res.ok) {
        const data = await res.json();
        setChannels(data || []);
      }
    } catch (err) {
      console.error(err);
    }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name || !webhookUrl || !selectedProjectId) return;

    setIsCreating(true);
    setMessage(null);
    try {
      const token = localStorage.getItem("accessToken");
      const orgId = localStorage.getItem("organizationId");
      const res = await fetch(`${apiUrl}/api/projects/${selectedProjectId}/alerts`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
          "x-organization-id": orgId || "",
        },
        body: JSON.stringify({
          name,
          channelType,
          webhookUrl,
          minSeverity,
        }),
      });

      if (res.ok) {
        setMessage({ type: "success", text: "Bildirim kanalı başarıyla eklendi!" });
        setName("");
        setWebhookUrl("");
        fetchChannels(selectedProjectId);
      } else {
        const errData = await res.json();
        setMessage({ type: "error", text: errData.error?.message || "Kanal eklenemedi." });
      }
    } catch (err: any) {
      setMessage({ type: "error", text: err.message });
    } finally {
      setIsCreating(false);
    }
  };

  const handleDelete = async (channelId: string) => {
    try {
      const token = localStorage.getItem("accessToken");
      const res = await fetch(`${apiUrl}/api/alerts/${channelId}`, {
        method: "DELETE",
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      if (res.ok) {
        fetchChannels(selectedProjectId);
      }
    } catch (err) {
      console.error(err);
    }
  };

  const handleTest = async (channelId: string) => {
    setTestingId(channelId);
    setMessage(null);
    try {
      const token = localStorage.getItem("accessToken");
      const res = await fetch(`${apiUrl}/api/alerts/${channelId}/test`, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      if (res.ok) {
        setMessage({ type: "success", text: "Test bildirimi hedefe başarıyla ulaştırıldı! 🎉" });
      } else {
        setMessage({ type: "error", text: "Test bildirimi gönderilemedi. Webhook URL'sini kontrol edin." });
      }
    } catch (err: any) {
      setMessage({ type: "error", text: err.message });
    } finally {
      setTestingId(null);
    }
  };

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <BellRing className="h-6 w-6 text-primary" />
            Çok Kanallı Güvenlik Bildirimleri
          </h1>
          <p className="text-sm text-muted-foreground">
            Kritik sızıntılarda (OpenAI, AWS, DB Şifresi) Slack, Discord, Telegram veya Webhook'a anında uyarı fırlatın.
          </p>
        </div>

        {/* Project Selector */}
        {projects.length > 0 && (
          <div className="flex items-center gap-3">
            <span className="text-xs font-semibold text-muted-foreground">Proje:</span>
            <select
              value={selectedProjectId}
              onChange={(e) => setSelectedProjectId(e.target.value)}
              className="rounded-lg border border-border bg-card px-3 py-1.5 text-sm font-medium text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
            >
              {projects.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </div>
        )}
      </div>

      {message && (
        <div
          className={`flex items-center gap-2 rounded-lg p-4 text-sm font-medium ${
            message.type === "success"
              ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20"
              : "bg-destructive/10 text-destructive border border-destructive/20"
          }`}
        >
          {message.type === "success" ? (
            <CheckCircle2 className="h-5 w-5" />
          ) : (
            <AlertTriangle className="h-5 w-5" />
          )}
          <span>{message.text}</span>
        </div>
      )}

      {/* Grid Layout: Create Form + Channels List */}
      <div className="grid grid-cols-1 gap-8 lg:grid-cols-3">
        {/* Create Form */}
        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          <h2 className="text-base font-semibold text-foreground mb-4 flex items-center gap-2">
            <Plus className="h-4 w-4 text-primary" />
            Yeni Bildirim Kanalı Ekle
          </h2>
          <form onSubmit={handleCreate} className="space-y-4">
            <div>
              <label className="block text-xs font-semibold text-muted-foreground mb-1">
                Kanal Adı
              </label>
              <input
                type="text"
                placeholder="Örn: Security Ops Slack"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-muted-foreground mb-1">
                Kanal Türü
              </label>
              <div className="grid grid-cols-2 gap-2">
                {(["SLACK", "DISCORD", "TELEGRAM", "WEBHOOK"] as const).map((type) => (
                  <button
                    key={type}
                    type="button"
                    onClick={() => setChannelType(type)}
                    className={`rounded-lg border px-3 py-2 text-xs font-semibold transition ${
                      channelType === type
                        ? "border-primary bg-primary/10 text-primary"
                        : "border-border bg-background text-muted-foreground hover:text-foreground"
                    }`}
                  >
                    {type}
                  </button>
                ))}
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-muted-foreground mb-1">
                Webhook URL / Endpoint
              </label>
              <input
                type="url"
                placeholder="https://hooks.slack.com/services/..."
                value={webhookUrl}
                onChange={(e) => setWebhookUrl(e.target.value)}
                required
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-muted-foreground mb-1">
                Minimum Bildirim Seviyesi
              </label>
              <select
                value={minSeverity}
                onChange={(e) => setMinSeverity(e.target.value)}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              >
                <option value="CRITICAL">Sadece CRITICAL (Kritik Sızıntılar)</option>
                <option value="HIGH">HIGH ve Üzeri (Önerilen)</option>
                <option value="ALL">Tüm Olaylar (INFO Dahil)</option>
              </select>
            </div>

            <button
              type="submit"
              disabled={isCreating || !name || !webhookUrl}
              className="w-full flex items-center justify-center gap-2 rounded-lg bg-primary py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90 transition disabled:opacity-50"
            >
              {isCreating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
              Kanalı Kaydet
            </button>
          </form>
        </div>

        {/* Channel Cards */}
        <div className="lg:col-span-2 space-y-4">
          <h2 className="text-base font-semibold text-foreground flex items-center justify-between">
            <span>Aktif Bildirim Kanalları ({channels.length})</span>
          </h2>

          {channels.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border p-12 text-center bg-card/50">
              <Radio className="h-10 w-10 text-muted-foreground mb-3" />
              <p className="text-sm font-semibold text-foreground">Henüz bildirim kanalı eklenmedi</p>
              <p className="text-xs text-muted-foreground mt-1 max-w-sm">
                Sol taraftaki panelden Slack, Discord veya Telegram webhook adresinizi tanımlayarak sızıntılardan anında haberdar olun.
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {channels.map((ch) => (
                <div
                  key={ch.id}
                  className="flex items-center justify-between rounded-xl border border-border bg-card p-4 transition hover:border-primary/40"
                >
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary font-bold text-xs">
                      {ch.channel_type}
                    </div>
                    <div>
                      <h3 className="text-sm font-bold text-foreground">{ch.name}</h3>
                      <p className="text-xs text-muted-foreground truncate max-w-md font-mono">
                        {ch.webhook_url}
                      </p>
                      <div className="flex items-center gap-2 mt-1">
                        <span className="rounded bg-secondary px-2 py-0.5 text-[10px] font-semibold text-muted-foreground">
                          Eşik: {ch.min_severity}
                        </span>
                        <span className="text-xs text-emerald-400 flex items-center gap-1">
                          <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
                          Aktif
                        </span>
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handleTest(ch.id)}
                      disabled={testingId === ch.id}
                      className="flex items-center gap-1.5 rounded-lg border border-border bg-secondary/60 px-3 py-1.5 text-xs font-semibold text-foreground hover:bg-secondary transition disabled:opacity-50"
                    >
                      {testingId === ch.id ? (
                        <Loader2 className="h-3.5 w-3.5 animate-spin" />
                      ) : (
                        <Send className="h-3.5 w-3.5" />
                      )}
                      Test Gönder
                    </button>
                    <button
                      onClick={() => handleDelete(ch.id)}
                      className="rounded-lg p-2 text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
