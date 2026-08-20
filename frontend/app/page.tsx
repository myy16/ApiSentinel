import { Shield, ArrowRight, Terminal, Activity, Zap, CheckCircle2 } from "lucide-react";

export default function Home() {
  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      {/* Header */}
      <header className="sticky top-0 z-50 flex h-16 w-full items-center justify-between border-b border-border bg-background/80 px-6 backdrop-blur">
        <div className="flex items-center gap-3">
          <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/20 text-primary">
            <Shield className="h-5 w-5" />
          </div>
          <span className="text-xl font-bold tracking-tight">ApiSentinel</span>
          <span className="rounded-full bg-primary/10 px-2.5 py-0.5 text-xs font-semibold text-primary">
            Phase 0 Initialized
          </span>
        </div>
        <div className="flex items-center gap-4">
          <a
            href="http://localhost:3001/health"
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-1.5 text-xs text-muted-foreground transition hover:text-foreground"
          >
            <Activity className="h-3.5 w-3.5 text-emerald-500" />
            Backend Health
          </a>
        </div>
      </header>

      {/* Hero Section */}
      <main className="flex flex-1 flex-col items-center justify-center px-6 py-20 text-center">
        <div className="inline-flex items-center gap-2 rounded-full border border-border bg-card/60 px-3 py-1 text-xs font-medium text-muted-foreground backdrop-blur mb-6">
          <Zap className="h-3.5 w-3.5 text-primary" />
          <span>Full-Stack TypeScript Architecture Ready</span>
        </div>

        <h1 className="max-w-4xl text-4xl font-extrabold tracking-tight sm:text-6xl">
          Developer Security &{" "}
          <span className="bg-gradient-to-r from-blue-400 via-indigo-400 to-sky-400 bg-clip-text text-transparent">
            Integration Platform
          </span>
        </h1>

        <p className="mt-6 max-w-2xl text-lg text-muted-foreground">
          Detect. Decide. Prevent. Test. Protect your webhooks and APIs at the source, gateway, and runtime with deterministic security policies.
        </p>

        {/* Feature Highlights Grid */}
        <div className="mt-16 grid w-full max-w-5xl grid-cols-1 gap-6 sm:grid-cols-3 text-left">
          <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-500/10 text-blue-400 mb-4">
              <Terminal className="h-5 w-5" />
            </div>
            <h3 className="text-lg font-semibold">Fastify + Drizzle ORM</h3>
            <p className="mt-2 text-sm text-muted-foreground">
              Ultra low-latency webhook ingestion with PostgreSQL 16 & zero-overhead type safety.
            </p>
          </div>

          <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400 mb-4">
              <Zap className="h-5 w-5" />
            </div>
            <h3 className="text-lg font-semibold">Valkey 8 Streams</h3>
            <p className="mt-2 text-sm text-muted-foreground">
              100% open-source streaming engine for async PII, secret scanning and duplicate detection.
            </p>
          </div>

          <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-500/10 text-indigo-400 mb-4">
              <CheckCircle2 className="h-5 w-5" />
            </div>
            <h3 className="text-lg font-semibold">Next.js 14 Dashboard</h3>
            <p className="mt-2 text-sm text-muted-foreground">
              Real-time request inspector with SSE streaming, replay lab, and visual mock builder.
            </p>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-border py-6 text-center text-xs text-muted-foreground">
        ApiSentinel © 2026 — Phase 0 Monorepo Bootstrap Complete
      </footer>
    </div>
  );
}
