"use client";

import { useEffect, useRef, useState } from "react";
import type * as G from "guacamole-common-js";
import { ClipboardSync, type ClipStatus } from "@/lib/clipboardSync";
import { uploadFile, type Transfer } from "@/lib/fileTransfer";
import { receiveStream, RemoteFs } from "@/lib/remoteFs";
import { attachKeyboard, type KeyboardHandle } from "@/lib/rdpKeyboard";
import { openInExplorer } from "@/lib/remoteKeys";
import { ClipboardChip } from "./ClipboardChip";
import { ClipboardPanel } from "./ClipboardPanel";
import { Modal } from "./Modal";
import { SessionToolbar } from "./SessionToolbar";
import { TransferPanel } from "./TransferPanel";
import { useCapture } from "./useCapture";
import { useSessionKeys } from "./useSessionKeys";
import { KeysMenu } from "./KeysMenu";
import { TypeTextDialog } from "./TypeTextDialog";
import { Button } from "./ui";

export type ClipboardPolicy = { upload: boolean; download: boolean };
export type FilePolicy = { upload: boolean; download: boolean; maxMB: number };

type Props = {
  ticket: string;
  label: string;
  clipboard: ClipboardPolicy;
  files: FilePolicy;
  serverRecorded: boolean;
  onClose: () => void;
};

const STATES = ["Idle", "Connecting…", "Waiting for desktop…", "Connected", "Disconnecting…", "Disconnected"];

// guacd status codes worth translating for humans.
const ERRORS: Record<number, string> = {
  519: "The remote desktop closed the connection.",
  769: "Login rejected by the remote desktop (wrong username or password).",
  771: "Access denied by the remote desktop.",
  514: "The remote desktop did not respond in time.",
  512: "Upstream error while connecting.",
  515: "The remote desktop refused the connection.",
  516: "Could not resolve or reach the remote desktop.",
};

export function RdpViewer({ ticket, label, clipboard, files, serverRecorded, onClose }: Props) {
  const host = useRef<HTMLDivElement>(null);
  const pasteTarget = useRef<HTMLTextAreaElement>(null);
  const clientRef = useRef<G.Client | null>(null);
  const syncRef = useRef<ClipboardSync | null>(null);
  const [state, setState] = useState(1);
  const [error, setError] = useState("");
  const [clip, setClip] = useState<ClipStatus>("off");
  const [remoteText, setRemoteText] = useState("");
  const [toast, setToast] = useState("");
  const [panel, setPanel] = useState<"clipboard" | "files" | "keys" | null>(null);
  const [typing, setTyping] = useState(false);
  const kbdRef = useRef<KeyboardHandle | null>(null);
  const keys = useSessionKeys(clientRef, kbdRef);
  const takeStickyRef = useRef(keys.takeSticky);
  takeStickyRef.current = keys.takeSticky;
  const [transfers, setTransfers] = useState<Transfer[]>([]);
  const [dragging, setDragging] = useState(false);
  const guacRef = useRef<typeof G | null>(null);
  const displayRef = useRef<G.Display | null>(null);
  const flash = (msg: string) => { setToast(msg); setTimeout(() => setToast(""), 2000); };
  const capture = useCapture(displayRef, label, flash);
  const { upload, download } = clipboard;
  const fileUp = files.upload, fileDown = files.download;
  const [uploadedToast, setUploadedToast] = useState("");
  const [fs, setFs] = useState<RemoteFs | null>(null);
  const track = (t: Transfer) => {
    setTransfers((all) => [t, ...all.filter((x) => x.id !== t.id)].slice(0, 20));
    if (t.direction === "upload" && t.state === "done") setUploadedToast(t.name);
  };
  const openTransfer = () => {
    setUploadedToast("");
    if (clientRef.current) void openInExplorer(clientRef.current);
  };

  useEffect(() => {
    let disposed = false;
    let cleanup = () => {};
    let toastTimer: ReturnType<typeof setTimeout>;
    (async () => {
      const Guac = (await import("guacamole-common-js")).default as unknown as typeof G;
      const el = host.current, target = pasteTarget.current;
      if (disposed || !el || !target) return;

      const tunnel = new Guac.WebSocketTunnel("/api/v1/tunnel");
      const client = new Guac.Client(tunnel);
      clientRef.current = client;
      guacRef.current = Guac;
      if (fileDown) {
        // Files Windows drops into Transfer\Download arrive as "file" streams.
        client.onfile = (stream: G.InputStream, mimetype: string, name: string) => {
          setPanel("files");
          receiveStream(Guac, stream, mimetype, name, track);
        };
        client.onfilesystem = (object: G.Object) => setFs(new RemoteFs(Guac, object));
      }
      const display = client.getDisplay();
      displayRef.current = display;
      const view = display.getElement();
      el.appendChild(view);

      client.onstatechange = (s: number) => setState(s);
      client.onerror = (st: G.Status) => setError(ERRORS[st.code] ?? st.message ?? "Connection error.");
      tunnel.onerror = (st: G.Status) => setError(ERRORS[st.code] ?? st.message ?? "Tunnel error.");

      const fit = () => {
        const w = display.getWidth(), h = display.getHeight();
        if (w && h) display.scale(Math.min(el.clientWidth / w, el.clientHeight / h));
      };
      display.onresize = fit;

      const sync = new ClipboardSync(Guac, client, {
        upload, download, onStatus: setClip,
        onRemoteText: (text, copied) => {
          setRemoteText(text);
          setToast(copied ? "Copied from remote ✓" : "Copied in Windows: click or press a key to finish copying");
          clearTimeout(toastTimer);
          toastTimer = setTimeout(() => setToast(""), copied ? 1800 : 4000);
        },
      });
      syncRef.current = sync;
      void sync.start();

      const mouse = new Guac.Mouse(view);
      // applyDisplayScale=true: the client maps screen coords back to remote pixels.
      mouse.onEach(["mousedown", "mouseup", "mousemove"], (ev) =>
        client.sendMouseState((ev as unknown as G.Mouse.Event).state, true));
      // Native cursor: the remote pointer image becomes the real CSS cursor.
      display.oncursor = (canvas: HTMLCanvasElement, x: number, y: number) => {
        display.showCursor(!mouse.setCursor(canvas, x, y));
      };
      const onDown = () => {
        target.focus({ preventScroll: true });
        void sync.gesture();
      };
      view.addEventListener("mousedown", onDown);

      const kbd = attachKeyboard(Guac, client, {
        interceptPaste: upload, pasteTarget: target, takeSticky: () => takeStickyRef.current(),
        onGesture: () => void sync.gesture(), onPaste: (t) => sync.pasted(t),
      });

      kbdRef.current = kbd;

      let t: ReturnType<typeof setTimeout>;
      const onResize = () => {
        clearTimeout(t);
        t = setTimeout(() => client.sendSize(el.clientWidth, el.clientHeight), 300);
        fit();
      };
      window.addEventListener("resize", onResize);

      const dpi = Math.round(96 * Math.min(window.devicePixelRatio || 1, 2));
      client.connect(new URLSearchParams({
        ticket, width: String(el.clientWidth), height: String(el.clientHeight), dpi: String(dpi),
      }).toString());
      target.focus({ preventScroll: true });

      cleanup = () => {
        window.removeEventListener("resize", onResize);
        view.removeEventListener("mousedown", onDown);
        kbd.detach();
        kbdRef.current = null;
        client.disconnect();
      };
    })();
    return () => {
      disposed = true;
      clearTimeout(toastTimer);
      cleanup();
    };
  }, [ticket, upload, download, fileDown]);

  // Drag-and-drop uploads onto the session area.
  const uploadRef = useRef<(l: FileList) => void>(() => {});
  useEffect(() => {
    const el = host.current;
    if (!el || !fileUp) return;
    const over = (e: DragEvent) => { e.preventDefault(); setDragging(true); };
    const leave = () => setDragging(false);
    const drop = (e: DragEvent) => {
      e.preventDefault();
      setDragging(false);
      if (e.dataTransfer?.files.length) uploadRef.current(e.dataTransfer.files);
    };
    el.addEventListener("dragover", over);
    el.addEventListener("dragleave", leave);
    el.addEventListener("drop", drop);
    return () => {
      el.removeEventListener("dragover", over);
      el.removeEventListener("dragleave", leave);
      el.removeEventListener("drop", drop);
    };
  }, [fileUp]);

  function uploadAll(list: FileList) {
    const c = clientRef.current, Guac = guacRef.current;
    if (!c || !Guac || !fileUp) return;
    setPanel("files");
    Array.from(list).forEach((f) => uploadFile(Guac, c, f, track));
  }
  uploadRef.current = uploadAll;

  return (
    <Modal variant="fullscreen" label={`Remote desktop ${label}`} onClose={onClose} closeOnEscape={false}>
      <SessionToolbar label={label} stateText={STATES[state] ?? ""} recording={capture.recording}
        elapsed={capture.elapsed} serverRecorded={serverRecorded} onScreenshot={() => void capture.shoot()}
        onToggleRecord={() => void capture.toggleRecord()} onDisconnect={onClose}
        fullscreen={keys.fullscreen} onToggleFullscreen={() => void keys.toggleFullscreen()}
        keysOpen={panel === "keys"} stickyCount={keys.sticky.size}
        onToggleKeys={() => setPanel((p) => (p === "keys" ? null : "keys"))}>
        <ClipboardChip status={clip} onAllow={() => void syncRef.current?.requestPermission()}
          onOpenPanel={() => setPanel((p) => (p === "clipboard" ? null : "clipboard"))} />
        {(fileUp || fileDown) && (
          <Button variant="ghost" className="h-8" aria-pressed={panel === "files"}
            onClick={() => setPanel((p) => (p === "files" ? null : "files"))}>
            Files{transfers.some((t) => t.state === "running") ? " ⋯" : ""}
          </Button>
        )}
      </SessionToolbar>
      {/* Drop target for uploads; the keyboard-accessible path is the Files panel. */}
      <div ref={host} className="relative min-h-0 flex-1 overflow-hidden" />
      {dragging && (
        <div className="pointer-events-none absolute inset-x-6 bottom-6 top-16 flex items-center justify-center rounded-xl border-2 border-dashed border-primary bg-black/40 text-sm text-white">
          Drop to upload: Windows → This PC → “Transfer on WebRDP”
        </div>
      )}
      {/* Off-screen paste target: receives the native paste event for Ctrl/Cmd+V. */}
      <textarea ref={pasteTarget} aria-hidden="true" tabIndex={-1} defaultValue=""
        className="pointer-events-none fixed -start-[9999px] top-0 size-px opacity-0" />
      {panel === "keys" && (
        <KeysMenu sticky={keys.sticky} onToggleSticky={keys.toggleSticky} onCombo={keys.combo}
          onTypeText={() => setTyping(true)} onReleaseAll={keys.releaseAll} onClose={() => setPanel(null)} />
      )}
      {typing && <TypeTextDialog onType={keys.type} onClose={() => setTyping(false)} />}
      {keys.fullscreen === "locked" && (
        <div role="status" className="pointer-events-none absolute start-1/2 top-14 -translate-x-1/2 rounded-lg bg-surface/90 px-3 py-1.5 text-xs text-ink shadow-sm">
          Keyboard locked to Windows: Win, Alt+Tab and Esc go to the remote. Hold Esc to exit fullscreen.
        </div>
      )}
      {panel === "files" && (
        <TransferPanel transfers={transfers} canUpload={fileUp} canDownload={fileDown} maxMB={files.maxMB}
          fs={fs} onTrack={track}
          onPick={uploadAll} onOpenInWindows={openTransfer} onClose={() => setPanel(null)} />
      )}
      {panel === "clipboard" && clip !== "off" && (
        <ClipboardPanel remoteText={remoteText} canSend={upload}
          onSend={(text) => syncRef.current?.send(text)} onClose={() => setPanel(null)} />
      )}
      {uploadedToast && (
        <div role="status" className="absolute bottom-16 start-1/2 flex -translate-x-1/2 items-center gap-3 rounded-lg bg-surface px-3 py-2 text-xs text-ink shadow-sm">
          <span className="max-w-[40vw] truncate">Uploaded “{uploadedToast}” to \\tsclient\Transfer</span>
          <Button className="h-7" onClick={openTransfer}>Open in Windows</Button>
          <Button variant="ghost" className="h-7" onClick={() => setUploadedToast("")} aria-label="Dismiss">✕</Button>
        </div>
      )}
      {toast && (
        <div role="status" className="pointer-events-none absolute bottom-6 start-1/2 -translate-x-1/2 rounded-lg bg-surface px-3 py-2 text-xs text-ink shadow-sm">
          {toast}
        </div>
      )}
      {(error || state === 5) && (
        <div className="absolute inset-x-0 top-16 mx-auto w-fit max-w-[90vw] rounded-lg border border-red-200 bg-surface px-4 py-3 text-sm text-ink shadow">
          <p>{error || "Disconnected."}</p>
          <Button className="mt-2 h-8" onClick={onClose}>Back</Button>
        </div>
      )}
    </Modal>
  );
}
