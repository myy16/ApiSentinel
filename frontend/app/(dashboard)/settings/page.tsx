"use client";

import React, { useState } from "react";
import { useAuth } from "../../../hooks/useAuth";
import {
  Settings,
  User,
  Building2,
  Key,
  Shield,
  Bell,
  Copy,
  Check,
  CheckCircle2,
  AlertTriangle,
  Lock,
  Terminal,
  Cpu,
  RefreshCw,
} from "lucide-react";

export default function SettingsPage() {
  const { user, organization, accessToken } = useAuth();
  const [copiedToken, setCopiedToken] = useState(false);
  const [message, setMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  // Toggle Preferences state
  const [autoMaskPII, setAutoMaskPII] = useState(true);
  const [strictSqlXss, setStrictSqlXss] = useState(true);
  const [autoDlqRetry, setAutoDlqRetry] = useState(true);
  const [slackAlertsEnabled, setSlackAlertsEnabled] = useState(true);

  const copyToken = () => {
    if (!accessToken) return;
    navigator.clipboard.writeText(accessToken);
    setCopiedToken(true);
    setTimeout(() => setCopiedToken(false), 2500);
  };

  const handleSavePreferences = (e: React.FormEvent) => {
    e.preventDefault();
    setMessage({ type: "success", text: "Güvenlik ve bildirim tercihleri başarıyla kaydedildi!" });
    setTimeout(() => setMessage(null), 4000);
  };

  return (
    <div className="space-y-8 max-w-5xl">
      {/* Header */}
      <div>
        <div className="flex items-center gap-2">
          <Settings className="h-6 w-6 text-primary" />
          <h1 className="text-2xl font-bold tracking-tight text-foreground">Sistem & Organizasyon Ayarları</h1>
        </div>
        <p className="text-sm text-muted-foreground mt-1">
          Hesap profili, JWT kimlik doğrulama anahtarları, güvenlik politikası tercihleri ve ortam parametrelerini yönetin.
        </p>
      </div>

      {message && (
        <div
          className={`flex items-center gap-2 rounded-xl p-4 text-sm font-medium ${
            message.type === "success"
              ? "bg-emerald-500/10 text-emerald-400 border border-emerald-500/20"
              : "bg-destructive/10 text-destructive border border-destructive/20"
          }`}
        >
          {message.type === "success" ? <CheckCircle2 className="h-5 w-5" /> : <AlertTriangle className="h-5 w-5" />}
          <span>{message.text}</span>
        </div>
      )}

      {/* Grid: 2 Columns */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Profile Card */}
        <div className="rounded-2xl border border-border bg-card p-6 shadow-sm space-y-4">
          <div className="flex items-center gap-3 border-b border-border pb-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10 text-primary border border-primary/20">
              <User className="h-5 w-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-foreground">Kullanıcı & Organizasyon Profili</h2>
              <p className="text-xs text-muted-foreground">Aktif oturum bilgileri</p>
            </div>
          </div>

          <div className="space-y-3 text-xs">
            <div>
              <label className="text-muted-foreground uppercase font-semibold text-[10px] tracking-wider">
                E-posta Adresi
              </label>
              <p className="mt-1 font-semibold text-foreground bg-secondary/30 rounded-lg p-2.5 border border-border">
                {user?.email}
              </p>
            </div>

            <div>
              <label className="text-muted-foreground uppercase font-semibold text-[10px] tracking-wider">
                Organizasyon
              </label>
              <p className="mt-1 font-semibold text-foreground bg-secondary/30 rounded-lg p-2.5 border border-border flex items-center justify-between">
                <span>{organization?.name || "Varsayılan Organizasyon"}</span>
                <span className="rounded bg-primary/10 text-primary px-2 py-0.5 text-[10px] font-bold">SAHİP</span>
              </p>
            </div>

            <div>
              <label className="text-muted-foreground uppercase font-semibold text-[10px] tracking-wider">
                Kullanıcı ID
              </label>
              <p className="mt-1 font-mono text-muted-foreground bg-secondary/30 rounded-lg p-2.5 border border-border truncate">
                {user?.id}
              </p>
            </div>
          </div>
        </div>

        {/* API & JWT Key Card */}
        <div className="rounded-2xl border border-border bg-card p-6 shadow-sm space-y-4">
          <div className="flex items-center gap-3 border-b border-border pb-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-amber-500/10 text-amber-400 border border-amber-500/20">
              <Key className="h-5 w-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-foreground">API & CLI Erişim Token'ı</h2>
              <p className="text-xs text-muted-foreground">CLI ve gRPC tünelleri için JWT yetkilendirmesi</p>
            </div>
          </div>

          <div className="space-y-3 text-xs">
            <p className="text-muted-foreground">
              ApiSentinel CLI ajanını veya CI/CD boru hattını bağlarken bu taşıyıcı (bearer) token kullanılır.
            </p>

            <div className="relative">
              <textarea
                readOnly
                rows={3}
                value={accessToken || "Token bulunamadı"}
                className="w-full rounded-lg bg-background p-3 font-mono text-[11px] text-muted-foreground border border-border focus:outline-none select-all"
              />
              <button
                onClick={copyToken}
                className="mt-2 w-full flex items-center justify-center gap-2 rounded-lg bg-secondary px-3 py-2 text-xs font-semibold text-foreground hover:bg-primary hover:text-primary-foreground transition"
              >
                {copiedToken ? <Check className="h-4 w-4 text-emerald-400" /> : <Copy className="h-4 w-4" />}
                <span>{copiedToken ? "Token Panoya Kopyalandı!" : "JWT Token'ı Kopyala"}</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Global Security Policy Preferences */}
      <div className="rounded-2xl border border-border bg-card p-6 shadow-sm space-y-6">
        <div className="flex items-center gap-3 border-b border-border pb-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
            <Shield className="h-5 w-5" />
          </div>
          <div>
            <h2 className="text-base font-bold text-foreground">Otomatik Güvenlik & İhlal Politikaları</h2>
            <p className="text-xs text-muted-foreground">Gateway seviyesinde uygulanan güvenlik korumaları</p>
          </div>
        </div>

        <form onSubmit={handleSavePreferences} className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="flex items-start gap-3 rounded-xl border border-border bg-secondary/20 p-4">
              <input
                type="checkbox"
                id="autoMask"
                checked={autoMaskPII}
                onChange={(e) => setAutoMaskPII(e.target.checked)}
                className="mt-1 h-4 w-4 rounded border-border text-primary focus:ring-primary"
              />
              <div>
                <label htmlFor="autoMask" className="text-sm font-semibold text-foreground cursor-pointer">
                  Otomatik PII & Secret Maskeleme
                </label>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Kredi kartı, API anahtarı ve TCKN gibi hassas verileri veritabanına ve loglara yazılmadan önce otomatik `****` ile sansürle.
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3 rounded-xl border border-border bg-secondary/20 p-4">
              <input
                type="checkbox"
                id="strictSql"
                checked={strictSqlXss}
                onChange={(e) => setStrictSqlXss(e.target.checked)}
                className="mt-1 h-4 w-4 rounded border-border text-primary focus:ring-primary"
              />
              <div>
                <label htmlFor="strictSql" className="text-sm font-semibold text-foreground cursor-pointer">
                  Katı SQLi / XSS Engelleme (Active Defense)
                </label>
                <p className="text-xs text-muted-foreground mt-0.5">
                  SQL Enjeksiyonu veya Script saldırısı içeren webhook isteklerini anında 403 Forbidden ile reddet ve DLQ'ya al.
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3 rounded-xl border border-border bg-secondary/20 p-4">
              <input
                type="checkbox"
                id="autoDlq"
                checked={autoDlqRetry}
                onChange={(e) => setAutoDlqRetry(e.target.checked)}
                className="mt-1 h-4 w-4 rounded border-border text-primary focus:ring-primary"
              />
              <div>
                <label htmlFor="autoDlq" className="text-sm font-semibold text-foreground cursor-pointer">
                  DLQ Üstel Geri Çekilme (Exponential Backoff)
                </label>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Asıl sunucuya iletilemeyen webhook'ları 3 defaya kadar artan aralıklarla (1s, 2s, 4s) otomatik tekrar dene.
                </p>
              </div>
            </div>

            <div className="flex items-start gap-3 rounded-xl border border-border bg-secondary/20 p-4">
              <input
                type="checkbox"
                id="slackAlerts"
                checked={slackAlertsEnabled}
                onChange={(e) => setSlackAlertsEnabled(e.target.checked)}
                className="mt-1 h-4 w-4 rounded border-border text-primary focus:ring-primary"
              />
              <div>
                <label htmlFor="slackAlerts" className="text-sm font-semibold text-foreground cursor-pointer">
                  Kritik Alarm Gönderimi (Slack/Discord)
                </label>
                <p className="text-xs text-muted-foreground mt-0.5">
                  CRITICAL ve HIGH seviyesindeki güvenlik ihlallerinde bildirim kanallarına anlık webhook gönder.
                </p>
              </div>
            </div>
          </div>

          <div className="flex justify-end pt-2">
            <button
              type="submit"
              className="flex items-center gap-2 rounded-xl bg-primary px-5 py-2 text-sm font-semibold text-primary-foreground shadow-sm hover:bg-primary/90 transition"
            >
              <span>Tercihleri Kaydet</span>
            </button>
          </div>
        </form>
      </div>

      {/* Engine & Runtime Specs */}
      <div className="rounded-2xl border border-border bg-card p-6 shadow-sm space-y-4">
        <div className="flex items-center gap-3 border-b border-border pb-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-purple-500/10 text-purple-400 border border-purple-500/20">
            <Cpu className="h-5 w-5" />
          </div>
          <div>
            <h2 className="text-base font-bold text-foreground">Sistem Mimarisi & Çalışma Durumu</h2>
            <p className="text-xs text-muted-foreground">Aktif servis katmanları</p>
          </div>
        </div>

        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 text-xs">
          <div className="rounded-xl border border-border bg-secondary/30 p-3.5 space-y-1">
            <span className="text-muted-foreground font-medium">Gateway Engine</span>
            <p className="font-bold text-foreground font-mono">Go 1.22 + Ingestion</p>
          </div>
          <div className="rounded-xl border border-border bg-secondary/30 p-3.5 space-y-1">
            <span className="text-muted-foreground font-medium">Message Broker</span>
            <p className="font-bold text-foreground font-mono">Valkey 8 Streams</p>
          </div>
          <div className="rounded-xl border border-border bg-secondary/30 p-3.5 space-y-1">
            <span className="text-muted-foreground font-medium">Veritabanı</span>
            <p className="font-bold text-foreground font-mono">PostgreSQL 16</p>
          </div>
          <div className="rounded-xl border border-border bg-secondary/30 p-3.5 space-y-1">
            <span className="text-muted-foreground font-medium">Frontend Console</span>
            <p className="font-bold text-foreground font-mono">Next.js 14 App Router</p>
          </div>
        </div>
      </div>
    </div>
  );
}
