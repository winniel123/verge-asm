import * as React from "react";
export interface SliderProps {
  label?: string;
  value?: number;
  onChange?: (value: number) => void;
  min?: number;
  max?: number;
  step?: number;
  /** Readout suffix, e.g. "ms" */
  unit?: string;
  style?: React.CSSProperties;
}
export function Slider(props: SliderProps): JSX.Element;
