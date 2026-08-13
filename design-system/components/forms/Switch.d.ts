import * as React from "react";
/** Square-cornered toggle, 34×18. Ink track when on. */
export interface SwitchProps {
  label?: React.ReactNode;
  checked?: boolean;
  onChange?: (checked: boolean) => void;
  disabled?: boolean;
  style?: React.CSSProperties;
}
export declare function Switch(props: SwitchProps): React.ReactElement;
