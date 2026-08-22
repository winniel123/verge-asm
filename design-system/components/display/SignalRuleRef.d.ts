import * as React from "react";
export interface SignalRuleRefProps {
  /** Rule id, e.g. "vnc-exposure" */
  id: string;
  version: number | string;
  onClick?: () => void;
  style?: React.CSSProperties;
}
export function SignalRuleRef(props: SignalRuleRefProps): JSX.Element;
