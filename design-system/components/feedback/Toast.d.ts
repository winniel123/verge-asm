import * as React from "react";
export interface ToastProps {
  title: string;
  description?: string;
  tone?: "neutral" | "ok" | "warn" | "danger";
  /** Optional action, e.g. a ghost Button "Undo" */
  action?: React.ReactNode;
  onDismiss?: () => void;
  /** Fixed bottom-right with slide-up entrance */
  floating?: boolean;
  style?: React.CSSProperties;
}
export function Toast(props: ToastProps): JSX.Element;
