"use client";

import React, { createContext, useContext, useEffect, useState, useCallback } from "react";
import { apiFetch } from "../lib/api";
import { User, Organization } from "@apisentinel/shared";

interface AuthState {
  user: User | null;
  organization: Organization | null;
  accessToken: string | null;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, organizationName?: string) => Promise<void>;
  logout: () => void;
  setOrganization: (org: Organization) => void;
}

const AuthContext = createContext<AuthState | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [organization, setOrganization] = useState<Organization | null>(null);
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Initialize auth from localStorage on client
  useEffect(() => {
    try {
      const storedToken = localStorage.getItem("apisentinel_access_token");
      const storedUser = localStorage.getItem("apisentinel_user");
      const storedOrg = localStorage.getItem("apisentinel_org");

      if (storedToken && storedUser) {
        setAccessToken(storedToken);
        setUser(JSON.parse(storedUser));
        if (storedOrg) {
          setOrganization(JSON.parse(storedOrg));
        }
      }
    } catch (e) {
      console.error("Failed to restore auth state", e);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const data = await apiFetch<{
      user: User;
      organization: Organization | null;
      accessToken: string;
      refreshToken: string;
    }>("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });

    setAccessToken(data.accessToken);
    setUser(data.user);
    setOrganization(data.organization);

    localStorage.setItem("apisentinel_access_token", data.accessToken);
    localStorage.setItem("apisentinel_refresh_token", data.refreshToken);
    localStorage.setItem("apisentinel_user", JSON.stringify(data.user));
    if (data.organization) {
      localStorage.setItem("apisentinel_org", JSON.stringify(data.organization));
    }
  }, []);

  const register = useCallback(async (email: string, password: string, organizationName?: string) => {
    const data = await apiFetch<{
      user: User;
      organization: Organization;
      accessToken: string;
      refreshToken: string;
    }>("/api/auth/register", {
      method: "POST",
      body: JSON.stringify({ email, password, organizationName }),
    });

    setAccessToken(data.accessToken);
    setUser(data.user);
    setOrganization(data.organization);

    localStorage.setItem("apisentinel_access_token", data.accessToken);
    localStorage.setItem("apisentinel_refresh_token", data.refreshToken);
    localStorage.setItem("apisentinel_user", JSON.stringify(data.user));
    localStorage.setItem("apisentinel_org", JSON.stringify(data.organization));
  }, []);

  const logout = useCallback(() => {
    setAccessToken(null);
    setUser(null);
    setOrganization(null);

    localStorage.removeItem("apisentinel_access_token");
    localStorage.removeItem("apisentinel_refresh_token");
    localStorage.removeItem("apisentinel_user");
    localStorage.removeItem("apisentinel_org");
  }, []);

  const selectOrg = useCallback((org: Organization) => {
    setOrganization(org);
    localStorage.setItem("apisentinel_org", JSON.stringify(org));
  }, []);

  return (
    <AuthContext.Provider
      value={{
        user,
        organization,
        accessToken,
        isLoading,
        login,
        register,
        logout,
        setOrganization: selectOrg,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
