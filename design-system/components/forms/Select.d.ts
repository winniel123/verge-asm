import * as React from "react";
/** Native select styled to the system; ▾ chevron. */
export interface SelectProps extends Omit<React.SelectHTMLAttributes<HTMLSelectElement>, "size" | "style"> {
  label?: React.ReactNode;
  hint?: React.ReactNode;
  /** sm=26px, md=32px. @default "md" */
  size?: "sm" | "md" | "lg";
  /** Strings or {value,label} pairs. */
  options?: Array<string | { value: string; label: string }>;
  style?: React.CSSProperties;
}
export declare function Select(props: SelectProps): React.ReactElement;
