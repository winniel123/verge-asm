import * as React from "react";
export interface InlineEditProps {
  value?: string;
  /** Called with the trimmed new value on commit (Enter/blur); empty commits are ignored */
  onChange?: (value: string) => void;
  mono?: boolean;
  /** Shown when value is empty. Default em dash */
  placeholder?: string;
  /** Accessible name, e.g. "seed note" */
  label?: string;
  style?: React.CSSProperties;
}
export function InlineEdit(props: InlineEditProps): JSX.Element;
