import * as React from "react";
export interface BulkAction {
  label: string;
  /** Lucide icon name */
  icon?: string;
  onClick?: () => void;
  /** "danger" renders the label in salmon */
  tone?: "danger";
}
export interface BulkActionsBarProps {
  /** 0 renders nothing */
  count: number;
  /** Default "selected" */
  itemLabel?: string;
  actions?: BulkAction[];
  onClear?: () => void;
  /** Fixed bottom-center (default). false = inline */
  floating?: boolean;
  style?: React.CSSProperties;
}
export function BulkActionsBar(props: BulkActionsBarProps): JSX.Element | null;
