import * as React from "react";
/** 15px square checkbox; ink fill with ✓ when checked. */
export interface CheckboxProps {
  label?: React.ReactNode;
  checked?: boolean;
  /** Called with the next boolean value. */
  onChange?: (checked: boolean) => void;
  disabled?: boolean;
  style?: React.CSSProperties;
}
export declare function Checkbox(props: CheckboxProps): React.ReactElement;
