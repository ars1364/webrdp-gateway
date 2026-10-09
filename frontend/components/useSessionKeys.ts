"use client";

import { useEffect, useRef, useState, type RefObject } from "react";
import type * as G from "guacamole-common-js";
import { enterFullscreen, exitFullscreen } from "@/lib/fullscreen";
import { K, sendCombo, typeText, type Modifier } from "@/lib/keyCombos";
import type { KeyboardHandle } from "@/lib/rdpKeyboard";

const MOD_KEYSYM: Record<Modifier, number> = { Ctrl: K.Ctrl, Alt: K.Alt, Shift: K.Shift, Win: K.Win };

// Keyboard features of one session: combos, sticky modifiers, typing text,
// releasing stuck keys, fullscreen with Keyboard Lock.
export function useSessionKeys(client: RefObject<G.Client | null>, kbd: RefObject<KeyboardHandle | null>) {
  const [sticky, setSticky] = useState<Set<Modifier>>(new Set());
  const stickyRef = useRef(sticky);
  const [fullscreen, setFullscreen] = useState<"off" | "on" | "locked">("off");

  useEffect(() => { stickyRef.current = sticky; }, [sticky]);
  useEffect(() => {
    const onChange = () => { if (!document.fullscreenElement) setFullscreen("off"); };
    document.addEventListener("fullscreenchange", onChange);
    return () => document.removeEventListener("fullscreenchange", onChange);
  }, []);

  // Called by the keyboard wiring on the next real key: apply + clear latches.
  const takeSticky = (): number[] => {
    const mods = [...stickyRef.current].map((m) => MOD_KEYSYM[m]);
    if (mods.length) {
      stickyRef.current = new Set();
      setSticky(new Set());
    }
    return mods;
  };

  const toggleSticky = (m: Modifier) =>
    setSticky((s) => {
      const n = new Set(s);
      if (n.has(m)) n.delete(m); else n.add(m);
      return n;
    });

  const combo = (keys: number[]) => { if (client.current) sendCombo(client.current, keys); };

  const type = async (text: string, signal: AbortSignal, onProgress: (n: number) => void) => {
    if (client.current) await typeText(client.current, text, signal, onProgress);
  };

  const releaseAll = () => {
    kbd.current?.releaseAll();
    setSticky(new Set());
  };

  const toggleFullscreen = async () => {
    if (document.fullscreenElement) {
      await exitFullscreen();
      setFullscreen("off");
      return;
    }
    const locked = await enterFullscreen();
    setFullscreen(document.fullscreenElement ? (locked ? "locked" : "on") : "off");
  };

  return { sticky, takeSticky, toggleSticky, combo, type, releaseAll, fullscreen, toggleFullscreen };
}
