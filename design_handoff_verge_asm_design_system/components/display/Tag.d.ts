import * as React from "react";
export interface TagProps {
  /** Shows a remove control */
  onRemove?: () => void;
  style?: React.CSSProperties;
  children?: React.ReactNode;
}
export function Tag(props: TagProps): JSX.Element;
