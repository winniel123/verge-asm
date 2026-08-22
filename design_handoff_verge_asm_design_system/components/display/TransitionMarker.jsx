import React from "react";
import { Tooltip } from "../feedback/Tooltip.jsx";
import { ChangeGlyph } from "./ChangeBadge.jsx";

const FAMILY = { appeared: "gain", revealed: "gain", returned: "gain", withdrawn: "loss", descoped: "loss", changed: "change" };

/* Inline latest-transition marker for subject rows; full transition on hover. */
export function TransitionMarker({ change = "changed", time, reason, style }) {
  const fam = FAMILY[change] || "change";
  const tip = change + (time ? " · " + time : "") + (reason ? " · " + reason : "");
  return (
    <Tooltip content={tip} mono>
      <span style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", width: 18, height: 18, borderRadius: 6, background: "var(--drift-" + fam + "-bg)", border: "1px solid var(--drift-" + fam + "-border)", color: "var(--drift-" + fam + "-fg)", cursor: "default", ...style }}>
        <ChangeGlyph change={change} size={9} />
      </span>
    </Tooltip>
  );
}
