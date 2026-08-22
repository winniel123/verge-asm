import * as React from "react";
export interface StepItem {
  title: string;
  detail?: string;
}
export interface StepperProps {
  steps: StepItem[];
  /** Index of the current step; earlier steps render as done */
  active?: number;
  style?: React.CSSProperties;
}
export function Stepper(props: StepperProps): JSX.Element;
