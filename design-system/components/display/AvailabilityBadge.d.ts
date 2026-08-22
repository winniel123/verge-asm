import * as React from "react";
export interface AvailabilityBadgeProps {
  state?: "available" | "degraded" | "unavailable" | "unverified";
  size?: "md" | "sm";
  style?: React.CSSProperties;
}
export function AvailabilityBadge(props: AvailabilityBadgeProps): JSX.Element;
