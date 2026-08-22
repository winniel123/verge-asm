import React from "react";
import { Checkbox } from "../forms/Checkbox.jsx";

/* columns: [{ key, label, width, align, mono, render, clip, sortable, sortValue }] — render(row, i) overrides cell
   content; clip:false lets floating children (menus) escape; sortable + optional sortValue(row) enable header sorting.
   Multi-select: selectable + rowKey + selectedKeys + onSelectionChange (shift-click ranges).
   maxHeight makes the body scroll with a sticky header. density "comfortable" | "compact" (dense is the legacy alias).
   virtual + rowHeight window the body to visible rows (requires maxHeight; fixed row height).
   onRowClick enables keyboard nav: j/k or arrows move a roving focus ring, Enter opens.
   onRowContextMenu(row, i, event) pairs with a controlled <ContextMenu>. */
export function Table({ columns = [], rows = [], selectedIndex = -1, onRowClick, onRowContextMenu, dense, density, framed = true, rowKey, selectable, selectedKeys = [], onSelectionChange, initialSort, maxHeight, virtual, rowHeight = 37, style }) {
  const [hov, setHov] = React.useState(-1);
  const [sort, setSort] = React.useState(initialSort || null);
  const lastIdx = React.useRef(null);
  const [scrollTop, setScrollTop] = React.useState(0);
  const [viewH, setViewH] = React.useState(0);
  const scrollRef = React.useRef(null);
  React.useEffect(() => { if (virtual && scrollRef.current) setViewH(scrollRef.current.clientHeight); }, [virtual, maxHeight]);
  const compact = density ? density === "compact" : dense;
  const padY = compact ? 7 : 10;
  const [focusIdx, setFocusIdx] = React.useState(-1);
  const [kbOn, setKbOn] = React.useState(false);
  const onKeyNav = (e) => {
    if (!onRowClick || /INPUT|TEXTAREA|SELECT/.test((e.target.tagName || ""))) return;
    const k = e.key;
    if (k === "j" || k === "ArrowDown" || k === "k" || k === "ArrowUp" || k === "Home" || k === "End") {
      e.preventDefault();
      setFocusIdx((f) => {
        const n = view.length - 1;
        const next = k === "Home" ? 0 : k === "End" ? n : k === "j" || k === "ArrowDown" ? Math.min(n, f + 1) : Math.max(0, f - 1);
        if (virtual && scrollRef.current) {
          const el = scrollRef.current, top = next * rowHeight, bot = top + rowHeight;
          if (top < el.scrollTop) el.scrollTop = top;
          else if (bot > el.scrollTop + el.clientHeight) el.scrollTop = bot - el.clientHeight;
        }
        return next;
      });
    } else if (k === "Enter") {
      e.preventDefault();
      setFocusIdx((f) => { if (f >= 0 && view[f]) onRowClick(view[f], f); return f; });
    }
  };
  const view = React.useMemo(() => {
    if (!sort) return rows;
    const col = columns.find((c) => c.key === sort.key);
    if (!col) return rows;
    const val = (r) => (col.sortValue ? col.sortValue(r) : r[sort.key]);
    const out = rows.slice().sort((a, b) => {
      const x = val(a), y = val(b);
      if (typeof x === "number" && typeof y === "number") return x - y;
      return String(x ?? "").localeCompare(String(y ?? ""));
    });
    return sort.dir === "desc" ? out.reverse() : out;
  }, [rows, sort, columns]);
  const keyOf = (r, i) => (rowKey ? r[rowKey] : i);
  const isSel = (r, i) => selectedKeys.indexOf(keyOf(r, i)) !== -1;
  const cell = (col) => ({ padding: padY + "px 16px", textAlign: col.align || "left", width: col.width, fontFamily: col.mono ? "var(--font-mono)" : "var(--font-ui)", fontSize: col.mono ? 12.5 : 13, color: "var(--text-body)", verticalAlign: "middle", whiteSpace: "nowrap", overflow: col.clip === false ? "visible" : "hidden", textOverflow: col.clip === false ? "clip" : "ellipsis", maxWidth: col.clip === false ? "none" : col.width || "auto" });
  const toggleRow = (i, shift) => {
    if (!onSelectionChange) return;
    const key = keyOf(view[i], i);
    let next;
    if (shift && lastIdx.current != null && lastIdx.current !== i) {
      const [a, b] = [Math.min(lastIdx.current, i), Math.max(lastIdx.current, i)];
      const range = view.slice(a, b + 1).map((r, j) => keyOf(r, a + j));
      next = Array.from(new Set(selectedKeys.concat(range)));
    } else {
      next = selectedKeys.indexOf(key) !== -1 ? selectedKeys.filter((k) => k !== key) : selectedKeys.concat(key);
    }
    lastIdx.current = i;
    onSelectionChange(next);
  };
  const allSel = view.length > 0 && view.every((r, i) => isSel(r, i));
  const someSel = !allSel && view.some((r, i) => isSel(r, i));
  const toggleAll = () => { if (onSelectionChange) onSelectionChange(allSel ? [] : view.map((r, i) => keyOf(r, i))); };
  const cycleSort = (c) => {
    if (!c.sortable) return;
    setSort((s) => (!s || s.key !== c.key ? { key: c.key, dir: "asc" } : s.dir === "asc" ? { key: c.key, dir: "desc" } : null));
  };
  const sticky = maxHeight != null;
  const thBase = { padding: "8px 16px", font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)", borderBottom: "1px solid var(--border-default)", whiteSpace: "nowrap", background: "var(--surface)", position: sticky ? "sticky" : "static", top: 0, zIndex: sticky ? 2 : "auto" };
  const checkCell = { width: 36, padding: padY + "px 0 " + padY + "px 16px", verticalAlign: "middle" };
  const caret = (c) => {
    if (!c.sortable) return null;
    const on = sort && sort.key === c.key;
    return (
      <svg viewBox="0 0 10 14" width="8" height="11" style={{ marginLeft: 5, verticalAlign: "-1px" }}>
        <path d="M5 1.5L8 5H2Z" fill={on && sort.dir === "asc" ? "var(--accent)" : "var(--border-strong)"}></path>
        <path d="M5 12.5L2 9h6Z" fill={on && sort.dir === "desc" ? "var(--accent)" : "var(--border-strong)"}></path>
      </svg>
    );
  };
  const windowed = virtual && maxHeight != null;
  let start = 0, end = view.length;
  if (windowed) {
    start = Math.max(0, Math.floor(scrollTop / rowHeight) - 8);
    end = Math.min(view.length, Math.ceil((scrollTop + (viewH || 480)) / rowHeight) + 8);
  }
  const slice = windowed ? view.slice(start, end) : view;
  const spanAll = columns.length + (selectable ? 1 : 0);
  const table = (
    <table style={{ width: "100%", borderCollapse: "separate", borderSpacing: 0, tableLayout: windowed ? "fixed" : "auto" }}>
      <thead>
        <tr>
          {selectable && (
            <th style={{ ...checkCell, ...thBase, width: 36, padding: padY + "px 0 " + padY + "px 16px" }}>
              <span onClick={toggleAll} style={{ display: "flex", alignItems: "center", cursor: "pointer" }}>
                <Checkbox checked={allSel} indeterminate={someSel} onChange={() => {}} style={{ pointerEvents: "none" }} />
              </span>
            </th>
          )}
          {columns.map((c) => (
            <th key={c.key} onClick={() => cycleSort(c)}
              style={{ ...thBase, textAlign: c.align || "left", cursor: c.sortable ? "pointer" : "default", userSelect: "none" }}>
              {c.label}{caret(c)}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {windowed && start > 0 && <tr aria-hidden="true" style={{ height: start * rowHeight }}><td colSpan={spanAll} style={{ padding: 0, border: "none" }}></td></tr>}
        {slice.map((r, j) => {
          const i = start + j;
          const selected = i === selectedIndex || isSel(r, i);
          const interactive = onRowClick || selectable;
          return (
            <tr key={keyOf(r, i)}
              onClick={onRowClick ? () => { setFocusIdx(i); onRowClick(r, i); } : undefined}
              onContextMenu={onRowContextMenu ? (e) => { e.preventDefault(); onRowContextMenu(r, i, e); } : undefined}
              onMouseEnter={() => setHov(i)} onMouseLeave={() => setHov(-1)}
              style={{ background: selected ? "var(--row-selected)" : hov === i && interactive ? "var(--surface-sunken)" : "transparent", cursor: onRowClick ? "pointer" : "default", position: "relative", height: windowed ? rowHeight : "auto", boxShadow: kbOn && focusIdx === i ? "inset 0 0 0 1.5px var(--accent)" : "none", transition: "background var(--dur-fast) var(--ease-out)" }}>
              {selectable && (
                <td style={{ ...checkCell, borderTop: i === 0 ? "none" : "1px solid var(--row-sep)" }}
                  onClick={(e) => { e.stopPropagation(); toggleRow(i, e.shiftKey); }}>
                  <span style={{ display: "flex", alignItems: "center", cursor: "pointer" }}>
                    <Checkbox checked={isSel(r, i)} onChange={() => {}} style={{ pointerEvents: "none" }} />
                  </span>
                </td>
              )}
              {columns.map((c, ci) => (
                <td key={c.key} style={{ ...cell(c), borderTop: i === 0 ? "none" : "1px solid var(--row-sep)", position: ci === 0 ? "relative" : "static" }}>
                  {ci === 0 && !selectable && selected && <span style={{ position: "absolute", left: 4, top: 6, bottom: 6, width: 3, borderRadius: 999, background: "var(--accent)" }} />}
                  {c.render ? c.render(r, i) : r[c.key]}
                </td>
              ))}
            </tr>
          );
        })}
        {windowed && end < view.length && <tr aria-hidden="true" style={{ height: (view.length - end) * rowHeight }}><td colSpan={spanAll} style={{ padding: 0, border: "none" }}></td></tr>}
      </tbody>
    </table>
  );
  const body = sticky ? <div ref={scrollRef} onScroll={windowed ? (e) => setScrollTop(e.currentTarget.scrollTop) : undefined} style={{ maxHeight, overflowY: "auto" }}>{table}</div> : table;
  const kbProps = onRowClick ? { tabIndex: 0, onKeyDown: onKeyNav, onFocus: () => setKbOn(true), onBlur: () => { setKbOn(false); }, style: { outline: "none" } } : {};
  if (!framed) return <div {...kbProps} style={{ ...(kbProps.style || {}) }}>{body}</div>;
  return <div {...kbProps} style={{ background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 16, boxShadow: "var(--shadow-xs)", overflow: "hidden", outline: "none", ...style }}>{body}</div>;
}
