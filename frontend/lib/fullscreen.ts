// Fullscreen + Keyboard Lock (Chromium): while locked, Win, Alt+Tab, Esc,
// Ctrl+W/N/T reach the remote desktop instead of the local OS/browser.
// Holding Esc exits fullscreen (browser-enforced).

type KeyboardLock = { lock?: (codes?: string[]) => Promise<void>; unlock?: () => void };

const kbd = (): KeyboardLock | undefined =>
  (typeof navigator !== "undefined" ? (navigator as Navigator & { keyboard?: KeyboardLock }).keyboard : undefined);

export const keyboardLockSupported = () => typeof kbd()?.lock === "function";

// Returns true when the keyboard ended up locked.
export async function enterFullscreen(): Promise<boolean> {
  try {
    await document.documentElement.requestFullscreen();
  } catch {
    return false;
  }
  try {
    await kbd()?.lock?.();
    return keyboardLockSupported();
  } catch {
    return false;
  }
}

export async function exitFullscreen(): Promise<void> {
  kbd()?.unlock?.();
  if (document.fullscreenElement) await document.exitFullscreen().catch(() => {});
}
