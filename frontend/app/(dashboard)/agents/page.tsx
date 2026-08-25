"use client";

import React, { useState } from "react";
import { useAuth } from "../../../hooks/useAuth";
import {
  Terminal,
  Activity,
  CheckCircle2,
  Copy,
  Check,
  Laptop,
  Radio,
  Clock,
  ShieldCheck,
  Zap,
} from "lucide-react";

export default function AgentsPage() {
  const { organization } = useAuth();
  const [copiedCommand, setCopiedCommand] = useState(false);

  // Demo connected agent list
  const agents = [
    {
      id: "agent_dev_macbook_1",
      hostname: "dev-macbook-pro.local",
      os: "darwin (arm64)",
      version: "v0.1.0",
      status: "ONLINE",
      lastSeen: new Date().toISOString(),
      activeTunnel: "http://localhost:8080",
    },
  ];

  const connectCommand = "apisentinel connect --server localhost:50051 --token super_secret_jwt_key_at_least_32_characters_long_12345";

  const copyToClipboard = () => {
    navigator.clipboard.writeText(connectCommand);
    setCopiedCommand(true);
    setTimeout(() => setCopiedCommand(false), 2000);
  };

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-bold tracking-tight">Local Geliştirici Ajanları (Agent Network)</h1>
            <span className="flex items-center gap-1 rounded-full bg-emerald-500/10 px-2.5 py-0.5 text-xs font-semibold text-emerald-400">
              <Radio className="h-3 w-3 animate-pulse" />
              gRPC Bi-directional Active
            </span>
          </div>
          <p className="text-sm text-muted-foreground">
            Geliştirici bilgisayarlarında çalışan ApiSentinel CLI ajanlarını ve canlı gRPC tünellerini yönetin
          </p>
        </div>
      </div>

      {/* Connection Command Card */}
      <div className="rounded-xl border border-border bg-card p-6 shadow-sm space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <Terminal className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-sm font-bold text-foreground">Yeni Ajan Bağla (gRPC Tunnel)</h3>
              <p className="text-xs text-muted-foreground">
                Terminalinizde aşağıdaki komutu çalıştırarak local API'nizi Cloud ile çift yönlü tünele bağlayın
              </p>
            </div>
          </div>
        </div>

        <div className="flex items-center justify-between rounded-lg border border-border bg-background p-3">
          <code className="text-xs font-mono text-foreground">{connectCommand}</code>
          <button
            onClick={copyToClipboard}
            className="flex items-center gap-1.5 rounded-lg bg-secondary px-3 py-1.5 text-xs font-semibold text-foreground transition hover:bg-primary hover:text-primary-foreground"
          >
            {copiedCommand ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
            <span>{copiedCommand ? "Kopyalandı" : "Kopyala"}</span>
          </button>
        </div>
      </div>

      {/* Connected Agents Grid */}
      <div className="grid grid-cols-1 gap-4">
        {agents.map((agent) => (
          <div
            key={agent.id}
            className="rounded-xl border border-border bg-card p-6 shadow-sm flex flex-col justify-between gap-4"
          >
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400">
                  <Laptop className="h-5 w-5" />
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <h3 className="text-sm font-bold text-foreground">{agent.hostname}</h3>
                    <span className="rounded bg-emerald-500/20 text-emerald-400 px-2 py-0.5 text-[11px] font-bold font-mono">
                      ONLINE
                    </span>
                  </div>
                  <p className="text-xs text-muted-foreground font-mono">
                    ID: {agent.id} • OS: {agent.os} • Sürüm: {agent.version}
                  </p>
                </div>
              </div>

              <div className="flex items-center gap-2 text-xs text-muted-foreground font-mono">
                <Clock className="h-3.5 w-3.5" />
                <span>Son Heartbeat: {new Date(agent.lastSeen).toLocaleTimeString("tr-TR")}</span>
              </div>
            </div>

            <div className="flex items-center justify-between pt-3 border-t border-border text-xs">
              <div className="flex items-center gap-2 text-muted-foreground">
                <ShieldCheck className="h-4 w-4 text-emerald-400" />
                <span>Git Pre-Push Hook: <strong>Aktif & Korumalı</strong></span>
              </div>

              <div className="flex items-center gap-2">
                <span className="text-muted-foreground">Local Port:</span>
                <span className="font-mono font-bold text-primary">{agent.activeTunnel}</span>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
