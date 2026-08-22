import * as React from "react";
export interface ConfirmDialogProps {
  open: boolean;
  title: string;
  /** The fact, e.g. "Removes the seed and its subjects from scope." */
  message?: string;
  detail?: string;
  /** Imperative, e.g. "Descope seed" */
  confirmLabel?: string;
  tone?: "danger" | "primary";
  /** Require typing this exact value (worst acts only) */
  typedConfirm?: string;
  onConfirm?: () => void;
  onClose?: () => void;
}
export function ConfirmDialog(props: ConfirmDialogProps): JSX.Element;
