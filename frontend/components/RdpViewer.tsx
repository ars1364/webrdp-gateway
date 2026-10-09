"use client";

import { useEffect, useRef, useState } from "react";
import type * as G from "guacamole-common-js";
import { Modal } from "./Modal";
import { Button } from "./ui";

type Props = { ticket: string; label: string; onClose: () => void };

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

export function RdpViewer({ ticket, label, onClose }: Props) {
  const host = useRef<HTMLDivElement>(null);
  const clientRef = useRef<G.Client | null>(null);
  const [state, setState] = useState(1);
  const [error, setError] = useState("");

  useEffect(() => {
    let disposed = false;
    let cleanup = () => {};
    (async () => {
      const Guac = (await import("guacamole-common-js")).default as unknown as typeof G;
      const el = host.current;
      if (disposed || !el) return;

      const tunnel = new Guac.WebSocketTunnel("/api/v1/tunnel");
      const client = new Guac.Client(tunnel);
      clientRef.current = client;
      const display = client.getDisplay();
      const view = display.getElement();
      view.style.cursor = "none";
      el.appendChild(view);

      client.onstatechange = (s: number) => setState(s);
      client.onerror = (st: G.Status) => setError(ERRORS[st.code] ?? st.message ?? "Connection error.");
      tunnel.onerror = (st: G.Status) => setError(ERRORS[st.code] ?? st.message ?? "Tunnel error.");

      const fit = () => {
        const w = display.getWidth(), h = display.getHeight();
        if (w && h) display.scale(Math.min(el.clientWidth / w, el.clientHeight / h));
      };
      display.onresize = fit;

      const mouse = new Guac.Mouse(view);
      // applyDisplayScale=true: the client maps screen coords back to remote pixels.
      mouse.onEach(["mousedown", "mouseup", "mousemove"], (ev) =>
        client.sendMouseState((ev as unknown as G.Mouse.Event).state, true));
      display.showCursor(true);
      mouse.onEach(["mouseout"], () => display.showCursor(false));

      const kb = new Guac.Keyboard(document);
      kb.onkeydown = (k: number) => { client.sendKeyEvent(1, k); return false; };
      kb.onkeyup = (k: number) => client.sendKeyEvent(0, k);

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

      cleanup = () => {
        window.removeEventListener("resize", onResize);
        kb.onkeydown = null;
        kb.onkeyup = null;
        kb.reset();
        client.disconnect();
      };
    })();
    return () => {
      disposed = true;
      cleanup();
    };
  }, [ticket]);

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
          <Button variant="ghost" className="h-8" onClick={ctrlAltDel}>Ctrl+Alt+Del</Button>
          <Button variant="ghost" className="h-8" onClick={() => document.documentElement.requestFullscreen?.()}>
            Fullscreen
          </Button>
          <Button className="h-8" onClick={onClose}>Disconnect</Button>
        </div>
      </div>
      <div ref={host} className="relative min-h-0 flex-1 overflow-hidden" />
      {(error || state === 5) && (
        <div className="absolute inset-x-0 top-16 mx-auto w-fit max-w-[90vw] rounded-lg border border-red-200 bg-surface px-4 py-3 text-sm text-ink shadow">
          <p>{error || "Disconnected."}</p>
          <Button className="mt-2 h-8" onClick={onClose}>Back</Button>
        </div>
      )}
    </Modal>
  );
}
