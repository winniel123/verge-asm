import * as React from "react";
export interface MiddleTruncateProps {
  value: string;
  /** Characters kept visible at the end. Default 12 */
  tail?: number;
  /** Default true (hostnames are technical values) */
  mono?: boolean;
  style?: React.CSSProperties;
}
export function MiddleTruncate(props: MiddleTruncateProps): JSX.Element;
