import * as React from "react";
export interface CustodyToggleProps {
  enabled?: boolean;
  onChange?: (enabled: boolean) => void;
  /** The recomputed extension census (display only) */
  censusCount?: number;
  unit?: string;
  detail?: string;
  style?: React.CSSProperties;
}
export function CustodyToggle(props: CustodyToggleProps): JSX.Element;
