"use client";

import React, { createContext, useContext, useState, useEffect, useMemo } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "../hooks/useAuth";
import { apiFetch } from "../lib/api";
import { Project } from "@apisentinel/shared";

interface ProjectContextType {
  projects: Project[];
  activeProjectId: string;
  activeProject: Project | null;
  isLoading: boolean;
  setActiveProjectId: (id: string) => void;
  refetchProjects: () => void;
}

const ProjectContext = createContext<ProjectContextType | undefined>(undefined);

const STORAGE_KEY = "apisentinel_active_project_id";

export function ProjectProvider({ children }: { children: React.ReactNode }) {
  const { accessToken, organization } = useAuth();
  const queryClient = useQueryClient();

  const [activeProjectId, setActiveProjectIdState] = useState<string>("");

  // Fetch projects from backend
  const { data: projectsData, isLoading, refetch } = useQuery({
    queryKey: ["projects", organization?.id],
    queryFn: () =>
      apiFetch<{ projects: Project[] }>("/api/projects", {
        token: accessToken,
        organizationId: organization?.id,
      }),
    enabled: !!accessToken && !!organization?.id,
  });

  const projects = useMemo(() => projectsData?.projects || [], [projectsData]);

  // Sync with localStorage on initial mount / projects change
  useEffect(() => {
    if (projects.length === 0) return;

    const storedId = typeof window !== "undefined" ? localStorage.getItem(STORAGE_KEY) : null;
    const exists = projects.find((p) => p.id === storedId);

    if (storedId && exists) {
      if (activeProjectId !== storedId) {
        setActiveProjectIdState(storedId);
      }
    } else {
      // Default to first project and persist
      const firstId = projects[0]?.id || "";
      setActiveProjectIdState(firstId);
      if (typeof window !== "undefined" && firstId) {
        localStorage.setItem(STORAGE_KEY, firstId);
      }
    }
  }, [projects]);

  const setActiveProjectId = (id: string) => {
    setActiveProjectIdState(id);
    if (typeof window !== "undefined") {
      localStorage.setItem(STORAGE_KEY, id);
    }
  };

  const activeProject = useMemo(
    () => projects.find((p) => p.id === activeProjectId) || projects[0] || null,
    [projects, activeProjectId]
  );

  return (
    <ProjectContext.Provider
      value={{
        projects,
        activeProjectId,
        activeProject,
        isLoading,
        setActiveProjectId,
        refetchProjects: refetch,
      }}
    >
      {children}
    </ProjectContext.Provider>
  );
}

export function useActiveProject() {
  const context = useContext(ProjectContext);
  if (!context) {
    throw new Error("useActiveProject must be used within a ProjectProvider");
  }
  return context;
}
