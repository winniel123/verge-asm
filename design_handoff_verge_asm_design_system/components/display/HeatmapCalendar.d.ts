import * as React from "react";
export interface HeatmapCalendarProps {
  /** One count per day, oldest first; columns are weeks (7 per column) */
  values: number[];
  /** Cell size px, default 12 */
  cell?: number;
  gap?: number;
  /** aria-label for the grid */
  label?: string;
  /** Tooltip unit, default "scans" */
  unit?: string;
  startLabel?: string;
  /** Default "today" */
  endLabel?: string;
  style?: React.CSSProperties;
}
export function HeatmapCalendar(props: HeatmapCalendarProps): JSX.Element;
