import React from "react";

/* Code on inverted ink. Copy control appears on hover; title renders a micro-label bar. */
export function CodeBlock({ children, title, copyText, style }) {
  const [hov, setHov] = React.useState(false);
  const [copied, setCopied] = React.useState(false);
  const preRef = React.useRef(null);
  const copy = () => {
    const text = copyText != null ? copyText : preRef.current ? preRef.current.textContent : "";
    const done = (ok) => { setCopied(ok ? "ok" : "fail"); setTimeout(() => setCopied(false), 1600); };
    const legacy = () => {
      try {
        const ta = document.createElement("textarea");
        ta.value = text;
        ta.setAttribute("readonly", "");
        ta.style.position = "fixed";
        ta.style.opacity = "0";
        document.body.appendChild(ta);
        ta.select();
        const ok = document.execCommand("copy");
        document.body.removeChild(ta);
        done(ok);
      } catch (e) { done(false); }
    };
    if (navigator.clipboard && window.isSecureContext) navigator.clipboard.writeText(text).then(() => done(true), legacy);
    else legacy();
  };
  return (
    <div onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
      style={{ position: "relative", background: "var(--surface-inverted)", borderRadius: 12, overflow: "hidden", ...style }}>
      {title && <div style={{ padding: "8px 14px 0", font: "500 10.5px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--neutral-500)" }}>{title}</div>}
      <pre ref={preRef} style={{ margin: 0, padding: "13px 16px", overflow: "auto" }}>
        <code style={{ font: "400 12.5px/1.6 var(--font-mono)", color: "var(--text-on-inverted)" }}>{children}</code>
      </pre>
      {children != null && (
        <button type="button" aria-label="Copy" onClick={copy}
          style={{ position: "absolute", top: title ? 6 : 8, right: 8, display: "inline-flex", alignItems: "center", gap: 5, height: 24, padding: "0 8px", border: "1px solid rgba(255,255,255,0.18)", borderRadius: 8, background: "rgba(255,255,255,0.08)", color: copied === "ok" ? "var(--primary-300)" : "var(--neutral-400)", font: "500 10.5px var(--font-mono)", letterSpacing: "0.05em", cursor: "pointer", opacity: hov || copied ? 1 : 0, transition: "opacity var(--dur-fast) var(--ease-out)" }}>
          {copied === "ok" ? "copied" : copied === "fail" ? "blocked" : "copy"}
        </button>
      )}
    </div>
  );
}
