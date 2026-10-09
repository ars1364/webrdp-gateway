"use client";

import type { ReactNode } from "react";
import { formatElapsed } from "./useCapture";
import { Button } from "./ui";

type Props = {
  label: string;
  stateText: string;
  children?: ReactNode; // clipboard chip, Files button
  recording: boolean;
  elapsed: number;
  serverRecorded: boolean;
  onScreenshot: () => void;
  onToggleRecord: () => void;
  onCtrlAltDel: () => void;
  onDisconnect: () => void;
};

export function SessionToolbar(p: Props) {
  return (
    <div className="flex h-12 shrink-0 items-center gap-3 border-b border-white/10 bg-stage-bar px-3 text-sm text-white">
      <span className="truncate font-medium" title={p.label}>{p.label}</span>
      <span className="text-white/60">{p.stateText}</span>
      {p.serverRecorded && (
        <span className="rounded-md border border-red-400/60 px-2 py-0.5 text-xs text-red-200" title="Your administrator records this session for audit.">
          ● Session recorded
        </span>
      )}
      <div className="ms-auto flex flex-wrap justify-end gap-2">
        {p.children}
        <Button variant="ghost" className="h-8" onClick={p.onScreenshot} title="Save a PNG of the remote screen">
          Screenshot
        </Button>
        <Button variant={p.recording ? "danger" : "ghost"} className="h-8" onClick={p.onToggleRecord}
          aria-pressed={p.recording} title={p.recording ? "Stop and save the video" : "Record the screen to a video file on this PC"}>
          {p.recording ? `■ Stop ${formatElapsed(p.elapsed)}` : "● Record"}
        </Button>
        <Button variant="ghost" className="h-8" onClick={p.onCtrlAltDel}>Ctrl+Alt+Del</Button>
        <Button variant="ghost" className="h-8" onClick={() => document.documentElement.requestFullscreen?.()}>
          Fullscreen
        </Button>
        <Button className="h-8" onClick={p.onDisconnect}>Disconnect</Button>
      </div>
    </div>
  );
}
