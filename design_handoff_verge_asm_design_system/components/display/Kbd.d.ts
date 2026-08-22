import * as React from "react";
export interface KbdProps {
  /** One cap ("\u2318K") or one cap per array entry */
  keys: string | string[];
  style?: React.CSSProperties;
}
export function Kbd(props: KbdProps): JSX.Element;
