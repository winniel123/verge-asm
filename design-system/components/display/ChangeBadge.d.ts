import * as React from "react";
export interface ChangeBadgeProps {
  /** Exactly the change vocabulary — never severity words */
  change: "appeared" | "revealed" | "withdrawn" | "descoped" | "returned" | "changed";
  /** Closure reason, e.g. "operator excluded subtree" — why it left, not just that it did */
  reason?: string;
  size?: "md" | "sm";
  style?: React.CSSProperties;
}
export function ChangeBadge(props: ChangeBadgeProps): JSX.Element;
/** Bare glyph for the six change kinds (used by TransitionMarker) */
export function ChangeGlyph(props: { change: ChangeBadgeProps["change"]; size?: number }): JSX.Element;
