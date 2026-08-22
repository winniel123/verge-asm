import * as React from "react";
export interface ComboOption { value: string; label: string; hint?: string; }
export interface ComboboxProps {
  label?: string;
  options: Array<string | ComboOption>;
  value?: string;
  onChange?: (value: string) => void;
  placeholder?: string;
  style?: React.CSSProperties;
}
export function Combobox(props: ComboboxProps): JSX.Element;
