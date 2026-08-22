import * as React from "react";
export interface TransitionMarkerProps {
  change: "appeared" | "revealed" | "withdrawn" | "descoped" | "returned" | "changed";
  /** Terse relative time for the tooltip */
  time?: string;
  reason?: string;
  style?: React.CSSProperties;
}
export function TransitionMarker(props: TransitionMarkerProps): JSX.Element;
