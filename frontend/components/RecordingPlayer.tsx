"use client";

import { useEffect, useRef, useState } from "react";
import type * as G from "guacamole-common-js";
import { api, type Recording } from "@/lib/api";
import { formatElapsed } from "./useCapture";
import { Modal } from "./Modal";
import { Button } from "./ui";

type Props = { rec: Recording; onClose: () => void };

// In-browser playback of a guacd session recording (Guacamole.SessionRecording).
export function RecordingPlayer({ rec, onClose }: Props) {
  const host = useRef<HTMLDivElement>(null);
  const player = useRef<G.SessionRecording | null>(null);
  const [state, setState] = useState<"loading" | "ready" | "error">("loading");
  const [playing, setPlaying] = useState(false);
  const [pos, setPos] = useState(0);
  const [dur, setDur] = useState(0);

  useEffect(() => {
    let disposed = false;
    let detach = () => {};
    (async () => {
      try {
        const [Guac, blob] = await Promise.all([
          import("guacamole-common-js").then((m) => m.default as unknown as typeof G),
          api.recordingBlob(rec.id),
        ]);
        const el = host.current;
        if (disposed || !el) return;
        const r = new Guac.SessionRecording(blob);
        player.current = r;
        const display = r.getDisplay();
        el.appendChild(display.getElement());
        const fit = () => {
          const w = display.getWidth(), h = display.getHeight();
          if (w && h) display.scale(Math.min(el.clientWidth / w, el.clientHeight / h));
        };
        display.onresize = fit;
        r.onprogress = (d: number) => setDur(d);
        r.onload = () => { setDur(r.getDuration()); setState("ready"); r.play(); };
        r.onerror = () => setState("error");
        r.onplay = () => setPlaying(true);
        r.onpause = () => setPlaying(false);
        r.onseek = (p: number) => setPos(p);
        window.addEventListener("resize", fit);
        detach = () => window.removeEventListener("resize", fit);
      } catch {
        if (!disposed) setState("error");
      }
    })();
    return () => {
      disposed = true;
      detach();
      player.current?.pause();
      player.current?.abort();
    };
  }, [rec.id]);

  const seconds = (ms: number) => Math.floor(ms / 1000);
  return (
    <Modal variant="fullscreen" label={`Recording ${rec.label}`} onClose={onClose}>
      <div className="flex h-12 shrink-0 items-center gap-3 border-b border-white/10 bg-stage-bar px-3 text-sm text-white">
        <span className="truncate font-medium" title={rec.label}>{rec.label || rec.target}</span>
        <span className="text-white/60">{new Date(rec.started_at).toLocaleString()}</span>
        <Button className="ms-auto h-8" onClick={onClose}>Close</Button>
      </div>
      <div ref={host} className="relative min-h-0 flex-1 overflow-hidden">
        {state === "loading" && <p className="p-6 text-sm text-white/70">Loading recording…</p>}
        {state === "error" && <p className="p-6 text-sm text-red-200">This recording could not be played.</p>}
      </div>
      <div className="flex h-14 shrink-0 items-center gap-3 border-t border-white/10 bg-stage-bar px-3 text-xs text-white">
        <Button variant="ghost" className="h-8 w-20" disabled={state !== "ready"}
          onClick={() => (playing ? player.current?.pause() : player.current?.play())}>
          {playing ? "Pause" : "Play"}
        </Button>
        <input type="range" min={0} max={Math.max(dur, 1)} value={Math.min(pos, dur)} disabled={state !== "ready"}
          aria-label="Seek" className="min-w-0 flex-1 accent-primary"
          onChange={(e) => player.current?.seek(Number(e.target.value))} />
        <span className="tabular-nums">{formatElapsed(seconds(pos))} / {formatElapsed(seconds(dur))}</span>
      </div>
    </Modal>
  );
}
