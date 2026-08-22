import * as React from "react";
export interface BannerProps {
  tone?: "neutral" | "accent" | "ok" | "warn" | "danger";
  title?: string;
  /** Optional right-side action (small Button) */
  action?: React.ReactNode;
  onDismiss?: () => void;
  /** Override the tone's default Lucide icon */
  icon?: string;
  style?: React.CSSProperties;
  children?: React.ReactNode;
}
export function Banner(props: BannerProps): JSX.Element;
