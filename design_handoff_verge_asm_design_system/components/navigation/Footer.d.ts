import * as React from "react";
export interface FooterProps {
  /** console = one-line chrome; marketing = link columns */
  variant?: "console" | "marketing";
  version?: string;
  style?: React.CSSProperties;
}
export function Footer(props: FooterProps): JSX.Element;
