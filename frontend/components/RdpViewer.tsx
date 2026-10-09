"use client";

import { useEffect, useRef, useState } from "react";
import type * as G from "guacamole-common-js";
import { ClipboardSync, type ClipStatus } from "@/lib/clipboardSync";
import { attachKeyboard } from "@/lib/rdpKeyboard";
import { ClipboardChip } from "./ClipboardChip";
import { ClipboardPanel } from "./ClipboardPanel";
import { Modal } from "./Modal";
import { Button } from "./ui";

export type ClipboardPolicy = { upload: boolean; download: boolean };

type Props = { ticket: string; label: string; clipboard: ClipboardPolicy; onClose: () => void };

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

export function RdpViewer({ ticket, label, clipboard, onClose }: Props) {
  const host = useRef<HTMLDivElement>(null);
  const pasteTarget = useRef<HTMLTextAreaElement>(null);
  const clientRef = useRef<G.Client | null>(null);
  const syncRef = useRef<ClipboardSync | null>(null);
  const [state, setState] = useState(1);
  const [error, setError] = useState("");
  const [clip, setClip] = useState<ClipStatus>("off");
  const [remoteText, setRemoteText] = useState("");
  const [toast, setToast] = useState("");
  const [panel, setPanel] = useState(false);
  const { upload, download } = clipboard;

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
      const display = client.getDisplay();
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

      const detachKeys = attachKeyboard(Guac, client, {
        interceptPaste: upload, pasteTarget: target,
        onGesture: () => void sync.gesture(), onPaste: (t) => sync.pasted(t),
      });

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
        detachKeys();
        client.disconnect();
      };
    })();
    return () => {
      disposed = true;
      clearTimeout(toastTimer);
      cleanup();
    };
  }, [ticket, upload, download]);

  function ctrlAltDel() {
    const c = clientRef.current;
    if (!c) return;
    const keys = [0xffe3, 0xffe9, 0xffff]; // Control_L, Alt_L, Delete
    keys.forEach((k) => c.sendKeyEvent(1, k));
    [...keys].reverse().forEach((k) => c.sendKeyEvent(0, k));
  }

  return (
    <Modal variant="fullscreen" label={`Remote desktop ${label}`} onClose={onClose} closeOnEscape={false}>
      <div className="flex h-12 shrink-0 items-center gap-3 border-b border-white/10 bg-stage-bar px-3 text-sm text-white">
        <span className="truncate font-medium" title={label}>{label}</span>
        <span className="text-white/60">{STATES[state] ?? ""}</span>
        <div className="ms-auto flex gap-2">
          <ClipboardChip status={clip} onAllow={() => void syncRef.current?.requestPermission()}
            onOpenPanel={() => setPanel((p) => !p)} />
          <Button variant="ghost" className="h-8" onClick={ctrlAltDel}>Ctrl+Alt+Del</Button>
          <Button variant="ghost" className="h-8" onClick={() => document.documentElement.requestFullscreen?.()}>
            Fullscreen
          </Button>
          <Button className="h-8" onClick={onClose}>Disconnect</Button>
        </div>
      </div>
      <div ref={host} className="relative min-h-0 flex-1 overflow-hidden" />
      {/* Off-screen paste target: receives the native paste event for Ctrl/Cmd+V. */}
      <textarea ref={pasteTarget} aria-hidden="true" tabIndex={-1} defaultValue=""
        className="pointer-events-none fixed -start-[9999px] top-0 size-px opacity-0" />
      {panel && clip !== "off" && (
        <ClipboardPanel remoteText={remoteText} canSend={upload}
          onSend={(text) => syncRef.current?.send(text)} onClose={() => setPanel(false)} />
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
