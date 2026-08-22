import * as React from "react";
export interface DropdownMenuItem {
  label: string;
  /** Lucide icon name */
  icon?: string;
  onSelect?: () => void;
  /** "danger" renders red with danger-soft hover */
  tone?: "danger";
  /** Mono hint, e.g. "\u2318E" */
  shortcut?: string;
  disabled?: boolean;
}
export interface DropdownMenuProps {
  trigger: React.ReactNode;
  /** Items, or "-" for a separator */
  items: Array<DropdownMenuItem | "-">;
  align?: "start" | "end";
  width?: number;
  style?: React.CSSProperties;
}
export function DropdownMenu(props: DropdownMenuProps): JSX.Element;
