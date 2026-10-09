"use client";

import { useEffect, useRef, useState, type RefObject } from "react";
import type * as G from "guacamole-common-js";
import { screenshot, ScreenRecorder } from "@/lib/capture";

// Screenshot + local recording state for one session.
export function useCapture(display: RefObject<G.Display | null>, label: string, notify: (msg: string) => void) {
  const rec = useRef<ScreenRecorder | null>(null);
  const [recordingSince, setRecordingSince] = useState<number | null>(null);
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    if (recordingSince === null) return;
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, [recordingSince]);

  // Never leave a recorder running after the session closes: save it.
  useEffect(() => () => void rec.current?.stop(), []);

  async function shoot() {
    if (!display.current) return;
    try {
      await screenshot(display.current, label);
      notify("Screenshot saved ✓");
    } catch {
      notify("Screenshot failed.");
    }
  }

  async function toggleRecord() {
    if (rec.current) {
      await rec.current.stop();
      rec.current = null;
      setRecordingSince(null);
      notify("Recording saved ✓");
      return;
    }
    if (!display.current) return;
    try {
      const r = new ScreenRecorder(display.current, label);
      r.start();
      rec.current = r;
      setRecordingSince(Date.now());
      setNow(Date.now());
    } catch (e) {
      notify(e instanceof Error ? e.message : "Recording failed.");
    }
  }

  const elapsed = recordingSince === null ? 0 : Math.max(0, Math.floor((now - recordingSince) / 1000));
  return { shoot, toggleRecord, recording: recordingSince !== null, elapsed };
}

export function formatElapsed(s: number): string {
  const m = Math.floor(s / 60), sec = s % 60, h = Math.floor(m / 60);
  const mm = String(m % 60).padStart(2, "0"), ss = String(sec).padStart(2, "0");
  return h ? `${h}:${mm}:${ss}` : `${mm}:${ss}`;
}
