import * as React from "react";
export interface SelectOption { value: string; label: string; }
export interface SelectProps {
  label?: string;
  /** Strings or {value, label} pairs */
  options: Array<string | SelectOption>;
  value?: string;
  defaultValue?: string;
  /** Event-shaped for drop-in compatibility: onChange({ target: { value } }) */
  onChange?: (e: { target: { value: string } }) => void;
  size?: "sm" | "md";
  disabled?: boolean;
  style?: React.CSSProperties;
}
export function Select(props: SelectProps): JSX.Element;
