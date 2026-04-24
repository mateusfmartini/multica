"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "../api";
import { useWSEvent } from "../realtime";

export function useIssueActiveTask(issueId: string): { isAgentRunning: boolean } {
  const [isAgentRunning, setIsAgentRunning] = useState(false);

  useEffect(() => {
    let cancelled = false;
    api.getActiveTasksForIssue(issueId).then(({ tasks }) => {
      if (cancelled) return;
      setIsAgentRunning(tasks.some((t) => t.status === "running" || t.status === "dispatched"));
    }).catch(() => {});
    return () => { cancelled = true; };
  }, [issueId]);

  useWSEvent(
    "task:dispatch",
    useCallback((payload: unknown) => {
      const p = payload as { issue_id?: string };
      if (p.issue_id && p.issue_id !== issueId) return;
      setIsAgentRunning(true);
    }, [issueId]),
  );

  const handleTaskEnd = useCallback((payload: unknown) => {
    const p = payload as { issue_id: string };
    if (p.issue_id !== issueId) return;
    api.getActiveTasksForIssue(issueId).then(({ tasks }) => {
      setIsAgentRunning(tasks.some((t) => t.status === "running" || t.status === "dispatched"));
    }).catch(() => {});
  }, [issueId]);

  useWSEvent("task:completed", handleTaskEnd);
  useWSEvent("task:failed", handleTaskEnd);
  useWSEvent("task:cancelled", handleTaskEnd);

  return { isAgentRunning };
}
