import * as React from "react";
export interface DiffLine {
  type: "add" | "remove" | "same";
  text: string;
}
export interface DiffViewProps {
  lines: DiffLine[];
  /** Micro-label header, e.g. "Open ports \u00b7 drift" */
  title?: string;
  style?: React.CSSProperties;
}
export function DiffView(props: DiffViewProps): JSX.Element;
