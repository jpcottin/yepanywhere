import { randomUUID } from "node:crypto";
import { mkdir, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import type { UrlProjectId } from "@yep-anywhere/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  type SessionsDeps,
  createSessionsRoutes,
} from "../../src/routes/sessions.js";
import { CodexSessionReader } from "../../src/sessions/codex-reader.js";
import type { Project } from "../../src/supervisor/types.js";

describe("Codex clone route", () => {
  let testDir: string;
  let projectId: UrlProjectId;
  let projectPath: string;
  let project: Project;
  let reader: CodexSessionReader;

  beforeEach(async () => {
    testDir = join(tmpdir(), `codex-clone-route-${randomUUID()}`);
    const sessionDir = join(testDir, "2026", "03", "08");
    await mkdir(sessionDir, { recursive: true });

    projectPath = "/tmp/demo-project";
    projectId = "tmp-demo-project" as UrlProjectId;
    project = {
      id: projectId,
      path: projectPath,
      name: "demo-project",
      sessionCount: 1,
      sessionDir: testDir,
      activeOwnedCount: 0,
      activeExternalCount: 0,
      lastActivity: null,
      provider: "codex",
    };

    await writeFile(
      join(sessionDir, "rollout-source-session.jsonl"),
      `${[
        JSON.stringify({
          type: "session_meta",
          timestamp: "2026-03-08T12:00:00.000Z",
          payload: {
            id: "source-session",
            cwd: projectPath,
            timestamp: "2026-03-08T12:00:00.000Z",
          },
        }),
        JSON.stringify({
          type: "event_msg",
          timestamp: "2026-03-08T12:00:01.000Z",
          payload: {
            type: "user_message",
            message: "Prime the cache",
          },
        }),
      ].join("\n")}\n`,
      "utf-8",
    );

    reader = new CodexSessionReader({
      sessionsDir: testDir,
      projectPath,
    });
  });

  afterEach(async () => {
    await rm(testDir, { recursive: true, force: true });
  });

  it("invalidates cached Codex readers so the cloned session opens immediately", async () => {
    const sourceSummary = await reader.getSessionSummary(
      "source-session",
      projectId,
    );
    expect(sourceSummary).not.toBeNull();

    const codexScanner = {
      invalidateCache: vi.fn(),
    };

    const routes = createSessionsRoutes({
      supervisor: {} as SessionsDeps["supervisor"],
      scanner: {
        getOrCreateProject: vi.fn(async () => project),
      } as SessionsDeps["scanner"],
      readerFactory: vi.fn(() => reader),
      codexReaderFactory: vi.fn(() => reader),
      codexScanner: codexScanner as SessionsDeps["codexScanner"],
    });

    const response = await routes.request(
      `/projects/${projectId}/sessions/source-session/clone`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: "{}",
      },
    );

    expect(response.status).toBe(200);
    const body = (await response.json()) as { sessionId: string };
    expect(body.sessionId).toBeTruthy();

    const clonedSummary = await reader.getSessionSummary(
      body.sessionId,
      projectId,
    );
    expect(clonedSummary).not.toBeNull();
    expect(clonedSummary?.id).toBe(body.sessionId);
    expect(codexScanner.invalidateCache).toHaveBeenCalledTimes(1);
  });
});
