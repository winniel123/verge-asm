import * as React from "react";
/** Transient notice: white panel, ink border, hard-sm shadow, status dot. Consumer positions/stacks it. */
export interface ToastProps {
  /** @default "neutral" */
  tone?: "ok" | "warn" | "danger" | "neutral" | "accent";
  title: React.ReactNode;
  detail?: React.ReactNode;
  onDismiss?: () => void;
  style?: React.CSSProperties;
}
export declare function Toast(props: ToastProps): React.ReactElement;
