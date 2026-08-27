"use client";

import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
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
  WifiOff,
  Loader2,
} from "lucide-react";

interface AgentSession {
  agentId: string;
  hostname: string;
  os: string;
  version: string;
  status: "ONLINE" | "STALE";
  lastSeen: string;
}

export default function AgentsPage() {
  const { accessToken, organization } = useAuth();
  const [copiedCommand, setCopiedCommand] = useState(false);

  // Fetch real connected agents from backend
  const { data: agentsData, isLoading } = useQuery({
    queryKey: ["agents", "sessions"],
    queryFn: () =>
      apiFetch<{ agents: AgentSession[]; total: number }>("/api/agents/sessions", {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken,
    refetchInterval: 10000, // Lightweight poll every 10s for agent status
  });

  const agents = agentsData?.agents || [];

  const connectCommand = "apisentinel connect --server localhost:50051 --token <YOUR_API_KEY>";

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
      {isLoading ? (
        <div className="flex items-center justify-center py-16">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      ) : agents.length === 0 ? (
        <div className="rounded-xl border border-dashed border-border bg-card p-12 text-center space-y-3">
          <div className="flex justify-center">
            <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-muted">
              <WifiOff className="h-7 w-7 text-muted-foreground" />
            </div>
          </div>
          <h3 className="text-lg font-semibold text-foreground">Bağlı Ajan Bulunamadı</h3>
          <p className="text-sm text-muted-foreground max-w-md mx-auto">
            Henüz hiçbir geliştirici ajanı bağlı değil. Yukarıdaki komutu terminalinizde çalıştırarak
            ilk ajanınızı bağlayın.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4">
          {agents.map((agent) => (
            <div
              key={agent.agentId}
              className="rounded-xl border border-border bg-card p-6 shadow-sm flex flex-col justify-between gap-4"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${
                    agent.status === "ONLINE"
                      ? "bg-emerald-500/10 text-emerald-400"
                      : "bg-yellow-500/10 text-yellow-400"
                  }`}>
                    <Laptop className="h-5 w-5" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="text-sm font-bold text-foreground">
                        {agent.hostname || agent.agentId}
                      </h3>
                      <span className={`rounded px-2 py-0.5 text-[11px] font-bold font-mono ${
                        agent.status === "ONLINE"
                          ? "bg-emerald-500/20 text-emerald-400"
                          : "bg-yellow-500/20 text-yellow-400"
                      }`}>
                        {agent.status}
                      </span>
                    </div>
                    <p className="text-xs text-muted-foreground font-mono">
                      ID: {agent.agentId} • OS: {agent.os || "N/A"} • Sürüm: {agent.version || "N/A"}
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
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
