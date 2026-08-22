import * as React from "react";
export interface CodeBlockProps {
  /** The code; the copy control copies its rendered text */
  children?: React.ReactNode;
  /** Micro-label bar, e.g. "shell" */
  title?: string;
  /** Override the copied text (defaults to the rendered textContent) */
  copyText?: string;
  style?: React.CSSProperties;
}
export function CodeBlock(props: CodeBlockProps): JSX.Element;
