import * as React from "react";
export interface SecretInputProps {
  label?: string;
  /** A secret exists (its value is never available to render) */
  isSet?: boolean;
  onSave?: (secret: string) => void;
  hint?: string;
  style?: React.CSSProperties;
}
export function SecretInput(props: SecretInputProps): JSX.Element;
