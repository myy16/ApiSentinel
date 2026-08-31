"use client";

import React, { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import { useActiveProject } from "../../../contexts/ProjectContext";
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
  GitBranch,
  GitCommit,
  AlertOctagon,
  ShieldAlert,
  FolderGit2,
} from "lucide-react";

interface AgentSession {
  agentId: string;
  hostname: string;
  os: string;
  version: string;
  status: "ONLINE" | "STALE";
  lastSeen: string;
}

interface APIKeyItem {
  id: string;
  name: string;
  keyPrefix: string;
  createdAt: string;
  isRevoked: boolean;
}

interface AgentScan {
  id: string;
  projectId: string;
  agentId: string;
  repository: string;
  branch: string;
  commitHash: string;
  scanType: string;
  totalFindings: number;
  action: "ALLOW" | "BLOCK" | "WARN";
  createdAt: string;
}

export default function AgentsPage() {
  const { accessToken, organization } = useAuth();
  const { activeProjectId, activeProject } = useActiveProject();
  const queryClient = useQueryClient();
  const [copiedCommand, setCopiedCommand] = useState(false);
  const [copiedHookPreCommit, setCopiedHookPreCommit] = useState(false);
  const [copiedHookPrePush, setCopiedHookPrePush] = useState(false);
  const [keyName, setKeyName] = useState("");
  const [isLiveKey, setIsLiveKey] = useState(false);
  const [createdKey, setCreatedKey] = useState<string | null>(null);

  // Fetch real connected agents from backend
  const { data: agentsData, isLoading } = useQuery({
    queryKey: ["agents", "sessions", organization?.id],
    queryFn: () =>
      apiFetch<{ agents: AgentSession[]; total: number }>("/api/agents/sessions", {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!organization?.id,
    refetchInterval: 10000, // Lightweight poll every 10s for agent status
  });

  const agents = agentsData?.agents || [];

  const { data: keysData } = useQuery({
    queryKey: ["api-keys", activeProjectId],
    queryFn: () =>
      apiFetch<{ keys: APIKeyItem[] }>(`/api/projects/${activeProjectId}/keys`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!organization?.id && !!activeProjectId,
  });

  const { data: scansData, isLoading: isScansLoading } = useQuery({
    queryKey: ["agent-scans", activeProjectId],
    queryFn: () =>
      apiFetch<{ scans: AgentScan[] }>(`/api/projects/${activeProjectId}/scans`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!organization?.id && !!activeProjectId,
  });

  const scans = scansData?.scans || [];

  const createKey = useMutation({
    mutationFn: () =>
      apiFetch<{ apiKey: { secretKey: string } }>(`/api/projects/${activeProjectId}/keys`, {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify({
          name: keyName.trim(),
          keyType: isLiveKey ? "AGENT_LIVE" : "AGENT_TEST",
          expiresInDays: 365,
        }),
      }),
    onSuccess: (data) => {
      setCreatedKey(data.apiKey.secretKey);
      setKeyName("");
      queryClient.invalidateQueries({ queryKey: ["api-keys", activeProjectId] });
    },
  });

  const revokeKey = useMutation({
    mutationFn: (keyId: string) =>
      apiFetch(`/api/projects/${activeProjectId}/keys/${keyId}`, {
        method: "DELETE",
        token: accessToken,
        organizationId: organization?.id,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["api-keys", activeProjectId] });
    },
  });

  const connectCommand = "apisentinel connect --server localhost:50051 --token <YOUR_API_KEY>";

  const copyToClipboard = (text: string, setter: (val: boolean) => void) => {
    navigator.clipboard.writeText(text);
    setter(true);
    setTimeout(() => setter(false), 2000);
  };

  const onlineAgentsCount = agents.filter((a) => a.status === "ONLINE").length;

  return (
    <div className="space-y-8">
      {/* Header */}
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-bold tracking-tight">Local Geliştirici Ajanları & Git Hook Koruması</h1>
            {onlineAgentsCount > 0 ? (
              <span className="flex items-center gap-1 rounded-full bg-emerald-500/10 px-2.5 py-0.5 text-xs font-semibold text-emerald-400">
                <Radio className="h-3 w-3 animate-pulse" />
                gRPC Aktif ({onlineAgentsCount} Ajan Bağlı)
              </span>
            ) : agents.length > 0 ? (
              <span className="flex items-center gap-1 rounded-full bg-yellow-500/10 px-2.5 py-0.5 text-xs font-semibold text-yellow-400">
                <Clock className="h-3 w-3" />
                Ajanlar Yanıt Vermiyor ({agents.length})
              </span>
            ) : (
              <span className="flex items-center gap-1 rounded-full bg-muted px-2.5 py-0.5 text-xs font-semibold text-muted-foreground">
                <WifiOff className="h-3 w-3" />
                Ajan Bekleniyor (gRPC Standby)
              </span>
            )}
          </div>
          <p className="text-sm text-muted-foreground">
            Geliştirici bilgisayarlarında çalışan ApiSentinel CLI ajanlarını, Pre-Commit / Pre-Push hook'larını ve tarama geçmişini yönetin.
          </p>
        </div>
      </div>

      {/* Connection & Git Hook Commands Grid */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Connection Command Card */}
        <div className="rounded-xl border border-border bg-card p-6 shadow-sm space-y-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <Terminal className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-sm font-bold text-foreground">Yeni Ajan Bağla (gRPC Tunnel)</h3>
              <p className="text-xs text-muted-foreground">
                Terminalinizde çalıştırarak local API'nizi Cloud ile çift yönlü tünele bağlayın
              </p>
            </div>
          </div>

          <div className="flex items-center justify-between rounded-lg border border-border bg-background p-3">
            <code className="text-xs font-mono text-foreground truncate mr-2">{connectCommand}</code>
            <button
              onClick={() => copyToClipboard(connectCommand, setCopiedCommand)}
              className="flex items-center gap-1.5 rounded-lg bg-secondary px-3 py-1.5 text-xs font-semibold text-foreground transition hover:bg-primary hover:text-primary-foreground shrink-0"
            >
              {copiedCommand ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
              <span>{copiedCommand ? "Kopyalandı" : "Kopyala"}</span>
            </button>
          </div>
        </div>

        {/* Git Hook Installation Card (#1.2, #10) */}
        <div className="rounded-xl border border-border bg-card p-6 shadow-sm space-y-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-500/10 text-purple-400">
              <FolderGit2 className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-sm font-bold text-foreground">Git Hook Kurulumu (Pre-Commit / Pre-Push)</h3>
              <p className="text-xs text-muted-foreground">
                Repo dizininde hook kurarak gizli anahtarların commitlemesini veya pushlanmasını engelleyin
              </p>
            </div>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between rounded-lg border border-border bg-background p-2.5">
              <div className="text-xs font-mono text-foreground truncate mr-2">
                <span className="text-muted-foreground"># Pre-Commit: </span>
                <span>apisentinel install-hook --type=pre-commit</span>
              </div>
              <button
                onClick={() => copyToClipboard("apisentinel install-hook --type=pre-commit", setCopiedHookPreCommit)}
                className="flex items-center gap-1 rounded bg-secondary px-2.5 py-1 text-xs font-semibold hover:bg-primary hover:text-primary-foreground shrink-0"
              >
                {copiedHookPreCommit ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
                <span>{copiedHookPreCommit ? "Kopyalandı" : "Kopyala"}</span>
              </button>
            </div>

            <div className="flex items-center justify-between rounded-lg border border-border bg-background p-2.5">
              <div className="text-xs font-mono text-foreground truncate mr-2">
                <span className="text-muted-foreground"># Pre-Push: </span>
                <span>apisentinel install-hook --type=pre-push</span>
              </div>
              <button
                onClick={() => copyToClipboard("apisentinel install-hook --type=pre-push", setCopiedHookPrePush)}
                className="flex items-center gap-1 rounded bg-secondary px-2.5 py-1 text-xs font-semibold hover:bg-primary hover:text-primary-foreground shrink-0"
              >
                {copiedHookPrePush ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
                <span>{copiedHookPrePush ? "Kopyalandı" : "Kopyala"}</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Project API key management */}
      <div className="rounded-xl border border-border bg-card p-6 shadow-sm space-y-4">
        <div>
          <h3 className="text-sm font-bold text-foreground">Agent API Key</h3>
          <p className="text-xs text-muted-foreground mt-1">
            {activeProject ? `Aktif proje: ${activeProject.name}` : "Önce bir proje oluştur ve seç."} Bu anahtar yalnızca bu projenin Agent bağlantısı içindir.
          </p>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row">
          <div className="flex flex-1 flex-col gap-1">
            <label htmlFor="agent-key-name" className="text-[11px] font-semibold text-muted-foreground">
              Anahtar adı (zorunlu)
            </label>
            <input
              id="agent-key-name"
              value={keyName}
              onChange={(e) => setKeyName(e.target.value)}
              placeholder="Örn. Production Development Agent"
              maxLength={100}
              className="rounded-lg border border-border bg-background px-3 py-2 text-sm"
            />
          </div>
          <select
            value={isLiveKey ? "live" : "test"}
            onChange={(e) => setIsLiveKey(e.target.value === "live")}
            className="rounded-lg border border-border bg-background px-3 py-2 text-sm"
          >
            <option value="test">Test anahtarı</option>
            <option value="live">Live anahtarı</option>
          </select>
          <button
            onClick={() => createKey.mutate()}
            disabled={!activeProjectId || !keyName.trim() || createKey.isPending}
            className="rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground disabled:opacity-50"
          >
            {createKey.isPending ? "Oluşturuluyor..." : "Yeni anahtar oluştur"}
          </button>
        </div>
        {createdKey && (
          <div className="rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-sm">
            <p className="font-semibold text-amber-300">Bu anahtarı şimdi kopyala. Tekrar gösterilmeyecek.</p>
            <div className="mt-2 flex gap-2">
              <code className="min-w-0 flex-1 break-all rounded bg-background px-2 py-1 text-xs">{createdKey}</code>
              <button onClick={() => navigator.clipboard.writeText(createdKey)} className="rounded bg-secondary px-3 py-1 text-xs">
                Kopyala
              </button>
            </div>
          </div>
        )}
        <div className="space-y-2">
          {(keysData?.keys || [])
            .filter((key) => !key.isRevoked)
            .map((key) => (
              <div key={key.id} className="flex items-center justify-between rounded-lg border border-border px-3 py-2 text-xs">
                <span>
                  <strong>{key.name}</strong> <span className="font-mono text-muted-foreground">({key.keyPrefix}...)</span>
                </span>
                <button onClick={() => revokeKey.mutate(key.id)} className="text-destructive hover:underline">
                  İptal Et
                </button>
              </div>
            ))}
        </div>
      </div>

      {/* Connected Agents List */}
      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      ) : agents.length === 0 ? (
        <div className="rounded-xl border border-dashed border-border bg-card/50 p-12 text-center">
          <Laptop className="mx-auto h-12 w-12 text-muted-foreground/50" />
          <h3 className="mt-4 text-base font-bold text-foreground">Henüz Bağlı Ajan Yok</h3>
          <p className="mt-1 text-xs text-muted-foreground">
            Yukarıdaki komutu local ortamınızda çalıştırarak ilk geliştirici ajanını sisteme bağlayabilirsiniz.
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
                  <div
                    className={`flex h-9 w-9 items-center justify-center rounded-lg ${
                      agent.status === "ONLINE"
                        ? "bg-emerald-500/10 text-emerald-400"
                        : "bg-yellow-500/10 text-yellow-400"
                    }`}
                  >
                    <Laptop className="h-5 w-5" />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <h3 className="text-sm font-bold text-foreground">{agent.hostname || agent.agentId}</h3>
                      <span
                        className={`rounded px-2 py-0.5 text-[11px] font-bold font-mono ${
                          agent.status === "ONLINE"
                            ? "bg-emerald-500/20 text-emerald-400"
                            : "bg-yellow-500/20 text-yellow-400"
                        }`}
                      >
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
                  <span>Git Hook Koruması: <strong>Pre-Commit & Pre-Push Destekli</strong></span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Recent Agent Scans History (#1.3, #1.4) */}
      <div className="rounded-xl border border-border bg-card p-6 shadow-sm space-y-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-500/10 text-purple-400">
              <GitBranch className="h-5 w-5" />
            </div>
            <div>
              <h3 className="text-sm font-bold text-foreground">Son Git & CLI Güvenlik Taramaları</h3>
              <p className="text-xs text-muted-foreground">
                Geliştiricilerin 'apisentinel scan', pre-commit veya pre-push hook ile gerçekleştirdiği tüm tarama kayıtları (0 bulgulu temiz taramalar dahil)
              </p>
            </div>
          </div>
        </div>

        {isScansLoading ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-primary" />
          </div>
        ) : scans.length === 0 ? (
          <div className="rounded-lg border border-dashed border-border bg-background p-8 text-center text-xs text-muted-foreground">
            Henüz kaydedilmiş bir CLI veya Git taraması bulunmuyor. Terminalde 'apisentinel scan' çalıştırarak tarama yapabilirsiniz.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="border-b border-border text-muted-foreground font-semibold">
                <tr>
                  <th className="pb-2">Depo (Repository)</th>
                  <th className="pb-2">Dal / Commit</th>
                  <th className="pb-2">Tarama Tipi</th>
                  <th className="pb-2">Bulgu Durumu</th>
                  <th className="pb-2">Sonuç Kararı</th>
                  <th className="pb-2 text-right">Tarih</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border font-mono">
                {scans.map((sc) => {
                  const isClean = sc.totalFindings === 0;

                  return (
                    <tr key={sc.id} className="hover:bg-secondary/30 transition">
                      <td className="py-2.5 font-bold text-foreground flex items-center gap-1.5">
                        <GitBranch className="h-3.5 w-3.5 text-purple-400" />
                        <span>{sc.repository}</span>
                      </td>
                      <td className="py-2.5 text-muted-foreground">
                        {sc.branch || "HEAD"} {sc.commitHash ? `(${sc.commitHash.slice(0, 7)})` : ""}
                      </td>
                      <td className="py-2.5 text-muted-foreground">
                        <span className="rounded bg-secondary px-2 py-0.5 text-[10px]">{sc.scanType}</span>
                      </td>
                      <td className="py-2.5">
                        {isClean ? (
                          <span className="flex items-center gap-1 text-emerald-400 font-bold">
                            <ShieldCheck className="h-3.5 w-3.5" />
                            <span>0 Bulgu - Temiz (Clean)</span>
                          </span>
                        ) : (
                          <span className="flex items-center gap-1 text-rose-400 font-bold">
                            <AlertOctagon className="h-3.5 w-3.5" />
                            <span>{sc.totalFindings} Tehdit Engellendi</span>
                          </span>
                        )}
                      </td>
                      <td className="py-2.5">
                        <span
                          className={`rounded px-2 py-0.5 text-[10px] font-bold ${
                            sc.action === "BLOCK"
                              ? "bg-rose-500/20 text-rose-400 border border-rose-500/30"
                              : "bg-emerald-500/20 text-emerald-400 border border-emerald-500/30"
                          }`}
                        >
                          {sc.action}
                        </span>
                      </td>
                      <td className="py-2.5 text-right text-muted-foreground">
                        {new Date(sc.createdAt).toLocaleString("tr-TR")}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
