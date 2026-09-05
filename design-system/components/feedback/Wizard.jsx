import React from "react";
import { Dialog } from "./Dialog.jsx";
import { Button } from "../forms/Button.jsx";
import { Icon } from "../media/Icon.jsx";

export function Wizard({ open, title, description, steps = [], onClose, onFinish, finishLabel = "Finish", width = 560 }) {
  const [i, setI] = React.useState(0);
  React.useEffect(() => { if (open) setI(0); }, [open]);
  const step = steps[i] || {};
  const last = i === steps.length - 1;
  const valid = step.valid !== false;
  return (
    <Dialog open={open} title={title} description={description} onClose={onClose} width={width}
      footer={
        <React.Fragment>
          <span style={{ marginRight: "auto", alignSelf: "center", font: "500 11px var(--font-mono)", letterSpacing: "0.06em", color: "var(--text-muted)" }}>{(i + 1) + " / " + steps.length}</span>
          {i > 0 && <Button variant="secondary" onClick={() => setI(i - 1)}>Back</Button>}
          <Button onClick={() => (last ? onFinish && onFinish() : setI(i + 1))} disabled={!valid}>{last ? finishLabel : "Next"}</Button>
        </React.Fragment>
      }>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 20 }}>
        {steps.map((s, si) => {
          const done = si < i, cur = si === i;
          return (
            <React.Fragment key={s.id || si}>
              {si > 0 && <span style={{ flex: 1, minWidth: 12, height: 1, background: done || cur ? "var(--accent)" : "var(--row-sep)", opacity: done || cur ? 0.4 : 1 }} />}
              <button type="button" onClick={done ? () => setI(si) : undefined} disabled={!done}
                style={{ display: "inline-flex", alignItems: "center", gap: 8, background: "none", border: "none", padding: 0, cursor: done ? "pointer" : "default", font: "inherit" }}>
                <span style={{ width: 22, height: 22, borderRadius: 999, flex: "none", display: "inline-flex", alignItems: "center", justifyContent: "center",
                  background: done ? "var(--accent-soft)" : "transparent",
                  border: done ? "1px solid transparent" : cur ? "1.5px solid var(--accent)" : "1px solid var(--border-strong)",
                  color: done || cur ? "var(--accent)" : "var(--text-muted)", font: "600 11px var(--font-mono)" }}>
                  {done ? <Icon name="check" size={12} /> : si + 1}
                </span>
                <span style={{ font: (cur ? "600" : "400") + " 12.5px var(--font-ui)", color: cur ? "var(--text-ink)" : "var(--text-secondary)", whiteSpace: "nowrap" }}>{s.title}</span>
              </button>
            </React.Fragment>
          );
        })}
      </div>
      {step.content}
    </Dialog>
  );
}
