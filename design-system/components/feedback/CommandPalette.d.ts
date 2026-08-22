import * as React from "react";
export interface CommandItem {
  id?: string;
  label: string;
  /** Lucide icon name */
  icon?: string;
  /** Mono right-aligned hint (shortcut, asset id) */
  hint?: string;
  onSelect?: () => void;
}
export interface CommandGroup { label: string; items: CommandItem[]; }
export interface CommandPaletteProps {
  open: boolean;
  onClose?: () => void;
  groups: CommandGroup[];
  placeholder?: string;
}
export function CommandPalette(props: CommandPaletteProps): JSX.Element | null;
