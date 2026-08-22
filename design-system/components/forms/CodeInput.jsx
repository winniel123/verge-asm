import React from "react";

/* TOTP / MFA code field: segmented digit boxes, auto-advance, paste-distributes, digits only.
   The verify step — enrollment secrets stay in SecretInput (write-only). Codes are never masked. */
export function CodeInput({ length = 6, value = "", onChange, onComplete, label, hint, error, disabled, autoFocus, style }) {
  const refs = React.useRef([]);
  const doneRef = React.useRef(false);
  React.useEffect(() => { if (value.length < length) doneRef.current = false; }, [value, length]);
  const chars = Array.from({ length }, (_, i) => value[i] || "");
  const set = (next) => {
    const v = next.slice(0, length);
    if (onChange) onChange(v);
    if (v.length === length && !doneRef.current) { doneRef.current = true; if (onComplete) onComplete(v); }
    if (v.length < length) doneRef.current = false;
  };
  const focusAt = (i) => { const el = refs.current[Math.max(0, Math.min(length - 1, i))]; if (el) { el.focus(); el.select(); } };
  const onKey = (i) => (e) => {
    if (e.key === "Backspace") {
      e.preventDefault();
      if (chars[i]) set(value.slice(0, i) + value.slice(i + 1));
      else { set(value.slice(0, Math.max(0, i - 1)) + value.slice(i)); focusAt(i - 1); }
    } else if (e.key === "ArrowLeft") { e.preventDefault(); focusAt(i - 1); }
    else if (e.key === "ArrowRight") { e.preventDefault(); focusAt(i + 1); }
  };
  const onInput = (i) => (e) => {
    const digits = e.target.value.replace(/\D/g, "");
    if (!digits) { e.target.value = chars[i] || ""; return; }
    const next = (value.slice(0, i) + digits + value.slice(i + 1)).replace(/\D/g, "").slice(0, length);
    set(next);
    focusAt(i + digits.length);
  };
  const onPaste = (e) => {
    e.preventDefault();
    const digits = (e.clipboardData.getData("text") || "").replace(/\D/g, "").slice(0, length);
    if (!digits) return;
    set(digits);
    focusAt(digits.length - 1);
  };
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8, fontFamily: "var(--font-ui)", ...style }}>
      {label && <span style={{ fontSize: 12.5, fontWeight: 500, color: "var(--text-body)" }}>{label}</span>}
      <div style={{ display: "flex", alignItems: "center", gap: 8 }} onPaste={onPaste}>
        {chars.map((c, i) => (
          <React.Fragment key={i}>
            {i > 0 && i % 3 === 0 && <span aria-hidden="true" style={{ width: 8, height: 1.5, background: "var(--border-strong)", borderRadius: 999, flex: "none" }} />}
            <input ref={(el) => { refs.current[i] = el; }} value={c} disabled={disabled}
              autoFocus={autoFocus && i === 0}
              inputMode="numeric" autoComplete={i === 0 ? "one-time-code" : "off"} aria-label={"Digit " + (i + 1) + " of " + length}
              aria-invalid={!!error}
              onKeyDown={onKey(i)} onChange={onInput(i)} onFocus={(e) => e.target.select()}
              style={{ width: 40, height: 48, textAlign: "center", font: "600 19px var(--font-mono)", color: "var(--text-ink)", background: "var(--surface)", border: "1.5px solid " + (error ? "var(--danger-solid)" : "var(--border-default)"), borderRadius: 12, outline: "none", caretColor: "var(--accent)", opacity: disabled ? 0.45 : 1, transition: "border-color var(--dur-fast) var(--ease-out), box-shadow var(--dur-fast) var(--ease-out)" }}
              onFocusCapture={(e) => { e.target.style.borderColor = error ? "var(--danger-solid)" : "var(--accent)"; e.target.style.boxShadow = "0 0 0 3px color-mix(in srgb, " + (error ? "var(--danger-solid)" : "var(--focus-ring)") + " 18%, transparent)"; }}
              onBlurCapture={(e) => { e.target.style.borderColor = error ? "var(--danger-solid)" : "var(--border-default)"; e.target.style.boxShadow = "none"; }} />
          </React.Fragment>
        ))}
      </div>
      {error ? <span style={{ fontSize: 11.5, color: "var(--danger)" }}>{error}</span>
        : hint ? <span style={{ fontSize: 11.5, color: "var(--text-muted)" }}>{hint}</span> : null}
    </div>
  );
}
