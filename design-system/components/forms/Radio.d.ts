import * as React from "react";
export interface RadioProps {
  label?: string;
  checked?: boolean;
  /** Called when this option is picked */
  onChange?: () => void;
  /** Group name */
  name?: string;
  disabled?: boolean;
  style?: React.CSSProperties;
}
export function Radio(props: RadioProps): JSX.Element;
