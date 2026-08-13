import * as React from "react";
/** KPI block: micro-label, 26px mono value, signed delta.
 * @startingPoint section="Components" subtitle="KPI numeral with label and delta" viewport="700x150"
 */
export interface StatProps {
  /** Micro-label caption, e.g. "Open findings". */
  label: React.ReactNode;
  /** The numeral — string with separators: "1,284". */
  value: React.ReactNode;
  /** Signed delta line: "+12 in 24h" (true minus for negatives). */
  delta?: React.ReactNode;
  /** Color of the delta line. @default "neutral" */
  deltaTone?: "ok" | "danger" | "neutral";
  /** "danger" paints the value red (critical counts). */
  tone?: "danger";
  style?: React.CSSProperties;
}
export declare function Stat(props: StatProps): React.ReactElement;
