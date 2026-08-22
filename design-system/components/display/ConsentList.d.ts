import * as React from "react";
export interface Grant {
  scope: string;
  detail?: string;
  /** Write grants render louder (warn tint + "writes" tag). */
  write?: boolean;
}
export interface ConsentListProps { grants: Grant[]; style?: React.CSSProperties; }
export declare function ConsentList(props: ConsentListProps): JSX.Element;
