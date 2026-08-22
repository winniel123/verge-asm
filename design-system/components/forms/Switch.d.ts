import * as React from "react";
export interface SwitchProps {
  checked?: boolean;
  onChange?: (checked: boolean) => void;
  label?: string;
  disabled?: boolean;
  style?: React.CSSProperties;
}
export function Switch(props: SwitchProps): JSX.Element;
