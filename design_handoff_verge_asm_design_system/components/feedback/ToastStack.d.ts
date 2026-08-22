import * as React from "react";
export interface StackToast {
  id: string;
  tone?: "neutral" | "ok" | "warn" | "danger";
  title: string;
  description?: string;
  action?: React.ReactNode;
  /** Per-toast override of the stack ttl */
  ttl?: number;
}
export interface ToastStackProps {
  toasts: StackToast[];
  onDismiss?: (id: string) => void;
  /** Auto-dismiss ms. Default 5000 */
  ttl?: number;
  style?: React.CSSProperties;
}
export function ToastStack(props: ToastStackProps): JSX.Element;
