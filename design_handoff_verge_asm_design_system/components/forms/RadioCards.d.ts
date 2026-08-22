import * as React from "react";
export interface RadioCardOption {
  value: string;
  title: string;
  description?: string;
  /** Small mono note in the corner, e.g. "default" */
  meta?: string;
  disabled?: boolean;
}
export interface RadioCardsProps {
  options: RadioCardOption[];
  value?: string;
  onChange?: (value: string) => void;
  /** Fixed column count; default auto-fit ≥180px */
  columns?: number;
  /** aria-label for the group */
  label?: string;
  style?: React.CSSProperties;
}
export function RadioCards(props: RadioCardsProps): JSX.Element;
