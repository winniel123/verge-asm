import * as React from "react";
export interface SpinnerProps {
  /** Diameter px. Default 16 */
  size?: number;
  label?: string;
  style?: React.CSSProperties;
}
export function Spinner(props: SpinnerProps): JSX.Element;
