import * as React from "react";
export interface ExposureBadgeProps {
  /** Reach states — never a numeric rating */
  state?: "exposed" | "firewalled" | "not-reached" | "unverified";
  size?: "md" | "sm";
  style?: React.CSSProperties;
}
export function ExposureBadge(props: ExposureBadgeProps): JSX.Element;
