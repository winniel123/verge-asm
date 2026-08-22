import React from "react";
import { Icon } from "../media/Icon.jsx";

/* Drag-and-drop file input (CSV imports). Click browses; drop accepts. */
export function FileDrop({ label = "Drop a CSV here or click to browse", hint, accept, multiple, onFiles, compact, style }) {
  const [over, setOver] = React.useState(false);
  const [files, setFiles] = React.useState([]);
  const inputRef = React.useRef(null);
  const handle = (list) => {
    const arr = Array.prototype.slice.call(list || []);
    if (!arr.length) return;
    setFiles(arr);
    onFiles && onFiles(arr);
  };
  const kb = (n) => (n / 1024).toFixed(1) + " KB";
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8, ...style }}>
      <div role="button" tabIndex={0}
        onClick={() => inputRef.current && inputRef.current.click()}
        onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); inputRef.current && inputRef.current.click(); } }}
        onDragOver={(e) => { e.preventDefault(); setOver(true); }}
        onDragLeave={() => setOver(false)}
        onDrop={(e) => { e.preventDefault(); setOver(false); handle(e.dataTransfer.files); }}
        style={{ display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", gap: 6, padding: compact ? "16px 14px" : "32px 20px", border: "1.5px dashed " + (over ? "var(--accent)" : "var(--border-strong)"), borderRadius: 16, background: over ? "var(--accent-softer)" : "var(--surface)", cursor: "pointer", transition: "border-color var(--dur-fast) var(--ease-out), background var(--dur-fast) var(--ease-out)", fontFamily: "var(--font-ui)" }}>
        <span style={{ color: over ? "var(--accent)" : "var(--text-muted)", display: "inline-flex" }}><Icon name="upload" size={compact ? 16 : 20} strokeWidth={1.6} /></span>
        <span style={{ font: "500 13px var(--font-ui)", color: "var(--text-body)" }}>{label}</span>
        {hint && <span style={{ font: "400 11.5px var(--font-ui)", color: "var(--text-muted)" }}>{hint}</span>}
        <input ref={inputRef} type="file" accept={accept} multiple={multiple} onChange={(e) => handle(e.target.files)} style={{ display: "none" }} />
      </div>
      {files.map((f) => (
        <div key={f.name} style={{ display: "flex", alignItems: "center", gap: 8, padding: "7px 10px", background: "var(--surface-sunken)", borderRadius: 10 }}>
          <span style={{ display: "inline-flex", color: "var(--text-secondary)" }}><Icon name="file-text" size={14} /></span>
          <span style={{ font: "400 12px var(--font-mono)", color: "var(--text-body)", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{f.name}</span>
          <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>{kb(f.size)}</span>
          <button type="button" aria-label="Remove file" onClick={() => setFiles(files.filter((x) => x !== f))}
            style={{ marginLeft: "auto", display: "inline-flex", alignItems: "center", justifyContent: "center", width: 20, height: 20, border: "none", borderRadius: 999, background: "transparent", color: "var(--text-muted)", cursor: "pointer" }}>
            <svg viewBox="0 0 10 10" width="9" height="9"><path d="M2 2l6 6M8 2l-6 6" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"></path></svg>
          </button>
        </div>
      ))}
    </div>
  );
}
