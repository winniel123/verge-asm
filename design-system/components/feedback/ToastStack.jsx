import React from "react";
import { Toast } from "./Toast.jsx";

export function ToastStack({ toasts = [], onDismiss, ttl = 5000, style }) {
  const started = React.useRef({});
  const [leaving, setLeaving] = React.useState([]);
  const begin = (id) => {
    setLeaving((l) => (l.indexOf(id) !== -1 ? l : l.concat(id)));
    setTimeout(() => { setLeaving((l) => l.filter((x) => x !== id)); onDismiss && onDismiss(id); }, 260);
  };
  React.useEffect(() => {
    toasts.forEach((t) => {
      if (started.current[t.id]) return;
      started.current[t.id] = setTimeout(() => { delete started.current[t.id]; begin(t.id); }, t.ttl || ttl);
    });
    Object.keys(started.current).forEach((id) => {
      if (!toasts.some((t) => t.id === id)) { clearTimeout(started.current[id]); delete started.current[id]; }
    });
  }, [toasts]);
  React.useEffect(() => () => Object.values(started.current).forEach(clearTimeout), []);
  return (
    <div aria-live="polite" style={{ position: "fixed", right: 24, bottom: 24, zIndex: 110, display: "flex", flexDirection: "column", alignItems: "flex-end", gap: 10, pointerEvents: "none", ...style }}>
      {toasts.map((t) => {
        const isLeaving = leaving.indexOf(t.id) !== -1;
        return (
          <div key={t.id} style={{ display: "grid", gridTemplateRows: isLeaving ? "0fr" : "1fr", transition: "grid-template-rows var(--dur-base) var(--ease-out)" }}>
            <div style={{ minHeight: 0, overflow: isLeaving ? "hidden" : "visible" }}>
              <div style={{ pointerEvents: "auto", animation: isLeaving ? "vg-toast-out var(--dur-base) var(--ease-out) forwards" : "vg-toast-in var(--dur-slow) var(--ease-out)" }}>
                <Toast tone={t.tone} title={t.title} description={t.description} action={t.action} onDismiss={() => begin(t.id)} />
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
