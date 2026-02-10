import { useCallback, useEffect, useState } from "react";

export type HeartbeatEvent = {
  ts: number;
  status: "sent" | "ok-empty" | "ok-token" | "skipped" | "failed";
  preview?: string;
  durationMs?: number;
  reason?: string;
};

export function useHeartbeat() {
  const [enabled, setEnabled] = useState(true);
  const [lastEvent, setLastEvent] = useState<HeartbeatEvent | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const event = (await window.electronAPI.heartbeat.last()) as HeartbeatEvent | null;
      setLastEvent(event);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
    const timer = setInterval(() => {
      void refresh();
    }, 15000);
    return () => clearInterval(timer);
  }, [refresh]);

  const toggleEnabled = useCallback(async () => {
    const next = !enabled;
    const result = await window.electronAPI.heartbeat.setEnabled(next);
    if (result.ok) {
      setEnabled(next);
    } else {
      setError(result.error ?? "Failed to update heartbeat setting");
    }
  }, [enabled]);

  const wakeNow = useCallback(async () => {
    setLoading(true);
    try {
      const result = await window.electronAPI.heartbeat.wake("manual");
      if (!result.ok) {
        setError(result.error ?? "Failed to run heartbeat");
      }
      await refresh();
    } finally {
      setLoading(false);
    }
  }, [refresh]);

  return {
    enabled,
    lastEvent,
    loading,
    error,
    refresh,
    toggleEnabled,
    wakeNow,
  };
}
