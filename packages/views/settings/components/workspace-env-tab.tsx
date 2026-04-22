"use client";

import { useState } from "react";
import {
  Loader2,
  Save,
  Plus,
  Trash2,
  Eye,
  EyeOff,
  Lock,
} from "lucide-react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@multica/core/auth";
import { useWorkspaceId } from "@multica/core/hooks";
import { useCurrentWorkspace } from "@multica/core/paths";
import { api } from "@multica/core/api";
import { memberListOptions, workspaceKeys } from "@multica/core/workspace/queries";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import { Label } from "@multica/ui/components/ui/label";
import { toast } from "sonner";

let nextEnvId = 0;

interface EnvEntry {
  id: number;
  key: string;
  value: string;
  visible: boolean;
}

function envMapToEntries(env: Record<string, string>): EnvEntry[] {
  return Object.entries(env).map(([key, value]) => ({
    id: nextEnvId++,
    key,
    value,
    visible: false,
  }));
}

function entriesToEnvMap(entries: EnvEntry[]): Record<string, string> {
  const map: Record<string, string> = {};
  for (const entry of entries) {
    const key = entry.key.trim();
    if (key) {
      map[key] = entry.value;
    }
  }
  return map;
}

export function WorkspaceEnvTab() {
  const workspace = useCurrentWorkspace();
  const member = useCurrentMember();
  const qc = useQueryClient();

  const readOnly = !member || member.role === "member";
  const wsEnv = workspace?.custom_env ?? {};

  const [envEntries, setEnvEntries] = useState<EnvEntry[]>(
    envMapToEntries(wsEnv),
  );
  const [saving, setSaving] = useState(false);

  const currentEnvMap = entriesToEnvMap(envEntries);
  const dirty = JSON.stringify(currentEnvMap) !== JSON.stringify(wsEnv);

  const addEnvEntry = () => {
    setEnvEntries([
      ...envEntries,
      { id: nextEnvId++, key: "", value: "", visible: true },
    ]);
  };

  const removeEnvEntry = (index: number) => {
    setEnvEntries(envEntries.filter((_, i) => i !== index));
  };

  const updateEnvEntry = (index: number, field: "key" | "value", val: string) => {
    setEnvEntries(
      envEntries.map((entry, i) =>
        i === index ? { ...entry, [field]: val } : entry,
      ),
    );
  };

  const toggleEnvVisibility = (index: number) => {
    setEnvEntries(
      envEntries.map((entry, i) =>
        i === index ? { ...entry, visible: !entry.visible } : entry,
      ),
    );
  };

  const handleSave = async () => {
    if (!workspace) return;

    const keys = envEntries.filter((e) => e.key.trim()).map((e) => e.key.trim());
    if (new Set(keys).size < keys.length) {
      toast.error("Duplicate environment variable keys");
      return;
    }

    setSaving(true);
    try {
      await api.updateWorkspace(workspace.id, { custom_env: currentEnvMap });
      qc.invalidateQueries({ queryKey: workspaceKeys.list() });
      toast.success("Environment variables saved");
    } catch {
      toast.error("Failed to save environment variables");
    } finally {
      setSaving(false);
    }
  };

  if (readOnly) {
    return (
      <div className="max-w-lg space-y-4">
        <div>
          <Label className="text-xs text-muted-foreground">
            Workspace Environment Variables
          </Label>
          <p className="text-xs text-muted-foreground mt-0.5">
            Shared across all agents. Agent-level vars take precedence on conflict. Values are hidden — only workspace owners and admins can view and edit them.
          </p>
        </div>
        {envEntries.length > 0 ? (
          <div className="space-y-2">
            {envEntries.map((entry) => (
              <div key={entry.id} className="flex items-center gap-2">
                <Input
                  value={entry.key}
                  readOnly
                  className="w-[40%] font-mono text-xs bg-muted"
                />
                <div className="relative flex-1">
                  <Input
                    type="password"
                    value="****"
                    readOnly
                    className="pr-8 font-mono text-xs bg-muted"
                  />
                  <Lock className="absolute right-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
                </div>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-xs text-muted-foreground italic">No environment variables configured.</p>
        )}
      </div>
    );
  }

  return (
    <div className="max-w-lg space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <Label className="text-xs text-muted-foreground">
            Workspace Environment Variables
          </Label>
          <p className="text-xs text-muted-foreground mt-0.5">
            Shared across all agents. Agent-level vars take precedence on conflict.
          </p>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={addEnvEntry}
          className="h-7 gap-1 text-xs"
        >
          <Plus className="h-3 w-3" />
          Add
        </Button>
      </div>
      {envEntries.length > 0 && (
        <div className="space-y-2">
          {envEntries.map((entry, index) => (
            <div key={entry.id} className="flex items-center gap-2">
              <Input
                value={entry.key}
                onChange={(e) => updateEnvEntry(index, "key", e.target.value)}
                placeholder="KEY"
                className="w-[40%] font-mono text-xs"
              />
              <div className="relative flex-1">
                <Input
                  type={entry.visible ? "text" : "password"}
                  value={entry.value}
                  onChange={(e) => updateEnvEntry(index, "value", e.target.value)}
                  placeholder="value"
                  className="pr-8 font-mono text-xs"
                />
                <button
                  type="button"
                  onClick={() => toggleEnvVisibility(index)}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  {entry.visible ? (
                    <EyeOff className="h-3.5 w-3.5" />
                  ) : (
                    <Eye className="h-3.5 w-3.5" />
                  )}
                </button>
              </div>
              <button
                type="button"
                onClick={() => removeEnvEntry(index)}
                className="shrink-0 text-muted-foreground hover:text-destructive"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
        </div>
      )}

      <Button onClick={handleSave} disabled={!dirty || saving} size="sm">
        {saving ? (
          <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
        ) : (
          <Save className="h-3.5 w-3.5 mr-1.5" />
        )}
        Save
      </Button>
    </div>
  );
}
