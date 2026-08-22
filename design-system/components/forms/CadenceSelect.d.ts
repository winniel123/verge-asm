import * as React from "react";
export interface CadenceSelectProps {
  label?: string;
  /** One of the presets, or "Custom\u2026" */
  value?: string;
  /** Cron line when value is Custom */
  customValue?: string;
  onChange?: (value: string, customValue?: string) => void;
  style?: React.CSSProperties;
}
export function CadenceSelect(props: CadenceSelectProps): JSX.Element;
