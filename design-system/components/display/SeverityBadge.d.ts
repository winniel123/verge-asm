import * as React from "react";
/** The finding severity chip — solid fill, fixed vocabulary, 52px min width. */
export interface SeverityBadgeProps {
  /** Exact scale; never synonymize. @default "info" */
  severity: "critical" | "high" | "medium" | "low" | "info";
  style?: React.CSSProperties;
}
export declare function SeverityBadge(props: SeverityBadgeProps): React.ReactElement;
