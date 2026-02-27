/**
 * CodexSessionScanner - Scans Codex sessions and groups them by project (cwd).
 *
 * Unlike Claude which organizes sessions by project directory, Codex stores
 * sessions by date: ~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl
 *
 * Each session file has session_meta as the first line containing the cwd.
 * We scan all sessions and group them by cwd to create virtual "projects".
 *
 * Scan results are cached with a short TTL to avoid redundant filesystem work
 * when multiple callers need session data in the same request cycle.
 */

import { open } from "node:fs/promises";
import { readdir, stat } from "node:fs/promises";
import { homedir } from "node:os";
import { basename, join, sep } from "node:path";
import type { UrlProjectId } from "@yep-anywhere/shared";
import { getLogger } from "../logging/logger.js";
import type { Project } from "../supervisor/types.js";
import { encodeProjectId } from "./paths.js";

export const CODEX_SESSIONS_DIR =
  process.env.CODEX_SESSIONS_DIR ?? join(homedir(), ".codex", "sessions");
export const CODEX_DIR = CODEX_SESSIONS_DIR.replace(
  new RegExp(`\\${sep}sessions$`),
  "",
);

interface CodexSessionMeta {
  id: string;
  cwd: string;
  timestamp: string;
  cli_version?: string;
  model_provider?: string;
}

interface CodexSessionInfo {
  id: string;
  cwd: string;
  filePath: string;
  timestamp: string;
  mtime: number;
}

export interface CodexScannerOptions {
  sessionsDir?: string; // override for testing
}

/** How long to cache scan results (ms) */
const SCAN_CACHE_TTL = 5_000;

/** Strip UTF-8 BOM if present (common on Windows) */
function stripBom(str: string): string {
  if (str.charCodeAt(0) === 0xfeff) {
    return str.slice(1);
  }
  return str;
}

export class CodexSessionScanner {
  private sessionsDir: string;
  private cachedScan: { result: CodexSessionInfo[]; timestamp: number } | null =
    null;

  constructor(options: CodexScannerOptions = {}) {
    this.sessionsDir = options.sessionsDir ?? CODEX_SESSIONS_DIR;
  }

  /**
   * Scan all Codex sessions and group them by project (cwd).
   * Returns projects sorted by last activity (most recent first).
   */
  async listProjects(): Promise<Project[]> {
    const sessions = await this.scanAllSessions();

    // Group sessions by cwd
    const projectMap = new Map<
      string,
      { sessions: CodexSessionInfo[]; lastActivity: number }
    >();

    for (const session of sessions) {
      const existing = projectMap.get(session.cwd);
      if (existing) {
        existing.sessions.push(session);
        if (session.mtime > existing.lastActivity) {
          existing.lastActivity = session.mtime;
        }
      } else {
        projectMap.set(session.cwd, {
          sessions: [session],
          lastActivity: session.mtime,
        });
      }
    }

    // Convert to Project[]
    const projects: Project[] = [];
    for (const [cwd, data] of projectMap) {
      projects.push({
        id: encodeProjectId(cwd),
        path: cwd,
        name: basename(cwd),
        sessionCount: data.sessions.length,
        sessionDir: this.sessionsDir, // All sessions are in the same tree
        activeOwnedCount: 0,
        activeExternalCount: 0,
        lastActivity: new Date(data.lastActivity).toISOString(),
        provider: "codex",
      });
    }

    // Sort by last activity descending
    projects.sort((a, b) => {
      if (!a.lastActivity) return 1;
      if (!b.lastActivity) return -1;
      return (
        new Date(b.lastActivity).getTime() - new Date(a.lastActivity).getTime()
      );
    });

    return projects;
  }

  /**
   * Get sessions for a specific project (cwd).
   */
  async getSessionsForProject(
    projectPath: string,
  ): Promise<CodexSessionInfo[]> {
    const sessions = await this.scanAllSessions();
    return sessions
      .filter((s) => s.cwd === projectPath)
      .sort((a, b) => b.mtime - a.mtime);
  }

  /**
   * Scan all session files and extract metadata from the first line.
   * Results are cached for SCAN_CACHE_TTL to avoid redundant filesystem work.
   */
  private async scanAllSessions(): Promise<CodexSessionInfo[]> {
    if (
      this.cachedScan &&
      Date.now() - this.cachedScan.timestamp < SCAN_CACHE_TTL
    ) {
      return this.cachedScan.result;
    }

    const sessions: CodexSessionInfo[] = [];

    try {
      await stat(this.sessionsDir);
    } catch {
      // Sessions directory doesn't exist
      this.cachedScan = { result: [], timestamp: Date.now() };
      return [];
    }

    // Recursively find all .jsonl files
    const files = await this.findJsonlFiles(this.sessionsDir);

    getLogger().debug(
      `[CodexScanner] Found ${files.length} .jsonl files in ${this.sessionsDir}`,
    );

    // Read first line of each file in parallel (with concurrency limit)
    const BATCH_SIZE = 50;
    let failCount = 0;
    for (let i = 0; i < files.length; i += BATCH_SIZE) {
      const batch = files.slice(i, i + BATCH_SIZE);
      const results = await Promise.all(
        batch.map((f) => this.readSessionMeta(f)),
      );
      for (const result of results) {
        if (result) {
          sessions.push(result);
        } else {
          failCount++;
        }
      }
    }

    if (files.length > 0 && sessions.length === 0) {
      getLogger().warn(
        `[CodexScanner] Found ${files.length} .jsonl files but parsed 0 sessions (${failCount} failed). Session files may use an unsupported format. First file: ${files[0]}`,
      );
    } else if (failCount > 0) {
      getLogger().debug(
        `[CodexScanner] Parsed ${sessions.length} sessions, ${failCount} files skipped`,
      );
    }

    this.cachedScan = { result: sessions, timestamp: Date.now() };
    return sessions;
  }

  /**
   * Recursively find all .jsonl files in a directory.
   */
  private async findJsonlFiles(dir: string): Promise<string[]> {
    const files: string[] = [];

    try {
      const entries = await readdir(dir, { withFileTypes: true });

      for (const entry of entries) {
        const fullPath = join(dir, entry.name);
        if (entry.isDirectory()) {
          const subFiles = await this.findJsonlFiles(fullPath);
          files.push(...subFiles);
        } else if (entry.isFile() && entry.name.endsWith(".jsonl")) {
          files.push(fullPath);
        }
      }
    } catch (error) {
      getLogger().debug(
        `[CodexScanner] Error scanning directory ${dir}: ${error instanceof Error ? error.message : error}`,
      );
    }

    return files;
  }

  /**
   * Read just the first line of a session file to extract metadata.
   * Reads only the first 4KB to avoid loading large session files.
   */
  private async readSessionMeta(
    filePath: string,
  ): Promise<CodexSessionInfo | null> {
    let fd: Awaited<ReturnType<typeof open>> | null = null;
    try {
      const stats = await stat(filePath);

      // Read only the first 4KB — session_meta is always the first line
      fd = await open(filePath, "r");
      const buf = Buffer.alloc(4096);
      const { bytesRead } = await fd.read(buf, 0, 4096, 0);
      if (bytesRead === 0) {
        getLogger().debug(`[CodexScanner] Empty file: ${filePath}`);
        return null;
      }

      const content = stripBom(buf.toString("utf-8", 0, bytesRead));
      const firstNewline = content.indexOf("\n");
      const firstLine =
        firstNewline > 0 ? content.slice(0, firstNewline) : content;

      if (!firstLine.trim()) {
        getLogger().debug(`[CodexScanner] Empty first line: ${filePath}`);
        return null;
      }

      const parsed = JSON.parse(firstLine);

      // Validate it's a session_meta entry
      if (parsed.type !== "session_meta" || !parsed.payload) {
        getLogger().debug(
          `[CodexScanner] Unexpected first line type=${parsed.type} payload=${!!parsed.payload}: ${filePath}`,
        );
        return null;
      }

      const meta = parsed.payload as CodexSessionMeta;
      if (!meta.id || !meta.cwd) {
        getLogger().debug(
          `[CodexScanner] session_meta missing id or cwd: ${filePath}`,
        );
        return null;
      }

      return {
        id: meta.id,
        cwd: meta.cwd,
        filePath,
        timestamp: meta.timestamp,
        mtime: stats.mtimeMs,
      };
    } catch (error) {
      getLogger().debug(
        `[CodexScanner] Error reading ${filePath}: ${error instanceof Error ? error.message : error}`,
      );
      return null;
    } finally {
      await fd?.close();
    }
  }
}

// Singleton for convenience
export const codexSessionScanner = new CodexSessionScanner();
