"use client";

import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { apiFetch } from "../../../lib/api";
import { Project } from "@apisentinel/shared";
import {
  FolderGit2,
  Plus,
  ArrowRight,
  Shield,
  Calendar,
  AlertCircle,
  Loader2,
  CheckCircle2,
  Radio,
} from "lucide-react";
import Link from "next/link";

export default function ProjectsPage() {
  const queryClient = useQueryClient();
  const { accessToken, organization } = useAuth();

  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [projectName, setProjectName] = useState("");
  const [createError, setCreateError] = useState<string | null>(null);

  // Fetch projects
  const { data, isLoading, error } = useQuery({
    queryKey: ["projects", organization?.id],
    queryFn: () =>
      apiFetch<{ projects: Project[] }>("/api/projects", {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!organization?.id,
  });

  // Create project mutation
  const createMutation = useMutation({
    mutationFn: (name: string) =>
      apiFetch<Project>("/api/projects", {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify({ name }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["projects", organization?.id] });
      setProjectName("");
      setIsCreateOpen(false);
      setCreateError(null);
    },
    onError: (err: any) => {
      setCreateError(err.message || "Proje oluşturulamadı.");
    },
  });

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!projectName.trim()) return;
    createMutation.mutate(projectName.trim());
  };

  const projects = data?.projects || [];

  return (
    <div className="space-y-8">
      {/* Page Header */}
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Projeler</h1>
          <p className="text-sm text-muted-foreground">
            API ve Webhook servislerinizi projeler halinde organize edin
          </p>
        </div>
        <button
          onClick={() => setIsCreateOpen(true)}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          <span>Yeni Proje Oluştur</span>
        </button>
      </div>

      {/* Create Modal / Form */}
      {isCreateOpen && (
        <div className="rounded-xl border border-border bg-card p-6 shadow-sm animate-in fade-in duration-200">
          <div className="flex items-center justify-between border-b border-border pb-4 mb-4">
            <h3 className="text-base font-semibold">Yeni Proje Oluştur</h3>
            <button
              onClick={() => {
                setIsCreateOpen(false);
                setCreateError(null);
              }}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              Vazgeç
            </button>
          </div>

          {createError && (
            <div className="flex items-center gap-2 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive mb-4">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{createError}</span>
            </div>
          )}

          <form onSubmit={handleCreate} className="flex flex-col gap-4 sm:flex-row sm:items-end">
            <div className="flex-1">
              <label className="block text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">
                Proje Adı
              </label>
              <input
                type="text"
                required
                value={projectName}
                onChange={(e) => setProjectName(e.target.value)}
                placeholder="Örn: Payment Gateway Service"
                className="w-full rounded-lg border border-input bg-background/50 px-3 py-2 text-sm placeholder:text-muted-foreground/60 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
            <button
              type="submit"
              disabled={createMutation.isPending}
              className="flex items-center justify-center gap-2 rounded-lg bg-primary px-5 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
            >
              {createMutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span>Oluşturuluyor...</span>
                </>
              ) : (
                <>
                  <span>Kaydet</span>
                  <CheckCircle2 className="h-4 w-4" />
                </>
              )}
            </button>
          </form>
        </div>
      )}

      {/* Projects Grid */}
      {isLoading ? (
        <div className="flex h-48 items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      ) : projects.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border py-16 text-center">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-secondary text-muted-foreground mb-4">
            <FolderGit2 className="h-6 w-6" />
          </div>
          <h3 className="text-base font-semibold">Henüz proje bulunmuyor</h3>
          <p className="mt-1 text-sm text-muted-foreground max-w-sm">
            Webhook ve API trafiğinizi denetlemeye başlamak için ilk projenizi oluşturun.
          </p>
          <button
            onClick={() => setIsCreateOpen(true)}
            className="mt-6 flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-primary-foreground shadow-sm transition hover:bg-primary/90"
          >
            <Plus className="h-4 w-4" />
            <span>İlk Projeyi Başlat</span>
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {projects.map((project) => (
            <div
              key={project.id}
              className="group flex flex-col justify-between rounded-xl border border-border bg-card p-6 shadow-sm transition hover:border-primary/50 hover:shadow-md"
            >
              <div>
                <div className="flex items-center justify-between mb-4">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <FolderGit2 className="h-5 w-5" />
                  </div>
                  <span className="rounded-full bg-secondary px-2.5 py-0.5 text-[11px] font-medium text-muted-foreground">
                    Aktif
                  </span>
                </div>
                <h3 className="text-base font-semibold text-foreground group-hover:text-primary transition truncate">
                  {project.name}
                </h3>
                <div className="mt-2 flex items-center gap-2 text-xs text-muted-foreground">
                  <Calendar className="h-3.5 w-3.5" />
                  <span>{new Date(project.createdAt).toLocaleDateString("tr-TR")}</span>
                </div>
              </div>

              <div className="mt-6 pt-4 border-t border-border flex items-center justify-between">
                <Link
                  href={`/requests?projectId=${project.id}`}
                  className="flex items-center gap-1.5 text-xs font-semibold text-primary hover:underline"
                >
                  <Radio className="h-3.5 w-3.5" />
                  <span>Canlı İstekler</span>
                </Link>
                <ArrowRight className="h-4 w-4 text-muted-foreground transition group-hover:translate-x-1 group-hover:text-primary" />
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
