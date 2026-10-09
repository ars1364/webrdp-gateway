// Collects SHA-256 hashes of every inline <script> in the static export so
// the CSP can allow exactly those scripts instead of 'unsafe-inline'.
import { createHash } from "node:crypto";
import { readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const hashes = new Set();
const walk = (dir) => {
  for (const f of readdirSync(dir)) {
    const p = join(dir, f);
    if (statSync(p).isDirectory()) walk(p);
    else if (p.endsWith(".html")) {
      const html = readFileSync(p, "utf8");
      for (const m of html.matchAll(/<script(?![^>]*\bsrc=)[^>]*>([\s\S]*?)<\/script>/g)) {
        if (m[1].length) hashes.add(`'sha256-${createHash("sha256").update(m[1]).digest("base64")}'`);
      }
    }
  }
};
walk("out");
writeFileSync("csp-hashes.txt", [...hashes].join(" "));
console.log(`csp: ${hashes.size} inline script hashes`);
