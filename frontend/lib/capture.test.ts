import { describe, expect, it } from "vitest";
import { captureName, pickVideoMime } from "./capture";

describe("captureName", () => {
  it.each([
    ["usa-rdp", "png", "webrdp-usa-rdp-2026-10-09T18-05-00Z.png"],
    ["146.70.41.142:6579", "webm", "webrdp-146.70.41.142_6579-2026-10-09T18-05-00Z.webm"],
    ["../../etc", "png", "webrdp-.._.._etc-2026-10-09T18-05-00Z.png"],
    ["", "png", "webrdp-session-2026-10-09T18-05-00Z.png"],
  ])("%s", (label, ext, want) => {
    expect(captureName(label, ext, new Date("2026-10-09T18:05:00.123Z"))).toBe(want);
  });
});

describe("pickVideoMime", () => {
  it.each<[string, string[], string | null]>([
    ["chrome", ["video/webm;codecs=vp9", "video/webm"], "video/webm;codecs=vp9"],
    ["safari", ["video/mp4"], "video/mp4"],
    ["none", [] as string[], null],
  ])("%s", (_n, supported, want) => {
    expect(pickVideoMime((m) => supported.includes(m))).toBe(want);
  });
});
