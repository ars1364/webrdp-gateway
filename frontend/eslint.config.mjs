import tseslint from "typescript-eslint";
import jsxA11y from "eslint-plugin-jsx-a11y";
import reactHooks from "eslint-plugin-react-hooks";

export default tseslint.config(
  { ignores: [".next/**", "out/**", "node_modules/**", "next-env.d.ts", "scripts/**"] },
  ...tseslint.configs.recommended,
  jsxA11y.flatConfigs.strict,
  {
    plugins: { "react-hooks": reactHooks },
    rules: {
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "error",
      // UI discipline: no raw hex colours in components, use tokens.
      "no-restricted-syntax": [
        "error",
        {
          selector: "Literal[value=/#[0-9a-fA-F]{3,8}\\b/]",
          message: "Use design tokens (lib/tokens.ts / Tailwind theme), not inline hex.",
        },
        {
          selector: "MemberExpression[object.name='window'][property.name='location']",
          message: "Navigate with the router, not window.location.",
        },
        {
          selector: "JSXAttribute[name.name='dangerouslySetInnerHTML']",
          message: "dangerouslySetInnerHTML is banned (XSS).",
        },
      ],
    },
  },
  { files: ["lib/tokens.ts"], rules: { "no-restricted-syntax": "off" } },
);
