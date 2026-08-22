import * as React from "react";
export interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string;
  hint?: string;
  /** Error message; also turns the border danger */
  error?: string;
  mono?: boolean;
  /** Grow with content instead of a resize handle */
  autoGrow?: boolean;
  style?: React.CSSProperties;
  inputStyle?: React.CSSProperties;
}
export function Textarea(props: TextareaProps): JSX.Element;
