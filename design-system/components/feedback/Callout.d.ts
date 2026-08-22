import * as React from "react";
export interface CalloutProps {
  tone?: "accent" | "neutral" | "ok" | "warn";
  /** Override the tone's Lucide icon */
  icon?: string;
  title?: string;
  style?: React.CSSProperties;
  children?: React.ReactNode;
}
export function Callout(props: CalloutProps): JSX.Element;
