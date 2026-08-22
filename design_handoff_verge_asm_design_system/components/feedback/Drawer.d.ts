import * as React from "react";
export interface DrawerProps {
  open: boolean;
  title: string;
  description?: string;
  /** Pinned action row at the bottom */
  footer?: React.ReactNode;
  /** Scrim click, Escape, and the close control all call this */
  onClose?: () => void;
  /** Panel width px. Default 440 */
  width?: number;
  side?: "right" | "left";
  children?: React.ReactNode;
}
export function Drawer(props: DrawerProps): JSX.Element | null;
