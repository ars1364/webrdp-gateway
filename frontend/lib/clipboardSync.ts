// Clipboard sync controller for one RDP session. Patterns borrowed from
// Teleport (sync on every click/keypress), Kasm/Bastion (manual panel
// fallback) and xterm.js (native paste event: no permission prompt).
import type * as G from "guacamole-common-js";
import { readRemote, sendToRemote } from "./clipboard";

export type ClipStatus = "off" | "ready" | "synced" | "needs-permission" | "blocked";

type Opts = {
  upload: boolean;   // local → remote allowed
  download: boolean; // remote → local allowed
  onRemoteText: (text: string, copiedLocally: boolean) => void;
  onStatus: (s: ClipStatus) => void;
};

const isFirefox = () => typeof navigator !== "undefined" && /firefox/i.test(navigator.userAgent);

export class ClipboardSync {
  private lastSeen = "";       // last text exchanged in either direction
  private pendingWrite: string | null = null;
  private lastPoll = 0;
  private status: ClipStatus = "off";

  constructor(private Guac: typeof G, private client: G.Client, private o: Opts) {}

  async start(): Promise<void> {
    if (!this.o.upload && !this.o.download) return this.set("off");
    if (this.o.download) {
      this.client.onclipboard = (stream: G.InputStream, mime: string) =>
        readRemote(this.Guac, stream, mime, (text) => void this.fromRemote(text));
    }
    this.set((await this.readPermission()) === "granted" ? "synced" : this.o.upload && !isFirefox() ? "needs-permission" : "ready");
  }

  // Native paste event (Ctrl/Cmd+V into the hidden target): works in every
  // browser without a permission prompt.
  pasted(text: string): void {
    if (this.o.upload && text) this.toRemote(text);
  }

  // Call on every click/keydown inside the session (user activation).
  async gesture(): Promise<void> {
    if (this.pendingWrite !== null) {
      const t = this.pendingWrite;
      if (await this.writeLocal(t)) {
        this.pendingWrite = null;
        this.o.onRemoteText(t, true);
      }
    }
    // Skip while a remote copy is still waiting to land locally: the local
    // clipboard is stale and would overwrite what was just copied in Windows.
    if (this.pendingWrite !== null) return;
    if (this.o.upload && this.status === "synced" && Date.now() - this.lastPoll > 500) {
      this.lastPoll = Date.now();
      const text = await this.readLocal();
      if (text && text !== this.lastSeen) this.toRemote(text); // never push an empty clipboard
    }
  }

  // "Allow clipboard" chip: must run inside a click handler.
  async requestPermission(): Promise<void> {
    const text = await this.readLocal();
    if (text === null) return this.set("blocked");
    this.set("synced");
    if (text && text !== this.lastSeen) this.toRemote(text);
  }

  send(text: string): void {
    if (this.o.upload) this.toRemote(text);
  }

  private toRemote(text: string): void {
    this.lastSeen = text;
    sendToRemote(this.Guac, this.client, text);
  }

  private async fromRemote(text: string): Promise<void> {
    this.lastSeen = text;
    const ok = await this.writeLocal(text);
    this.pendingWrite = ok ? null : text; // flushed on the next gesture
    this.o.onRemoteText(text, ok);
  }

  private async writeLocal(text: string): Promise<boolean> {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      return false;
    }
  }

  private async readLocal(): Promise<string | null> {
    if (isFirefox() || !navigator.clipboard?.readText) return null; // FF pops a "Paste" menu per read
    try {
      return await navigator.clipboard.readText();
    } catch {
      return null;
    }
  }

  private async readPermission(): Promise<PermissionState | "unsupported"> {
    try {
      const p = await navigator.permissions.query({ name: "clipboard-read" as PermissionName });
      return p.state;
    } catch {
      return "unsupported";
    }
  }

  private set(s: ClipStatus): void {
    this.status = s;
    this.o.onStatus(s);
  }
}

// Keysyms used by the paste interception and the macOS Cmd → Ctrl mapping.
export const KEYSYM = {
  ControlL: 0xffe3,
  metaKeys: new Set([0xffe7, 0xffe8, 0xffeb, 0xffec]), // Meta_L/R, Super_L/R
  v: 0x76,
  V: 0x56,
};

export const isMac = () => typeof navigator !== "undefined" && /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);
