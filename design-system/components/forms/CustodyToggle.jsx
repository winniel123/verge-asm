import React from "react";
import { Switch } from "./Switch.jsx";
import { CoverageMeter } from "../display/CoverageMeter.jsx";

export function CustodyToggle({ enabled, onChange, censusCount, unit = "addresses", detail, style }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12, fontFamily: "var(--font-ui)", ...style }}>
      <Switch checked={enabled} onChange={onChange} label="Extend custody to adjacent infrastructure" />
      {enabled && (
        <CoverageMeter label="Custody extension" counted={censusCount || 0} unit={unit} size="sm"
          detail={detail || "recomputed each batch \u2014 read-only, never per-address approval"} />
      )}
    </div>
  );
}
