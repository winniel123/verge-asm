import * as React from "react";
export interface CalendarProps {
  /** "YYYY-MM" — initial visible month (falls back to selected, then today) */
  month?: string;
  /** "YYYY-MM-DD" — controlled selection */
  selected?: string;
  onSelect?: (isoDate: string) => void;
  /** ISO date → count; renders up to 3 volume dots + tooltip */
  events?: Record<string, number>;
  /** ISO bounds — dates outside are disabled */
  min?: string;
  max?: string;
  /** Cell size px, default 36 */
  cell?: number;
  /** Tooltip unit, default "run" */
  unit?: string;
  /** aria-label for the grid */
  label?: string;
  style?: React.CSSProperties;
}
export function Calendar(props: CalendarProps): JSX.Element;
