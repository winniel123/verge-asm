import * as React from "react";
export interface TagInputProps {
  label?: string;
  hint?: string;
  /** Committed values (mono tags) */
  values: string[];
  onChange?: (values: string[]) => void;
  /** Typeahead suggestions shown while typing */
  suggestions?: string[];
  placeholder?: string;
  /** Per-token validation — return an error string to refuse the commit (e.g. CIDR cap) */
  validate?: (token: string) => string | null | undefined;
  style?: React.CSSProperties;
}
export function TagInput(props: TagInputProps): JSX.Element;
