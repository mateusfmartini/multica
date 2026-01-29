import { spawn, type ChildProcess } from "child_process";
import { Type } from "@sinclair/typebox";
import type { AgentTool } from "@mariozechner/pi-agent-core";
import { v7 as uuidv7 } from "uuid";

const ProcessSchema = Type.Object({
  action: Type.String({ description: "Action: start | status | stop." }),
  id: Type.Optional(Type.String({ description: "Process id for status/stop." })),
  command: Type.Optional(Type.String({ description: "Command to run for start." })),
  cwd: Type.Optional(Type.String({ description: "Working directory." })),
});

type ProcessEntry = {
  id: string;
  command: string;
  cwd?: string;
  child: ChildProcess;
  exitCode: number | null;
  startedAt: number;
};

const PROCESS_REGISTRY = new Map<string, ProcessEntry>();

export type ProcessResult = {
  id?: string;
  running?: boolean;
  exitCode?: number | null;
  message?: string;
};

export function createProcessTool(defaultCwd?: string): AgentTool<typeof ProcessSchema, ProcessResult> {
  return {
    name: "process",
    label: "Process",
    description: "Manage background processes (start, status, stop).",
    parameters: ProcessSchema,
    execute: async (_toolCallId, params, signal) => {
      const action = String(params.action ?? "").toLowerCase();
      if (!action) {
        throw new Error("Missing action");
      }

      if (action === "start") {
        const command = String(params.command ?? "");
        if (!command) throw new Error("Missing command");
        const id = params.id ? String(params.id) : uuidv7();
        if (PROCESS_REGISTRY.has(id)) {
          throw new Error(`Process already exists: ${id}`);
        }
        const child = spawn(command, {
          shell: true,
          cwd: params.cwd || defaultCwd,
          stdio: ["ignore", "pipe", "pipe"],
          detached: true,
        });
        const entry: ProcessEntry = {
          id,
          command,
          cwd: params.cwd || defaultCwd,
          child,
          exitCode: null,
          startedAt: Date.now(),
        };
        PROCESS_REGISTRY.set(id, entry);
        child.on("close", (code) => {
          entry.exitCode = code;
        });
        if (signal) {
          signal.addEventListener("abort", () => {
            child.kill("SIGTERM");
          });
        }
        return {
          content: [{ type: "text", text: `Started process ${id}` }],
          details: { id, running: true },
        };
      }

      if (action === "status") {
        const id = String(params.id ?? "");
        const entry = PROCESS_REGISTRY.get(id);
        if (!entry) {
          return {
            content: [{ type: "text", text: `Process not found: ${id}` }],
            details: { id, running: false },
          };
        }
        const running = entry.exitCode === null;
        return {
          content: [{ type: "text", text: running ? `Process running: ${id}` : `Process exited: ${id}` }],
          details: { id, running, exitCode: entry.exitCode },
        };
      }

      if (action === "stop") {
        const id = String(params.id ?? "");
        const entry = PROCESS_REGISTRY.get(id);
        if (!entry) {
          return {
            content: [{ type: "text", text: `Process not found: ${id}` }],
            details: { id, running: false },
          };
        }
        entry.child.kill("SIGTERM");
        return {
          content: [{ type: "text", text: `Stopped process ${id}` }],
          details: { id, running: false },
        };
      }

      throw new Error(`Unknown action: ${action}`);
    },
  };
}
