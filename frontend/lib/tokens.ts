// Single source of design tokens. globals.css mirrors these as CSS custom
// properties (light + dark); components use the Tailwind names, never hex.
export const color = {
  light: {
    primary: "#8a5c5c",
    primaryHover: "#a87272",
    ink: "#3d2a2e",
    ink2: "#6b5a5e",
    muted: "#9a8a82",
    line: "#e0d8d8",
    canvas: "#f7f3f2",
    surface: "#ffffff",
    stage: "#1a1a1a",
    stageBar: "#242424",
  },
  dark: {
    primary: "#c08f8f",
    primaryHover: "#d4a8a8",
    ink: "#f2e9e7",
    ink2: "#cbbcb8",
    muted: "#9a8a82",
    line: "#3a3a3a",
    canvas: "#1a1a1a",
    surface: "#242424",
    stage: "#111111",
    stageBar: "#1a1a1a",
  },
} as const;

export const type = {
  family: 'system-ui, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
  scale: { xs: "0.75rem", sm: "0.875rem", base: "1rem", lg: "1.125rem", xl: "1.25rem" },
} as const;
