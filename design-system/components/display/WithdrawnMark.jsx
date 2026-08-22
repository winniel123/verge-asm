import React from "react";

/* Marks a signal/subject whose key is in no current population — derived on read; no status, no age.
   Dashed drift-loss chip: present in the record, absent from the world. */
export function WithdrawnMark({ size = "md", style }) {
  const sm = size === "sm";
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: sm ? 4 : 5, height: sm ? 18 : 20, padding: sm ? "0 6px" : "0 8px", borderRadius: 8, background: "transparent", border: "1px dashed var(--drift-loss-border)", color: "var(--drift-loss-fg)", fontFamily: "var(--font-mono)", fontSize: sm ? 10 : 10.5, fontWeight: 600, letterSpacing: "0.04em", whiteSpace: "nowrap", ...style }}>
      <svg viewBox="0 0 10 10" width={sm ? 9 : 10} height={sm ? 9 : 10} style={{ flex: "none" }}><path d="M1.6 5h6.8" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round"></path></svg>
      withdrawn
    </span>
  );
}
