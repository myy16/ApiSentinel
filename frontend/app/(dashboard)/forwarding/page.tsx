"use client";

import React, { useState, useEffect } from "react";
import {
  Share2,
  Save,
  RotateCw,
  AlertOctagon,
  CheckCircle2,
  Clock,
  ShieldCheck,
  Server,
  Loader2,
  ArrowUpRight,
} from "lucide-react";
import { useAuth } from "../../../hooks/useAuth";

interface Endpoint {
  id: string;
  name: string;
  slug: string;
}

interface ForwardingConfig {
  id: string;
  endpoint_id: string;
  target_url: string;
  max_retries: number;
  timeout_ms: number;
  custom_headers: Record<string, string>;
  is_enabled: boolean;
}

interface DLQRecord {
  id: string;
  endpoint_id: string;
  request_id: string;
  target_url: string;
  attempts: number;
  last_error: string;
  payload: string;
  status: string;
  created_at: string;
  last_attempt_at: string;
}

export default function ForwardingPage() {
  const { organization } = useAuth();
  const [projects, setProjects] = useState<{ id: string; name: string }[]>([]);
  const [selectedProjectId, setSelectedProjectId] = useState<string>("");
  const [endpoints, setEndpoints] = useState<Endpoint[]>([]);
  const [selectedEndpointId, setSelectedEndpointId] = useState<string>("");

  const [targetUrl, setTargetUrl] = useState("");
  const [maxRetries, setMaxRetries] = useState(3);
  const [timeoutMs, setTimeoutMs] = useState(5000);
  const [isEnabled, setIsEnabled] = useState(true);
  const [dlqRecords, setDlqRecords] = useState<DLQRecord[]>([]);

  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3001";

  useEffect(() => {
    fetchProjects();
  }, []);

  useEffect(() => {
    if (selectedProjectId) {
      fetchEndpoints(selectedProjectId);
    }
  }, [selectedProjectId]);

  useEffect(() => {
    if (selectedEndpointId) {
      fetchForwardingConfig(selectedEndpointId);
      fetchDLQ(selectedEndpointId);
    }
  }, [selectedEndpointId]);

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

  const fetchEndpoints = async (projId: string) => {
    try {
      const token = localStorage.getItem("accessToken");
      const orgId = localStorage.getItem("organizationId");
      const res = await fetch(`${apiUrl}/api/projects/${projId}/endpoints`, {
        headers: {
          Authorization: `Bearer ${token}`,
          "x-organization-id": orgId || "",
        },
      });
      if (res.ok) {
        const data = await res.json();
        setEndpoints(data || []);
        if (data && data.length > 0) {
          setSelectedEndpointId(data[0].id);
        } else {
          setSelectedEndpointId("");
        }
      }
    } catch (err) {
      console.error(err);
    }
  };

  const fetchForwardingConfig = async (epId: string) => {
    try {
      const token = localStorage.getItem("accessToken");
      const res = await fetch(`${apiUrl}/api/endpoints/${epId}/forwarding`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      if (res.ok) {
        const data: ForwardingConfig = await res.json();
        setTargetUrl(data.target_url || "");
        setMaxRetries(data.max_retries || 3);
        setTimeoutMs(data.timeout_ms || 5000);
        setIsEnabled(data.is_enabled);
      } else {
        setTargetUrl("");
        setMaxRetries(3);
        setTimeoutMs(5000);
        setIsEnabled(true);
      }
    } catch (err) {
      console.error(err);
    }
  };

  const fetchDLQ = async (epId: string) => {
    try {
      const token = localStorage.getItem("accessToken");
      const res = await fetch(`${apiUrl}/api/endpoints/${epId}/dlq`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      if (res.ok) {
        const data = await res.json();
        setDlqRecords(data || []);
      }
    } catch (err) {
      console.error(err);
    }
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedEndpointId || !targetUrl) return;

    setIsSaving(true);
    setMessage(null);
    try {
      const token = localStorage.getItem("accessToken");
      const res = await fetch(`${apiUrl}/api/endpoints/${selectedEndpointId}/forwarding`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          targetUrl,
          maxRetries: Number(maxRetries),
          timeoutMs: Number(timeoutMs),
          isEnabled,
          customHeaders: {},
        }),
      });

      if (res.ok) {
        setMessage({ type: "success", text: "Upstream forwarding ayarları başarıyla kaydedildi!" });
      } else {
        const errData = await res.json();
        setMessage({ type: "error", text: errData.error?.message || "Ayarlar kaydedilemedi." });
      }
    } catch (err: any) {
      setMessage({ type: "error", text: err.message });
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
            <Share2 className="h-6 w-6 text-primary" />
            Upstream Forwarding & DLQ (Reverse Ingestion)
          </h1>
          <p className="text-sm text-muted-foreground">
            Temiz webhook isteklerini asıl sunucunuza iletin; başarısız iletimleri otomatik üstel geri çekilme (retry) ve Dead Letter Queue ile güvenceye alın.
          </p>
        </div>

        {/* Project & Endpoint Selectors */}
        <div className="flex items-center gap-3">
          <select
            value={selectedProjectId}
            onChange={(e) => setSelectedProjectId(e.target.value)}
            className="rounded-lg border border-border bg-card px-3 py-1.5 text-xs font-semibold text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
          >
            {projects.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>

          <select
            value={selectedEndpointId}
            onChange={(e) => setSelectedEndpointId(e.target.value)}
            className="rounded-lg border border-border bg-card px-3 py-1.5 text-xs font-semibold text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
          >
            {endpoints.map((ep) => (
              <option key={ep.id} value={ep.id}>
                {ep.name} (/{ep.slug})
              </option>
            ))}
          </select>
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
          {message.type === "success" ? (
            <CheckCircle2 className="h-5 w-5" />
          ) : (
            <AlertOctagon className="h-5 w-5" />
          )}
          <span>{message.text}</span>
        </div>
      )}

      {/* Grid: Forwarding Configuration + DLQ Queue */}
      <div className="grid grid-cols-1 gap-8 lg:grid-cols-3">
        {/* Configuration Card */}
        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          <h2 className="text-base font-semibold text-foreground mb-4 flex items-center gap-2">
            <Server className="h-4 w-4 text-primary" />
            Hedef Sunucu Ayarları
          </h2>
          <form onSubmit={handleSave} className="space-y-4">
            <div>
              <label className="block text-xs font-semibold text-muted-foreground mb-1">
                Hedef Upstream URL (Asıl Sunucunuz)
              </label>
              <input
                type="url"
                placeholder="https://api.mycompany.com/webhooks/stripe"
                value={targetUrl}
                onChange={(e) => setTargetUrl(e.target.value)}
                required
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
              />
              <p className="text-[11px] text-muted-foreground mt-1">
                SSRF Koruması aktiftir: `127.0.0.1` ve AWS metadata adresleri otomatik engellenir.
              </p>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-semibold text-muted-foreground mb-1">
                  Maks. Yeniden Deneme
                </label>
                <input
                  type="number"
                  min="1"
                  max="10"
                  value={maxRetries}
                  onChange={(e) => setMaxRetries(Number(e.target.value))}
                  className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-muted-foreground mb-1">
                  Zaman Aşımı (ms)
                </label>
                <input
                  type="number"
                  step="500"
                  min="500"
                  max="30000"
                  value={timeoutMs}
                  onChange={(e) => setTimeoutMs(Number(e.target.value))}
                  className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary"
                />
              </div>
            </div>

            <div className="flex items-center gap-3 pt-2">
              <input
                type="checkbox"
                id="enableFwd"
                checked={isEnabled}
                onChange={(e) => setIsEnabled(e.target.checked)}
                className="h-4 w-4 rounded border-border text-primary focus:ring-primary"
              />
              <label htmlFor="enableFwd" className="text-xs font-semibold text-foreground">
                Temiz Webhook İletimini Aktif Et
              </label>
            </div>

            <button
              type="submit"
              disabled={isSaving || !targetUrl}
              className="w-full flex items-center justify-center gap-2 rounded-lg bg-primary py-2.5 text-sm font-semibold text-primary-foreground hover:bg-primary/90 transition disabled:opacity-50"
            >
              {isSaving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
              Ayarları Kaydet
            </button>
          </form>
        </div>

        {/* DLQ Dead Letter Queue Table */}
        <div className="lg:col-span-2 space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-base font-semibold text-foreground flex items-center gap-2">
              <AlertOctagon className="h-4 w-4 text-destructive" />
              Dead Letter Queue (DLQ) — İletilemeyen İstekler ({dlqRecords.length})
            </h2>
            <button
              onClick={() => selectedEndpointId && fetchDLQ(selectedEndpointId)}
              className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
            >
              <RotateCw className="h-3 w-3" />
              Yenile
            </button>
          </div>

          {dlqRecords.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border p-12 text-center bg-card/50">
              <ShieldCheck className="h-10 w-10 text-emerald-400 mb-3" />
              <p className="text-sm font-semibold text-foreground">DLQ Tertemiz!</p>
              <p className="text-xs text-muted-foreground mt-1">
                Tüm temiz webhook iletimleri başarıyla tamamlandı, havuzda bekleyen başarısız istek bulunmuyor.
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {dlqRecords.map((item) => (
                <div
                  key={item.id}
                  className="rounded-xl border border-destructive/30 bg-destructive/5 p-4 space-y-2"
                >
                  <div className="flex items-center justify-between text-xs">
                    <span className="font-mono font-bold text-destructive">
                      Hedef: {item.target_url}
                    </span>
                    <span className="rounded bg-destructive/20 px-2 py-0.5 font-semibold text-destructive text-[10px]">
                      {item.attempts} Deneme Sonrası Başarısız
                    </span>
                  </div>

                  <p className="text-xs text-muted-foreground font-mono">
                    Hata: {item.last_error || "Hedef sunucu yanıt vermedi"}
                  </p>

                  <div className="flex items-center justify-between pt-1 text-[11px] text-muted-foreground border-t border-border/40">
                    <span className="flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      {new Date(item.last_attempt_at).toLocaleString("tr-TR")}
                    </span>
                    <span className="font-mono text-[10px]">İstek ID: {item.request_id}</span>
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
