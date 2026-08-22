import * as React from "react";
export interface StalenessBadgeProps {
  /** stale = old observation; not-evaluable; silent = "you stopped telling us" */
  kind?: "stale" | "not-evaluable" | "silent";
  /** Currency bound, e.g. "9d" */
  bound?: string;
  size?: "md" | "sm";
  style?: React.CSSProperties;
}
export function StalenessBadge(props: StalenessBadgeProps): JSX.Element;
