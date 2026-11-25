// Match the boot splash to the theme the app last applied. The page CSP
// blocks inline scripts, so this runs from a file before first paint.
try {
  const dark = localStorage.getItem("aibertai.boot-dark");
  if (dark !== null) document.documentElement.dataset.bootTheme = dark === "1" ? "dark" : "light";
} catch {}
