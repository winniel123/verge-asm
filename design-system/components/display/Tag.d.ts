import * as React from "react";
/** Filter/label chip; optionally removable or toggleable. */
export interface TagProps {
  /** Ink-filled selected state. @default false */
  active?: boolean;
  /** Shows a ✕; called on remove click. */
  onRemove?: () => void;
  onClick?: () => void;
  style?: React.CSSProperties;
  children?: React.ReactNode;
}
export declare function Tag(props: TagProps): React.ReactElement;
