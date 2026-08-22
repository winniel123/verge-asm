import * as React from "react";
export interface FileDropProps {
  /** Default "Drop a CSV here or click to browse" */
  label?: string;
  hint?: string;
  /** e.g. ".csv" */
  accept?: string;
  multiple?: boolean;
  onFiles?: (files: File[]) => void;
  /** Tighter padding for dialogs */
  compact?: boolean;
  style?: React.CSSProperties;
}
export function FileDrop(props: FileDropProps): JSX.Element;
