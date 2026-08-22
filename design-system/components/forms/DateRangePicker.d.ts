import * as React from "react";
export interface DateRange {
  /** Display label ("Last 7d" or "2026-08-01 \u2013 2026-08-22") */
  label?: string;
  /** ISO 8601 for custom ranges */
  start?: string;
  end?: string;
}
export interface DateRangePickerProps {
  value?: DateRange;
  onChange?: (range: DateRange) => void;
  /** Default: Last 24h / 7d / 30d / 90d */
  presets?: string[];
  /** Panel edge alignment relative to the trigger. Default "end" */
  align?: "start" | "end";
  style?: React.CSSProperties;
}
export function DateRangePicker(props: DateRangePickerProps): JSX.Element;
