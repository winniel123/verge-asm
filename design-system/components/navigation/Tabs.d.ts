import * as React from "react";
/** Underline tabs: 2px accent under the active item. */
export interface TabsProps {
  /** Tab labels, sentence case. */
  items: string[];
  /** Active label (controlled). */
  value?: string;
  onChange?: (label: string, index: number) => void;
  style?: React.CSSProperties;
}
export declare function Tabs(props: TabsProps): React.ReactElement;
