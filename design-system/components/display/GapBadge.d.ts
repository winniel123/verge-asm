import * as React from "react";
export interface GapBadgeProps {
  /** Default "gap"; name the absent facet, e.g. "no address" */
  label?: string;
  size?: "md" | "sm";
  style?: React.CSSProperties;
}
export function GapBadge(props: GapBadgeProps): JSX.Element;
