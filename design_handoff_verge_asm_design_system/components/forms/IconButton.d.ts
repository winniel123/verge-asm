import * as React from "react";
export interface IconButtonProps {
  /** Lucide icon name */
  icon: string;
  /** Accessible label (required — becomes aria-label + title) */
  label: string;
  variant?: "ghost" | "secondary" | "primary";
  /** sm 26 / md 32 / lg 36. Default "md" */
  size?: "sm" | "md" | "lg";
  /** Keeps the hover treatment on (e.g. open menu) */
  active?: boolean;
  disabled?: boolean;
  onClick?: () => void;
  style?: React.CSSProperties;
}
export function IconButton(props: IconButtonProps): JSX.Element;
