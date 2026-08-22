import * as React from "react";
export interface PickableColumn { key: string; label: string; /** Can't be hidden (identity columns) */ locked?: boolean; }
export interface ColumnPickerProps {
  columns: PickableColumn[];
  /** Keys currently shown */
  visible: string[];
  onChange?: (visibleKeys: string[]) => void;
  label?: string;
  size?: "sm" | "md";
  align?: "start" | "end";
}
export function ColumnPicker(props: ColumnPickerProps): JSX.Element;
