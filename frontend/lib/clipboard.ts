// Clipboard bridge between the browser and a Guacamole client (text only).
import type * as G from "guacamole-common-js";

export const MAX_CLIPBOARD_BYTES = 256 * 1024; // guacd's default clipboard limit

export function clampText(text: string): string {
  const bytes = new TextEncoder().encode(text);
  if (bytes.length <= MAX_CLIPBOARD_BYTES) return text;
  return new TextDecoder().decode(bytes.slice(0, MAX_CLIPBOARD_BYTES)).replace(/�$/, "");
}

// Sends text to the remote clipboard.
export function sendToRemote(Guac: typeof G, client: G.Client, text: string): void {
  const stream = client.createClipboardStream("text/plain");
  const writer = new Guac.StringWriter(stream);
  writer.sendText(clampText(text));
  writer.sendEnd();
}

// Reads a remote clipboard stream. Non-text streams are refused (ack with
// an error) so guacd doesn't stall waiting for us.
export function readRemote(Guac: typeof G, stream: G.InputStream, mimetype: string, done: (text: string) => void): void {
  if (!/^text\//.test(mimetype)) {
    stream.sendAck("Unsupported clipboard type", 0x0100);
    return;
  }
  const reader = new Guac.StringReader(stream);
  let data = "";
  reader.ontext = (t: string) => {
    data += t;
  };
  reader.onend = () => done(data);
}

// Best-effort local write; the browser may refuse without user activation.
export async function writeLocal(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text);
    return true;
  } catch {
    return false;
  }
}
