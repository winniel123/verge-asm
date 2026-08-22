import * as React from "react";
export interface PopoverProps {
  /** The anchor element; click toggles */
  trigger: React.ReactNode;
  /** Controlled open state (optional) */
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  align?: "start" | "end";
  side?: "bottom" | "top";
  /** Panel width px. Default 260 */
  width?: number;
  style?: React.CSSProperties;
  children?: React.ReactNode;
}
export function Popover(props: PopoverProps): JSX.Element;
