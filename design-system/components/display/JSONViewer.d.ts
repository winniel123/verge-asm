import * as React from "react";
export interface JSONViewerProps {
  /** Any JSON-serializable value */
  data: unknown;
  /** Levels expanded initially. Default 1 */
  defaultDepth?: number;
  label?: string;
  style?: React.CSSProperties;
}
export function JSONViewer(props: JSONViewerProps): JSX.Element;
