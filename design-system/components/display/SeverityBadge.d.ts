import * as React from "react";
export interface SeverityBadgeProps {
  /** Exactly these five levels — never synonymize */
  level: "critical" | "high" | "medium" | "low" | "info";
  /** md 22px / sm 18px (dense tables) */
  size?: "md" | "sm";
  style?: React.CSSProperties;
}
export function SeverityBadge(props: SeverityBadgeProps): JSX.Element;
