import * as React from "react";
export interface BadgeProps {
  tone?: "neutral" | "accent" | "ok" | "warn" | "danger";
  /** Leading 6px dot in the tone color */
  dot?: boolean;
  style?: React.CSSProperties;
  children?: React.ReactNode;
}
export function Badge(props: BadgeProps): JSX.Element;
