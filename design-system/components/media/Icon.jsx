import React from "react";

export function Icon({ name, size = 16, strokeWidth = 1.75, style }) {
  const ref = React.useRef(null);
  React.useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.innerHTML = '<i data-lucide="' + name + '" width="' + size + '" height="' + size + '" stroke-width="' + strokeWidth + '"></i>';
    let tries = 0;
    (function go() {
      if (window.lucide && window.lucide.createIcons) { try { window.lucide.createIcons(); } catch (e) {} }
      else if (tries++ < 40) setTimeout(go, 150);
    })();
  }, [name, size, strokeWidth]);
  return <span ref={ref} style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: size, height: size, flex: "none", ...style }} />;
}
