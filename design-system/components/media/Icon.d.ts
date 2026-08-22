import * as React from "react";
export interface IconProps {
  /** Lucide icon name, e.g. "radar", "globe", "shield-alert" */
  name: string;
  /** Square size in px. Default 16 */
  size?: number;
  /** Default 1.75 */
  strokeWidth?: number;
  style?: React.CSSProperties;
}
export function Icon(props: IconProps): JSX.Element;
