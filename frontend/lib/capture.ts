// Client-side capture of the remote screen: PNG screenshots and WebM
// recordings. Nothing leaves the browser except the file the user saves.
import type * as G from "guacamole-common-js";
import { saveBlob } from "./fileTransfer";

export function captureName(label: string, ext: string, now = new Date()): string {
  const safe = label.replace(/[^\w.-]+/g, "_").replace(/^_+|_+$/g, "").slice(0, 60) || "session";
  const ts = now.toISOString().replace(/[:]/g, "-").replace(/\.\d+Z$/, "Z");
  return `webrdp-${safe}-${ts}.${ext}`;
}

// Saves the full-resolution remote screen (all layers flattened) as PNG.
export function screenshot(display: G.Display, label: string): Promise<void> {
  return new Promise((resolve, reject) => {
    display.flatten().toBlob((blob) => {
      if (!blob) return reject(new Error("Screenshot failed"));
      saveBlob(blob, captureName(label, "png"));
      resolve();
    }, "image/png");
  });
}

const FPS = 10; // enough for desktop work, cheap on a laptop CPU

export function pickVideoMime(isSupported: (m: string) => boolean): string | null {
  for (const m of ["video/webm;codecs=vp9", "video/webm;codecs=vp8", "video/webm", "video/mp4"]) {
    if (isSupported(m)) return m;
  }
  return null;
}

// Records the remote screen to a video file via MediaRecorder on a canvas
// that is repainted from display.flatten() FPS times per second.
export class ScreenRecorder {
  private canvas = document.createElement("canvas");
  private timer: ReturnType<typeof setInterval> | undefined;
  private recorder: MediaRecorder | null = null;
  private chunks: Blob[] = [];
  readonly mime: string;

  constructor(private display: G.Display, private label: string) {
    const mime = typeof MediaRecorder === "undefined" ? null : pickVideoMime((m) => MediaRecorder.isTypeSupported(m));
    if (!mime) throw new Error("This browser cannot record video.");
    this.mime = mime;
  }

  start(): void {
    const ctx = this.canvas.getContext("2d");
    if (!ctx) throw new Error("Canvas unavailable");
    const paint = () => {
      const frame = this.display.flatten();
      if (this.canvas.width !== frame.width || this.canvas.height !== frame.height) {
        this.canvas.width = frame.width;
        this.canvas.height = frame.height;
      }
      ctx.drawImage(frame, 0, 0);
    };
    paint();
    this.timer = setInterval(paint, 1000 / FPS);
    this.recorder = new MediaRecorder(this.canvas.captureStream(FPS), { mimeType: this.mime, videoBitsPerSecond: 2_500_000 });
    this.recorder.ondataavailable = (e) => { if (e.data.size) this.chunks.push(e.data); };
    this.recorder.start(1000);
  }

  // Stops and saves the file; resolves once it is handed to the browser.
  stop(): Promise<void> {
    clearInterval(this.timer);
    const rec = this.recorder;
    if (!rec || rec.state === "inactive") return Promise.resolve();
    return new Promise((resolve) => {
      rec.onstop = () => {
        const ext = this.mime.startsWith("video/mp4") ? "mp4" : "webm";
        saveBlob(new Blob(this.chunks, { type: this.mime }), captureName(this.label, ext));
        this.chunks = [];
        resolve();
      };
      rec.stop();
    });
  }
}
