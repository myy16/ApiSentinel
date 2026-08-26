import Link from "next/link";
import {
  Shield,
  ArrowRight,
  Terminal,
  Activity,
  Zap,
  CheckCircle2,
  Lock,
  Radio,
  FileCode,
  Repeat,
  Sparkles,
  Bot,
  Layers,
  Cpu,
  Globe,
  Share2,
} from "lucide-react";

export default function Home() {
  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground selection:bg-primary/30 selection:text-white">
      {/* Header */}
      <header className="sticky top-0 z-50 flex h-16 w-full items-center justify-between border-b border-border bg-background/80 px-6 md:px-12 backdrop-blur-md">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-primary/30 to-primary/10 border border-primary/30 text-primary shadow-sm">
            <Shield className="h-5 w-5" />
          </div>
          <div className="flex flex-col">
            <span className="text-lg font-extrabold tracking-tight">ApiSentinel</span>
            <span className="text-[10px] font-mono text-muted-foreground uppercase tracking-widest">
              Security Gateway
            </span>
          </div>
        </div>

        <nav className="hidden md:flex items-center gap-8 text-xs font-semibold text-muted-foreground">
          <a href="#features" className="hover:text-foreground transition">
            Özellikler
          </a>
          <a href="#architecture" className="hover:text-foreground transition">
            Mimari
          </a>
          <a href="#interactive-preview" className="hover:text-foreground transition">
            Canlı Simülasyon
          </a>
        </nav>

        <div className="flex items-center gap-3">
          <Link
            href="/login"
            className="rounded-xl px-4 py-2 text-xs font-semibold text-foreground hover:bg-secondary transition"
          >
            Giriş Yap
          </Link>
          <Link
            href="/overview"
            className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-xs font-bold text-primary-foreground shadow-sm hover:bg-primary/90 transition"
          >
            <span>Konsola Git</span>
            <ArrowRight className="h-3.5 w-3.5" />
          </Link>
        </div>
      </header>

      {/* Hero Section */}
      <main className="flex flex-1 flex-col items-center justify-center px-6 py-20 md:py-28 text-center max-w-6xl mx-auto">
        <div className="inline-flex items-center gap-2 rounded-full border border-primary/30 bg-primary/10 px-3.5 py-1 text-xs font-semibold text-primary backdrop-blur mb-6 animate-in fade-in duration-300">
          <Zap className="h-3.5 w-3.5 text-primary" />
          <span>Go Ingestion Gateway + AI Remediation 2.0</span>
        </div>

        <h1 className="max-w-4xl text-4xl font-black tracking-tight sm:text-6xl lg:text-7xl leading-tight">
          Webhook & API Güvenliğinizi{" "}
          <span className="bg-gradient-to-r from-blue-400 via-indigo-400 to-sky-400 bg-clip-text text-transparent">
            Saniyeler İçinde
          </span>{" "}
          Zırhlayın.
        </h1>

        <p className="mt-6 max-w-2xl text-base sm:text-lg text-muted-foreground leading-relaxed">
          Gelen tüm webhook trafiğini gerçek zamanlı denetleyin. PII ve Secret sızıntılarını maskeleyin, SQLi/XSS saldırılarını kapıda durdurun, JSON Schema ile sözleşmeleri doğrulayın ve DLQ ile hiçbir isteği kaybetmeyin.
        </p>

        {/* CTA Buttons */}
        <div className="mt-8 flex flex-col sm:flex-row items-center gap-4">
          <Link
            href="/register"
            className="flex items-center gap-2 rounded-xl bg-primary px-6 py-3.5 text-sm font-bold text-primary-foreground shadow-lg shadow-primary/20 hover:bg-primary/90 hover:scale-105 transition"
          >
            <span>Ücretsiz Başlayın</span>
            <ArrowRight className="h-4 w-4" />
          </Link>
          <Link
            href="/overview"
            className="flex items-center gap-2 rounded-xl border border-border bg-card px-6 py-3.5 text-sm font-semibold text-foreground hover:bg-secondary transition"
          >
            <Radio className="h-4 w-4 text-emerald-400" />
            <span>Canlı Konsolu Aç</span>
          </Link>
        </div>

        {/* Interactive Live Defense Preview Box */}
        <div
          id="interactive-preview"
          className="mt-16 w-full max-w-4xl rounded-2xl border border-border bg-card/80 p-6 md:p-8 shadow-2xl backdrop-blur-md text-left space-y-6 glow-subtle"
        >
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 border-b border-border pb-4">
            <div className="flex items-center gap-3">
              <div className="flex h-3 w-3 rounded-full bg-rose-500" />
              <div className="flex h-3 w-3 rounded-full bg-amber-500" />
              <div className="flex h-3 w-3 rounded-full bg-emerald-500" />
              <span className="font-mono text-xs text-muted-foreground ml-2">
                POST /hook/payment-gateway
              </span>
            </div>
            <div className="flex items-center gap-2">
              <span className="rounded-md bg-rose-500/20 text-rose-400 px-2.5 py-0.5 text-xs font-mono font-bold border border-rose-500/30">
                POLICY_BLOCKED (403)
              </span>
              <span className="rounded-md bg-purple-500/20 text-purple-400 px-2.5 py-0.5 text-xs font-mono font-bold border border-purple-500/30">
                AI Explained
              </span>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 font-mono text-xs">
            {/* Incoming Malicious Request */}
            <div className="rounded-xl bg-background p-4 border border-border space-y-2">
              <span className="text-[11px] font-bold text-muted-foreground uppercase tracking-wider">
                Gelen Zararlı Webhook Yükü
              </span>
              <pre className="text-rose-400 text-[11px] overflow-x-auto leading-relaxed">
{`{
  "order_id": "' OR 1=1 --",
  "credit_card": "4532-****-****-1092",
  "customer": {
    "email": "ahmet@example.com"
  }
}`}
              </pre>
            </div>

            {/* Ingestion Gateway Defense Decision */}
            <div className="rounded-xl bg-background p-4 border border-border space-y-2">
              <span className="text-[11px] font-bold text-emerald-400 uppercase tracking-wider">
                Gateway Kararı & Maskeleme
              </span>
              <pre className="text-foreground text-[11px] overflow-x-auto leading-relaxed">
{`{
  "status": 403,
  "action": "BLOCK",
  "threat": "SQLI_TAUTOLOGY",
  "secret_masked": true,
  "dlq_stored": true,
  "slack_notified": true
}`}
              </pre>
            </div>
          </div>
        </div>

        {/* 6 Feature Pillars */}
        <div id="features" className="mt-24 w-full text-left space-y-8">
          <div className="text-center space-y-2">
            <h2 className="text-3xl font-extrabold tracking-tight">Eksiksiz API & Webhook Güvenlik Ekosistemi</h2>
            <p className="text-sm text-muted-foreground max-w-xl mx-auto">
              Uçtan uca mimari ile webhook trafiğinizi dinleyin, doğrulayın, test edin ve güvenle yönlendirin.
            </p>
          </div>

          <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
            <div className="rounded-2xl border border-border bg-card p-6 shadow-sm glow-card space-y-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-blue-500/10 text-blue-400 border border-blue-500/20">
                <Globe className="h-5 w-5" />
              </div>
              <h3 className="text-base font-bold">Ultra Düşük Gecikmeli Gateway</h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Go 1.22 tabanlı ingestion motoru ile mikrosaniyeler içinde webhook karşılayın ve modları yönetin.
              </p>
            </div>

            <div className="rounded-2xl border border-border bg-card p-6 shadow-sm glow-card space-y-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-rose-500/10 text-rose-400 border border-rose-500/20">
                <Lock className="h-5 w-5" />
              </div>
              <h3 className="text-base font-bold">Aktif Tehdit & SQLi Koruması</h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                SQL injection, script etiketleri ve zararlı kalıpları asıl sunucuya ulaşmadan anında 403 ile engelleyin.
              </p>
            </div>

            <div className="rounded-2xl border border-border bg-card p-6 shadow-sm glow-card space-y-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-purple-500/10 text-purple-400 border border-purple-500/20">
                <Bot className="h-5 w-5" />
              </div>
              <h3 className="text-base font-bold">AI Güvenlik & Kod Danışmanı</h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                İhlal edilen güvenlik bulgularını tek tıkla AI ile analiz edin ve anında uygulanabilir onarım kodunu alın.
              </p>
            </div>

            <div className="rounded-2xl border border-border bg-card p-6 shadow-sm glow-card space-y-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                <Share2 className="h-5 w-5" />
              </div>
              <h3 className="text-base font-bold">Upstream Forwarding & DLQ</h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Temiz istekleri asıl sunucunuza iletin; başarısız iletimleri Dead Letter Queue ve üstel retry ile güvenceye alın.
              </p>
            </div>

            <div className="rounded-2xl border border-border bg-card p-6 shadow-sm glow-card space-y-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-amber-500/10 text-amber-400 border border-amber-500/20">
                <FileCode className="h-5 w-5" />
              </div>
              <h3 className="text-base font-bold">JSON Schema Sözleşmeleri</h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Draft 2020-12 standardı ile gelen yükleri doğrulayın; eksik zorunlu alanları kapıda yakalayın.
              </p>
            </div>

            <div className="rounded-2xl border border-border bg-card p-6 shadow-sm glow-card space-y-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-400 border border-indigo-500/20">
                <Repeat className="h-5 w-5" />
              </div>
              <h3 className="text-base font-bold">Replay & Mock Simulator</h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                SSRF korumalı Replay Lab ve dinamik 503/429 mock yanıt motoruyla webhook entegrasyonlarınızı test edin.
              </p>
            </div>
          </div>
        </div>

        {/* Architecture Specs */}
        <div id="architecture" className="mt-20 w-full rounded-2xl border border-border bg-secondary/20 p-8 text-left space-y-6">
          <div className="flex items-center gap-2">
            <Cpu className="h-5 w-5 text-primary" />
            <h3 className="text-lg font-bold">Modern Monorepo Mimarisi</h3>
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 text-xs">
            <div className="rounded-xl bg-card p-4 border border-border space-y-1">
              <span className="text-muted-foreground">Backend Engine</span>
              <p className="font-bold text-foreground font-mono">Go 1.22 + Fast Ingestion</p>
            </div>
            <div className="rounded-xl bg-card p-4 border border-border space-y-1">
              <span className="text-muted-foreground">Message Stream</span>
              <p className="font-bold text-foreground font-mono">Valkey 8 Streams</p>
            </div>
            <div className="rounded-xl bg-card p-4 border border-border space-y-1">
              <span className="text-muted-foreground">Veritabanı</span>
              <p className="font-bold text-foreground font-mono">PostgreSQL 16</p>
            </div>
            <div className="rounded-xl bg-card p-4 border border-border space-y-1">
              <span className="text-muted-foreground">Dashboard</span>
              <p className="font-bold text-foreground font-mono">Next.js 14 + Tailwind</p>
            </div>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-border py-8 text-center text-xs text-muted-foreground bg-card/40">
        <div className="flex flex-col items-center gap-2">
          <div className="flex items-center gap-2 font-bold text-foreground">
            <Shield className="h-4 w-4 text-primary" />
            <span>ApiSentinel Developer Security Console</span>
          </div>
          <p>© 2026 ApiSentinel — Tüm hakları saklıdır.</p>
        </div>
      </footer>
    </div>
  );
}

