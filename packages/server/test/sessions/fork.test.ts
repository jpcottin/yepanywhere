import { randomUUID } from "node:crypto";
import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cloneCodexSession } from "../../src/sessions/fork.js";

describe("cloneCodexSession", () => {
  let testDir: string;
  let sessionDir: string;
  let sourceFilePath: string;

  beforeEach(async () => {
    testDir = join(tmpdir(), `codex-fork-test-${randomUUID()}`);
    sessionDir = join(testDir, "2026", "03", "08");
    await mkdir(sessionDir, { recursive: true });

    sourceFilePath = join(sessionDir, "rollout-source-session.jsonl");
    await writeFile(
      sourceFilePath,
      `${[
        JSON.stringify({
          type: "session_meta",
          timestamp: "2026-03-08T12:00:00.000Z",
          payload: {
            id: "source-session",
            cwd: "/tmp/demo-project",
            timestamp: "2026-03-08T12:00:00.000Z",
          },
        }),
        JSON.stringify({
          type: "event_msg",
          timestamp: "2026-03-08T12:00:01.000Z",
          payload: {
            type: "user_message",
            message: "Hello from Codex",
          },
        }),
      ].join("\n")}\n`,
      "utf-8",
    );
  });

  afterEach(async () => {
    await rm(testDir, { recursive: true, force: true });
  });

  it("writes clones with rollout-* filenames and updates session_meta.id", async () => {
    const result = await cloneCodexSession(sourceFilePath, "cloned-session");

    expect(result).toEqual({
      newSessionId: "cloned-session",
      entries: 2,
    });

    const clonedFilePath = join(sessionDir, "rollout-cloned-session.jsonl");
    const clonedContent = await readFile(clonedFilePath, "utf-8");
    const [metaLine, messageLine] = clonedContent.trim().split("\n");
    const meta = JSON.parse(metaLine) as {
      type: string;
      payload: { id: string };
    };
    const message = JSON.parse(messageLine) as {
      payload: { type: string; message: string };
    };

    expect(meta.type).toBe("session_meta");
    expect(meta.payload.id).toBe("cloned-session");
    expect(message.payload).toEqual({
      type: "user_message",
      message: "Hello from Codex",
    });
  });
});
