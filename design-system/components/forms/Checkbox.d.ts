import * as React from "react";
export interface CheckboxProps {
  label?: string;
  /** Muted second line */
  description?: string;
  checked?: boolean;
  /** Dash state (partial selection) */
  indeterminate?: boolean;
  onChange?: (checked: boolean) => void;
  disabled?: boolean;
  style?: React.CSSProperties;
}
export function Checkbox(props: CheckboxProps): JSX.Element;
