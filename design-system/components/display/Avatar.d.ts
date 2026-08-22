import * as React from "react";
export interface AvatarProps {
  /** Full name; initials are derived (max 2) */
  name: string;
  /** Diameter px. Default 28 */
  size?: number;
  /** Presence/status dot */
  dot?: "ok" | "warn" | "danger" | "accent";
  style?: React.CSSProperties;
}
export function Avatar(props: AvatarProps): JSX.Element;
