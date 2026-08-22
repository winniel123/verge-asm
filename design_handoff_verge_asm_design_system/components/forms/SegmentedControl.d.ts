import * as React from "react";
export interface SegmentedOption {
  value: string;
  label: React.ReactNode;
  icon?: React.ReactNode;
}
export interface SegmentedControlProps {
  /** 2-4 options; strings are shorthand for {value,label} */
  options: (string | SegmentedOption)[];
  value?: string;
  onChange?: (value: string) => void;
  size?: "sm" | "md";
  /** aria-label for the group */
  label?: string;
  style?: React.CSSProperties;
}
export function SegmentedControl(props: SegmentedControlProps): JSX.Element;
