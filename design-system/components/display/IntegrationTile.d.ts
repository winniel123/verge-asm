import * as React from "react";
export interface IntegrationTileProps {
  name: string;
  category: string;
  description: string;
  /** 1-2 letter neutral mark — not a brand logo. */
  mark: string;
  state?: "installed" | "available" | "attention";
  onClick?: () => void;
  style?: React.CSSProperties;
}
export declare function IntegrationTile(props: IntegrationTileProps): JSX.Element;
