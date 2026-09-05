import React from "react";
import { Dialog } from "./Dialog.jsx";
import { Button } from "../forms/Button.jsx";

export function ConfirmDialog({ open, title, message, detail, confirmLabel = "Confirm", tone = "danger", typedConfirm, onConfirm, onClose }) {
  const [typed, setTyped] = React.useState("");
  const [foc, setFoc] = React.useState(false);
  React.useEffect(() => { if (!open) setTyped(""); }, [open]);
  const ok = !typedConfirm || typed === typedConfirm;
  return (
    <Dialog open={open} title={title} description={message} onClose={onClose} width={440}
      footer={
        <React.Fragment>
          <Button variant="ghost" onClick={onClose}>Cancel</Button>
          <Button variant={tone === "danger" ? "danger" : "primary"} disabled={!ok} onClick={() => { if (onConfirm) onConfirm(); if (onClose) onClose(); }}>{confirmLabel}</Button>
        </React.Fragment>
      }>
      {(detail || typedConfirm) && (
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          {detail && <p style={{ margin: 0, font: "400 13px/1.55 var(--font-ui)", color: "var(--text-body)" }}>{detail}</p>}
          {typedConfirm && (
            <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
              <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-secondary)" }}>Type <span style={{ font: "600 12px var(--font-mono)", color: "var(--text-ink)" }}>{typedConfirm}</span> to confirm.</span>
              <input value={typed} spellCheck={false} onChange={(e) => setTyped(e.target.value)}
                onFocus={() => setFoc(true)} onBlur={() => setFoc(false)}
                style={{ height: 34, padding: "0 12px", background: "var(--surface)", color: "var(--text-ink)", font: "400 12.5px var(--font-mono)", border: "1px solid " + (foc ? "var(--danger-solid)" : "var(--border-default)"), borderRadius: 10, outline: "none", boxShadow: foc ? "0 0 0 3px color-mix(in srgb, var(--danger-solid) 16%, transparent)" : "none" }} />
            </div>
          )}
        </div>
      )}
    </Dialog>
  );
}
