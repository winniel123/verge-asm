import * as React from "react";
/** 15px radio; the one round control (dots may be round). */
export interface RadioProps {
  label?: React.ReactNode;
  checked?: boolean;
  /** Called with this radio's value when selected. */
  onChange?: (value: string) => void;
  name?: string;
  value?: string;
  disabled?: boolean;
  style?: React.CSSProperties;
}
export declare function Radio(props: RadioProps): React.ReactElement;
