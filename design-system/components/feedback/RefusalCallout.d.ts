import * as React from "react";
export interface RefusalCalloutProps {
  /** The refused declaration, verbatim */
  input: string;
  /** Why it was refused */
  reason: string;
  /** The largest acceptable set, named — never applied automatically */
  reachable?: string;
  style?: React.CSSProperties;
}
export function RefusalCallout(props: RefusalCalloutProps): JSX.Element;
