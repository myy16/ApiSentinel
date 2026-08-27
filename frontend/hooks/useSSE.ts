"use client";

import { useEffect, useRef, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:3001";

interface UseSSEOptions {
  /** Project ID to subscribe to */
  projectId: string | null;
  /** Auth token for the SSE endpoint */
  token: string | null;
	/** Verified organization context required by the API tenant guard */
  organizationId: string | null;
  /** React Query keys to invalidate on new events */
  queryKeys?: string[][];
  /** Custom event handler */
  onEvent?: (event: string, data: unknown) => void;
  /** Whether the hook is enabled */
  enabled?: boolean;
}

/**
 * Custom hook for Server-Sent Events (SSE) integration with the backend.
 * Replaces polling (refetchInterval) with real-time push notifications.
 *
 * Automatically reconnects on connection loss with exponential backoff.
 * Invalidates React Query caches when new events arrive.
 */
export function useSSE({
  projectId,
  token,
  organizationId,
  queryKeys = [],
  onEvent,
  enabled = true,
}: UseSSEOptions) {
  const queryClient = useQueryClient();
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptRef = useRef(0);
  const maxReconnectDelay = 30_000; // 30 seconds max

  const connect = useCallback(() => {
    if (!projectId || !token || !organizationId || !enabled) return;

    // Close existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    // EventSource doesn't natively support auth headers.
    // Pass token as query parameter (backend SSE endpoint is already auth-protected via middleware).
    const url = `${API_BASE_URL}/api/projects/${projectId}/events/stream?token=${encodeURIComponent(token)}&organizationId=${encodeURIComponent(organizationId)}`;

    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.addEventListener("connected", () => {
      // Reset reconnect counter on successful connection
      reconnectAttemptRef.current = 0;
    });

    es.addEventListener("request.created", (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data);

        // Invalidate specified query keys so React Query refetches
        for (const key of queryKeys) {
          queryClient.invalidateQueries({ queryKey: key });
        }

        // Call custom handler
        onEvent?.("request.created", data);
      } catch {
        // Silently ignore parse errors for malformed events
      }
    });

    es.addEventListener("finding.created", (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data);
        queryClient.invalidateQueries({ queryKey: ["findings", projectId] });
        onEvent?.("finding.created", data);
      } catch {
        // Silently ignore parse errors
      }
    });

    es.onerror = () => {
      es.close();
      eventSourceRef.current = null;

      // Exponential backoff reconnect
      const delay = Math.min(
        1000 * Math.pow(2, reconnectAttemptRef.current),
        maxReconnectDelay
      );
      reconnectAttemptRef.current++;

      reconnectTimeoutRef.current = setTimeout(() => {
        connect();
      }, delay);
    };
  }, [projectId, token, organizationId, enabled, queryClient, queryKeys, onEvent]);

  useEffect(() => {
    connect();

    return () => {
      // Cleanup on unmount
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
    };
  }, [connect]);
}
