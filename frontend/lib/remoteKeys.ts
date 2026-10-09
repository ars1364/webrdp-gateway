// Drive the remote desktop with synthetic key events (X11 keysyms).
import type * as G from "guacamole-common-js";

export const TRANSFER_PATH = "\\\\tsclient\\Transfer";

const SUPER_L = 0xffeb;
const RETURN = 0xff0d;

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

// Latin-1 printable characters map 1:1 to keysyms; others use the Unicode
// keysym range (0x01000000 + code point).
export function keysymFor(ch: string): number {
  const cp = ch.codePointAt(0) ?? 0;
  return cp >= 0x20 && cp <= 0xff ? cp : 0x01000000 + cp;
}

function tap(client: G.Client, k: number): void {
  client.sendKeyEvent(1, k);
  client.sendKeyEvent(0, k);
}

// Opens a folder in Windows Explorer via the Run dialog (Win+R).
export async function openInExplorer(client: G.Client, path: string = TRANSFER_PATH): Promise<void> {
  client.sendKeyEvent(1, SUPER_L);
  tap(client, keysymFor("r"));
  client.sendKeyEvent(0, SUPER_L);
  await sleep(600); // let the Run dialog open
  for (const ch of path) tap(client, keysymFor(ch));
  await sleep(50);
  tap(client, RETURN);
}
