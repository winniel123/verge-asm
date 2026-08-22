import * as React from "react";
export interface CodeInputProps {
  /** Default 6 (grouped 3+3) */
  length?: number;
  /** Controlled digits string */
  value?: string;
  onChange?: (value: string) => void;
  /** Fires once when all digits are filled */
  onComplete?: (value: string) => void;
  label?: string;
  hint?: string;
  /** Error message; also turns the boxes danger */
  error?: string;
  disabled?: boolean;
  autoFocus?: boolean;
  style?: React.CSSProperties;
}
export function CodeInput(props: CodeInputProps): JSX.Element;
