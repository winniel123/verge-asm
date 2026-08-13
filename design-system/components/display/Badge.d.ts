import * as React from "react";
/** Mono uppercase status chip (soft fill by default). */
export interface BadgeProps {
  /** @default "neutral" */
  tone?: "neutral" | "accent" | "ok" | "warn" | "danger";
  /** Solid fill with white text (ink for neutral). @default false */
  solid?: boolean;
  style?: React.CSSProperties;
  children?: React.ReactNode;
}
export declare function Badge(props: BadgeProps): React.ReactElement;
