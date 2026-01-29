import { spawn } from "child_process";
import { Type } from "@sinclair/typebox";
import type { AgentTool } from "@mariozechner/pi-agent-core";

const ExecSchema = Type.Object({
  command: Type.String({ description: "Shell command to execute." }),
  cwd: Type.Optional(Type.String({ description: "Working directory." })),
  timeoutMs: Type.Optional(
    Type.Number({ description: "Timeout in milliseconds.", minimum: 0 }),
  ),
});

type ExecArgs = {
  command: string;
  cwd?: string;
  timeoutMs?: number;
};

export type ExecResult = {
  output: string;
  exitCode: number | null;
  truncated: boolean;
};

const MAX_OUTPUT_BYTES = 64 * 1024;

export function createExecTool(defaultCwd?: string): AgentTool<typeof ExecSchema, ExecResult> {
  return {
    name: "exec",
    label: "Exec",
    description: "Execute a shell command and return its output.",
    parameters: ExecSchema,
    execute: async (_toolCallId, args, signal) => {
      const { command, cwd, timeoutMs } = args as ExecArgs;
      return new Promise((resolve, reject) => {
        const child = spawn(command, {
          shell: true,
          cwd: cwd || defaultCwd,
          stdio: ["ignore", "pipe", "pipe"],
        });

        let timedOut = false;
        let timeout: NodeJS.Timeout | undefined;
        if (timeoutMs && timeoutMs > 0) {
          timeout = setTimeout(() => {
            timedOut = true;
            child.kill("SIGTERM");
          }, timeoutMs);
        }

        const chunks: Buffer[] = [];
        let size = 0;
        let truncated = false;

        const handleData = (data: Buffer) => {
          if (truncated) return;
          size += data.length;
          if (size > MAX_OUTPUT_BYTES) {
            truncated = true;
            const remaining = MAX_OUTPUT_BYTES - (size - data.length);
            if (remaining > 0) chunks.push(data.subarray(0, remaining));
            return;
          }
          chunks.push(data);
        };

        child.stdout?.on("data", handleData);
        child.stderr?.on("data", handleData);
        child.on("error", (err) => {
          if (timeout) clearTimeout(timeout);
          reject(err);
        });
        child.on("close", (code) => {
          if (timeout) clearTimeout(timeout);
          const output = Buffer.concat(chunks).toString("utf8");
          resolve({
            content: [{ type: "text", text: output || (timedOut ? "Process timed out." : "") }],
            details: {
              output,
              exitCode: code,
              truncated,
            },
          });
        });

        if (signal) {
          signal.addEventListener("abort", () => {
            child.kill("SIGTERM");
          });
        }
      });
    },
  };
}
