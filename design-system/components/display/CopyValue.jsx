import React from "react";
import { Icon } from "../media/Icon.jsx";

export function CopyValue({ value, display, size = 12.5, style }) {
  const [hov, setHov] = React.useState(false);
  const [copied, setCopied] = React.useState(false);
  const copy = () => {
    const done = (ok) => { setCopied(ok); setTimeout(() => setCopied(false), 1400); };
    const legacy = () => {
      try {
        const ta = document.createElement("textarea");
        ta.value = value; ta.setAttribute("readonly", ""); ta.style.position = "fixed"; ta.style.opacity = "0";
        document.body.appendChild(ta); ta.select();
        const ok = document.execCommand("copy");
        document.body.removeChild(ta); done(ok);
      } catch (e) { done(false); }
    };
    if (navigator.clipboard && window.isSecureContext) navigator.clipboard.writeText(value).then(() => done(true), legacy);
    else legacy();
  };
  return (
    <span onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
      style={{ display: "inline-flex", alignItems: "center", gap: 6, minWidth: 0, ...style }}>
      <span style={{ font: "400 " + size + "px var(--font-mono)", color: "var(--text-body)", overflowWrap: "anywhere" }}>{display || value}</span>
      <button type="button" aria-label="Copy value" onClick={copy}
        style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 18, height: 18, border: "none", borderRadius: 6, background: "transparent", color: copied ? "var(--ok)" : "var(--text-muted)", cursor: "pointer", padding: 0, opacity: hov || copied ? 1 : 0, transition: "opacity var(--dur-fast) var(--ease-out)", flex: "none" }}>
        <Icon name={copied ? "check" : "copy"} size={11} />
      </button>
    </span>
  );
}
