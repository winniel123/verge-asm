import * as React from "react";
export interface NumberInputProps {
  label?: string;
  value?: number | "";
  onChange?: (value: number | "") => void;
  min?: number;
  max?: number;
  step?: number;
  /** Mono suffix, e.g. "addresses" */
  unit?: string;
  hint?: string;
  style?: React.CSSProperties;
}
export function NumberInput(props: NumberInputProps): JSX.Element;
