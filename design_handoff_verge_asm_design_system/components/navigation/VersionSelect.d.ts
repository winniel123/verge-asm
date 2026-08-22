import * as React from "react";
export interface VersionOption {
  value: string;
  /** "current" renders accent; anything else muted (e.g. "dev", "eol") */
  tag?: string;
}
export interface VersionSelectProps {
  versions: VersionOption[];
  value?: string;
  onChange?: (value: string) => void;
  style?: React.CSSProperties;
}
export function VersionSelect(props: VersionSelectProps): JSX.Element;
