import * as React from "react";
export interface ContextMenuItem {
  label: string;
  icon?: string;
  /** "danger" for destructive acts (still confirm via ConfirmDialog) */
  tone?: "danger";
  shortcut?: string;
  disabled?: boolean;
  onSelect?: () => void;
}
export interface ContextMenuProps {
  items: (ContextMenuItem | "-")[];
  /** Wrapper mode: right-clicking these children opens the menu at the cursor */
  children?: React.ReactNode;
  /** Controlled mode (no children): open at viewport x/y — pair with Table onRowContextMenu */
  open?: boolean;
  x?: number;
  y?: number;
  onClose?: () => void;
}
export function ContextMenu(props: ContextMenuProps): JSX.Element | null;
