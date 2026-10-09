import type { NextConfig } from "next";

// Static export: nginx serves the files; no Node process runs in production.
const nextConfig: NextConfig = {
  output: "export",
  trailingSlash: true,
  images: { unoptimized: true },
  poweredByHeader: false,
  reactStrictMode: true,
  devIndicators: false,
};

export default nextConfig;
