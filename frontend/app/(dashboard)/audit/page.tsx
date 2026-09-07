"use client";

import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { useActiveProject } from "../../../contexts/ProjectContext";
import { apiFetch } from "../../../lib/api";
import { AuditLog } from "@apisentinel/shared";
import {
  History,
  Shield,
  Clock,
  User,
  Search,
  Filter,
  RefreshCw,
  FileText,
  AlertTriangle,
  RotateCcw,
  Key,
  Globe,
  Loader2,
} from "lucide-react";

export default function AuditLogsPage() {
  const { accessToken, organization } = useAuth();
  const { activeProjectId } = useActiveProject();

  const [searchQuery, setSearchQuery] = useState("");
  const [actionFilter, setActionFilter] = useState("ALL");

  const {
    data: auditData,
    isLoading,
    refetch,
    isRefetching,
  } = useQuery({
    queryKey: ["audit-logs", activeProjectId],
    queryFn: () =>
      apiFetch<{ auditLogs: AuditLog[] }>(`/api/projects/${activeProjectId}/audit-logs?limit=100`, {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!activeProjectId && !!organization?.id,
    refetchInterval: 3000,
  });

  const logs = auditData?.auditLogs || [];

  const filteredLogs = logs.filter((log) => {
    if (actionFilter !== "ALL" && log.action !== actionFilter) {
      return false;
    }
    if (searchQuery) {
      const q = searchQuery.toLowerCase();
      return (
        log.action.toLowerCase().includes(q) ||
        log.resourceType.toLowerCase().includes(q) ||
        log.resourceId.toLowerCase().includes(q) ||
        (log.justification && log.justification.toLowerCase().includes(q))
      );
    }
    return true;
  });

  const getActionIcon = (action: string) => {
    if (action.includes("REPLAY")) {
      return <RotateCcw className="h-4 w-4 text-amber-400" />;
    }
    if (action.includes("SECRET") || action.includes("KEY")) {
      return <Key className="h-4 w-4 text-rose-400" />;
    }
    if (action.includes("ENDPOINT") || action.includes("URL")) {
      return <Globe className="h-4 w-4 text-primary" />;
    }
    return <Shield className="h-4 w-4 text-emerald-400" />;
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        <div>
          <div className="inline-flex items-center gap-2 rounded-full border border-primary/30 bg-primary/10 px-3 py-1 text-xs font-semibold text-primary mb-2">
            <History className="h-3.5 w-3.5" />
            <span>Audit & Compliance Trail</span>
          </div>
          <h1 className="text-2xl font-extrabold tracking-tight">Güvenlik ve Operasyon Denetim Kayıtları</h1>
          <p className="text-xs md:text-sm text-muted-foreground">
            Idempotency override işlemleri, replay tetiklemeleri, secret ve upstream URL değişikliklerinin kim tarafından hangi gerekçeyle yapıldığını denetleyin.
          </p>
        </div>
        <button
          onClick={() => refetch()}
          className="inline-flex items-center gap-2 px-3 py-2 rounded-xl border border-border bg-card hover:bg-muted text-xs font-semibold transition shadow-sm self-start"
        >
          <RefreshCw className="h-3.5 w-3.5 text-primary" />
          <span>Yenile</span>
        </button>
      </div>

      {/* Filter Bar */}
      <div className="flex flex-col md:flex-row gap-3 items-center justify-between bg-card/40 p-3 rounded-2xl border border-border">
        <div className="flex items-center gap-2 w-full md:w-auto">
          <select
            value={actionFilter}
            onChange={(e) => setActionFilter(e.target.value)}
            className="rounded-xl border border-border bg-card px-3 py-1.5 text-xs font-semibold text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="ALL">Tüm Eylemler</option>
            <option value="REPLAY_IDEMPOTENCY_OVERRIDDEN">Replay Idempotency Override (Riskli)</option>
            <option value="REPLAY_EXECUTED">Standart Replay</option>
            <option value="REPLAY_LAB_EXECUTED">Replay Lab Tetiklemesi</option>
            <option value="SECRET_UPDATED">Secret Güncelleme</option>
            <option value="UPSTREAM_URL_CHANGED">Upstream URL Değişikliği</option>
          </select>
        </div>

        <div className="relative w-full md:w-72">
          <Search className="absolute left-3 top-2.5 h-3.5 w-3.5 text-muted-foreground" />
          <input
            type="text"
            placeholder="Aksiyon, Resource ID veya Gerekçe ara..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full rounded-xl border border-border bg-card pl-9 pr-3 py-1.5 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>
      </div>

      {/* Audit Log Timeline */}
      <div className="rounded-2xl border border-border bg-card/60 overflow-hidden shadow-sm">
        <div className="px-4 py-3 border-b border-border bg-card/80 flex items-center justify-between">
          <span className="text-xs font-bold text-foreground">
            Denetim Olayları ({filteredLogs.length})
          </span>
          <span className="text-[11px] text-muted-foreground">
            Değiştirilemez (Immutable) Kayıtlar
          </span>
        </div>

        {isLoading ? (
          <div className="p-12 text-center text-muted-foreground flex flex-col items-center gap-2">
            <Loader2 className="h-6 w-6 animate-spin text-primary" />
            <span className="text-xs">Audit kayıtları yükleniyor...</span>
          </div>
        ) : filteredLogs.length === 0 ? (
          <div className="p-12 text-center text-muted-foreground">
            <History className="h-8 w-8 mx-auto text-muted-foreground/40 mb-2" />
            <p className="text-sm font-semibold">Henüz denetim kaydı bulunmuyor</p>
            <p className="text-xs text-muted-foreground mt-1">
              Kritik operasyonlar (Replay override, yetki ve ayar değişiklikleri) burada listelenir.
            </p>
          </div>
        ) : (
          <div className="divide-y divide-border">
            {filteredLogs.map((log) => (
              <div key={log.id} className="p-4 flex flex-col sm:flex-row sm:items-start justify-between gap-3 hover:bg-muted/30 transition">
                <div className="flex items-start gap-3">
                  <div className="p-2 rounded-xl bg-muted border border-border mt-0.5">
                    {getActionIcon(log.action)}
                  </div>
                  <div className="space-y-1">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-xs font-bold font-mono text-foreground">
                        {log.action}
                      </span>
                      <span className="text-[10px] px-2 py-0.5 rounded bg-primary/10 text-primary font-bold">
                        {log.resourceType}
                      </span>
                    </div>

                    <div className="text-xs text-muted-foreground">
                      <span className="font-mono text-foreground/80">{log.resourceId}</span>
                    </div>

                    {log.justification && (
                      <div className="p-2.5 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-300 text-xs mt-1">
                        <strong>Gerekçe:</strong> {log.justification}
                      </div>
                    )}
                  </div>
                </div>

                <div className="text-right text-[11px] text-muted-foreground shrink-0 space-y-1">
                  <div>{new Date(log.createdAt).toLocaleTimeString()} · {new Date(log.createdAt).toLocaleDateString()}</div>
                  {log.ipAddress && <div className="font-mono text-[10px]">IP: {log.ipAddress}</div>}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
