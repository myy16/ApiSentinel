"use client";

import React, { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../../../hooks/useAuth";
import { useActiveProject } from "../../../contexts/ProjectContext";
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
  Edit2,
  Trash2,
  Check,
  Search,
  Zap,
  Globe,
  ShieldAlert,
  X,
  Settings2,
} from "lucide-react";
import Link from "next/link";

export default function ProjectsPage() {
  const queryClient = useQueryClient();
  const { accessToken, organization } = useAuth();
  const { projects, activeProjectId, setActiveProjectId, isLoading } = useActiveProject();

  const [searchQuery, setSearchQuery] = useState("");
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [projectName, setProjectName] = useState("");
  const [createError, setCreateError] = useState<string | null>(null);

  // Edit State
  const [editingProject, setEditingProject] = useState<Project | null>(null);
  const [editName, setEditName] = useState("");
  const [editError, setEditError] = useState<string | null>(null);

  // Create project mutation
  const createMutation = useMutation({
    mutationFn: (name: string) =>
      apiFetch<Project>("/api/projects", {
        method: "POST",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify({ name }),
      }),
    onSuccess: (newProj) => {
      queryClient.invalidateQueries({ queryKey: ["projects", organization?.id] });
      setProjectName("");
      setIsCreateOpen(false);
      setCreateError(null);
      if (newProj?.id) {
        setActiveProjectId(newProj.id);
      }
    },
    onError: (err: any) => {
      setCreateError(err.message || "Proje oluşturulamadı.");
    },
  });

  // Update (Rename) project mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      apiFetch<Project>(`/api/projects/${id}`, {
        method: "PUT",
        token: accessToken,
        organizationId: organization?.id,
        body: JSON.stringify({ name }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["projects", organization?.id] });
      setEditingProject(null);
      setEditError(null);
    },
    onError: (err: any) => {
      setEditError(err.message || "Proje güncellenemedi.");
    },
  });

  // Delete project mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) =>
      apiFetch(`/api/projects/${id}`, {
        method: "DELETE",
        token: accessToken,
        organizationId: organization?.id,
      }),
    onSuccess: (_, deletedId) => {
      queryClient.invalidateQueries({ queryKey: ["projects", organization?.id] });
      if (activeProjectId === deletedId) {
        const remaining = projects.filter((p) => p.id !== deletedId);
        if (remaining.length > 0) {
          setActiveProjectId(remaining[0].id);
        }
      }
    },
  });

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!projectName.trim()) return;
    createMutation.mutate(projectName.trim());
  };

  const handleStartEdit = (p: Project) => {
    setEditingProject(p);
    setEditName(p.name);
    setEditError(null);
  };

  const handleSaveEdit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingProject || !editName.trim()) return;
    updateMutation.mutate({ id: editingProject.id, name: editName.trim() });
  };

  const filteredProjects = projects.filter((p) => {
    if (!searchQuery) return true;
    return p.name.toLowerCase().includes(searchQuery.toLowerCase());
  });

  return (
    <div className="space-y-8">
      {/* Page Header */}
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">Projeler & Çalışma Alanları</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Webhook ve API servislerinizi projeler altında izole edin; aktif çalışma alanınızı tek tıkla yönetin.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={() => setIsCreateOpen(true)}
            className="flex items-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-xs font-bold text-primary-foreground shadow-sm transition hover:bg-primary/90"
          >
            <Plus className="h-4 w-4" />
            <span>Yeni Proje Oluştur</span>
          </button>
        </div>
      </div>

      {/* Search Filter Bar */}
      {projects.length > 2 && (
        <div className="relative flex items-center max-w-md">
          <Search className="absolute left-3.5 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Proje adına göre ara..."
            className="w-full rounded-xl border border-border bg-card py-2 pl-9 pr-3 text-xs focus:outline-none focus:ring-2 focus:ring-primary"
          />
        </div>
      )}

      {/* Create Modal / Form */}
      {isCreateOpen && (
        <div className="rounded-2xl border border-border bg-card p-6 shadow-md animate-in fade-in duration-200">
          <div className="flex items-center justify-between border-b border-border pb-4 mb-4">
            <h3 className="text-base font-bold text-foreground">Yeni Proje Tanımla</h3>
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
            <div className="flex items-center gap-2 rounded-xl border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive mb-4">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{createError}</span>
            </div>
          )}

          <form onSubmit={handleCreate} className="flex flex-col gap-4 sm:flex-row sm:items-end">
            <div className="flex-1">
              <label className="block text-[10px] font-bold uppercase tracking-wider text-muted-foreground mb-1.5">
                Proje Adı
              </label>
              <input
                type="text"
                required
                value={projectName}
                onChange={(e) => setProjectName(e.target.value)}
                placeholder="Örn: E-Ticaret Ödeme Servisi (Production)"
                className="w-full rounded-xl border border-input bg-background/50 px-3 py-2 text-sm placeholder:text-muted-foreground/60 focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
            <button
              type="submit"
              disabled={createMutation.isPending}
              className="flex items-center justify-center gap-2 rounded-xl bg-primary px-5 py-2.5 text-xs font-bold text-primary-foreground shadow-sm transition hover:bg-primary/90 disabled:opacity-50"
            >
              {createMutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span>Oluşturuluyor...</span>
                </>
              ) : (
                <>
                  <span>Projeyi Başlat</span>
                  <Zap className="h-4 w-4" />
                </>
              )}
            </button>
          </form>
        </div>
      )}

      {/* Edit Modal / Drawer */}
      {editingProject && (
        <div className="rounded-2xl border border-primary/40 bg-card p-6 shadow-md animate-in fade-in duration-200 space-y-4">
          <div className="flex items-center justify-between border-b border-border pb-4">
            <div className="flex items-center gap-2">
              <Settings2 className="h-5 w-5 text-primary" />
              <h3 className="text-base font-bold text-foreground">Projeyi Düzenle</h3>
            </div>
            <button
              onClick={() => setEditingProject(null)}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          {editError && (
            <div className="flex items-center gap-2 rounded-xl border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{editError}</span>
            </div>
          )}

          <form onSubmit={handleSaveEdit} className="flex flex-col gap-4 sm:flex-row sm:items-end">
            <div className="flex-1">
              <label className="block text-[10px] font-bold uppercase tracking-wider text-muted-foreground mb-1.5">
                Proje Yeni Adı
              </label>
              <input
                type="text"
                required
                value={editName}
                onChange={(e) => setEditName(e.target.value)}
                className="w-full rounded-xl border border-input bg-background/50 px-3 py-2 text-sm focus:border-primary focus:outline-none"
              />
            </div>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setEditingProject(null)}
                className="rounded-xl border border-border px-4 py-2 text-xs font-semibold text-muted-foreground hover:text-foreground"
              >
                İptal
              </button>
              <button
                type="submit"
                disabled={updateMutation.isPending}
                className="flex items-center gap-2 rounded-xl bg-primary px-5 py-2 text-xs font-bold text-primary-foreground shadow-sm hover:bg-primary/90 disabled:opacity-50"
              >
                {updateMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
                <span>Kaydet</span>
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Projects Grid */}
      {isLoading ? (
        <div className="flex h-48 items-center justify-center">
          <Loader2 className="h-6 w-6 animate-spin text-primary" />
        </div>
      ) : projects.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border py-16 text-center bg-card/40">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-secondary text-muted-foreground mb-4">
            <FolderGit2 className="h-6 w-6" />
          </div>
          <h3 className="text-base font-bold text-foreground">Henüz proje bulunmuyor</h3>
          <p className="mt-1 text-sm text-muted-foreground max-w-sm">
            Webhook ve API trafiğinizi denetlemeye başlamak için ilk projenizi oluşturun.
          </p>
          <button
            onClick={() => setIsCreateOpen(true)}
            className="mt-6 flex items-center gap-2 rounded-xl bg-primary px-4 py-2 text-xs font-bold text-primary-foreground shadow-sm transition hover:bg-primary/90"
          >
            <Plus className="h-4 w-4" />
            <span>İlk Projeyi Başlat</span>
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {filteredProjects.map((project) => {
            const isActive = project.id === activeProjectId;

            return (
              <div
                key={project.id}
                className={`group flex flex-col justify-between rounded-2xl border p-6 shadow-sm transition space-y-5 glow-card ${
                  isActive
                    ? "border-primary/50 bg-primary/5 ring-1 ring-primary/30"
                    : "border-border bg-card hover:border-border/80"
                }`}
              >
                <div>
                  <div className="flex items-center justify-between mb-4">
                    <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10 text-primary border border-primary/20">
                      <FolderGit2 className="h-5 w-5" />
                    </div>

                    <div className="flex items-center gap-2">
                      {isActive ? (
                        <span className="flex items-center gap-1 rounded-full bg-emerald-500/10 border border-emerald-500/20 px-2.5 py-0.5 text-[10px] font-bold text-emerald-400">
                          <Check className="h-3 w-3" />
                          AKTİF ÇALIŞMA ALANI
                        </span>
                      ) : (
                        <button
                          onClick={() => setActiveProjectId(project.id)}
                          className="rounded-full border border-border bg-background px-2.5 py-0.5 text-[10px] font-semibold text-muted-foreground hover:text-foreground hover:border-primary/40 transition"
                          title="Bu projeyi genel çalışma alanı olarak seç"
                        >
                          Seç
                        </button>
                      )}
                    </div>
                  </div>

                  <h3 className="text-base font-bold text-foreground group-hover:text-primary transition truncate">
                    {project.name}
                  </h3>

                  <div className="mt-2 flex items-center gap-2 text-xs text-muted-foreground">
                    <Calendar className="h-3.5 w-3.5" />
                    <span>{new Date(project.createdAt).toLocaleDateString("tr-TR")}</span>
                  </div>
                </div>

                {/* Quick links to Endpoints, Requests, Security */}
                <div className="space-y-3 pt-3 border-t border-border">
                  <div className="grid grid-cols-3 gap-2 text-center text-xs">
                    <Link
                      href="/endpoints"
                      onClick={() => setActiveProjectId(project.id)}
                      className="flex flex-col items-center justify-center rounded-xl border border-border/80 bg-background/60 p-2 hover:bg-secondary transition text-muted-foreground hover:text-foreground"
                    >
                      <Globe className="h-3.5 w-3.5 text-blue-400 mb-1" />
                      <span className="text-[10px] font-semibold">Endpoints</span>
                    </Link>

                    <Link
                      href="/requests"
                      onClick={() => setActiveProjectId(project.id)}
                      className="flex flex-col items-center justify-center rounded-xl border border-border/80 bg-background/60 p-2 hover:bg-secondary transition text-muted-foreground hover:text-foreground"
                    >
                      <Radio className="h-3.5 w-3.5 text-emerald-400 mb-1" />
                      <span className="text-[10px] font-semibold">İstekler</span>
                    </Link>

                    <Link
                      href="/security"
                      onClick={() => setActiveProjectId(project.id)}
                      className="flex flex-col items-center justify-center rounded-xl border border-border/80 bg-background/60 p-2 hover:bg-secondary transition text-muted-foreground hover:text-foreground"
                    >
                      <ShieldAlert className="h-3.5 w-3.5 text-rose-400 mb-1" />
                      <span className="text-[10px] font-semibold">Güvenlik</span>
                    </Link>
                  </div>

                  {/* Actions: Edit & Delete */}
                  <div className="flex items-center justify-between pt-2">
                    <button
                      onClick={() => handleStartEdit(project)}
                      className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground font-semibold"
                    >
                      <Edit2 className="h-3.5 w-3.5" />
                      <span>Yeniden Adlandır</span>
                    </button>

                    <button
                      onClick={() => {
                        if (
                          confirm(
                            `"${project.name}" projesini ve ilişkili tüm endpoint/istek verilerini silmek istediğinize emin misiniz?`
                          )
                        ) {
                          deleteMutation.mutate(project.id);
                        }
                      }}
                      className="flex items-center gap-1 text-xs text-destructive hover:opacity-80 font-semibold"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                      <span>Sil</span>
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
