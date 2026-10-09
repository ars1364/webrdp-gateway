import { describe, expect, it } from "vitest";
import { clampText, MAX_CLIPBOARD_BYTES } from "./clipboard";

describe("clampText", () => {
  it.each([
    ["short ascii", "hello", 5],
    ["exactly at limit", "a".repeat(MAX_CLIPBOARD_BYTES), MAX_CLIPBOARD_BYTES],
    ["over limit", "a".repeat(MAX_CLIPBOARD_BYTES + 10), MAX_CLIPBOARD_BYTES],
  ])("%s", (_n, input, wantBytes) => {
    expect(new TextEncoder().encode(clampText(input)).length).toBe(wantBytes);
  });

  it("never splits a multi-byte character into garbage", () => {
    const out = clampText("€".repeat(MAX_CLIPBOARD_BYTES)); // 3 bytes each
    expect(out.endsWith("�")).toBe(false);
    expect(new TextEncoder().encode(out).length).toBeLessThanOrEqual(MAX_CLIPBOARD_BYTES);
  });
});
