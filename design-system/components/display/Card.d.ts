import * as React from "react";
/** @startingPoint section="Components" subtitle="Surface card with micro-label header" viewport="700x240" */
export interface CardProps {
  /** Mono uppercase eyebrow, e.g. "Open signals" */
  microLabel?: string;
  title?: string;
  /** Right-aligned header actions */
  action?: React.ReactNode;
  footer?: React.ReactNode;
  /** Default 20 */
  pad?: number;
  /** Default: "visible" for padded cards, "hidden" when pad is 0 (flush tables need corner clipping) */
  overflow?: "hidden" | "visible";
  style?: React.CSSProperties;
  children?: React.ReactNode;
}
export function Card(props: CardProps): JSX.Element;
