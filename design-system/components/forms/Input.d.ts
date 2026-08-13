import * as React from "react";
/** Text field with ink border. Use mono for technical values (hosts, CIDRs). */
export interface InputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "size" | "style"> {
  label?: React.ReactNode;
  /** Muted helper line below. */
  hint?: React.ReactNode;
  /** Error message; turns the border red and replaces hint. */
  error?: React.ReactNode;
  /** IBM Plex Mono input text — for hostnames, IPs, CIDRs. @default false */
  mono?: boolean;
  /** sm=26px, md=32px, lg=40px. @default "md" */
  size?: "sm" | "md" | "lg";
  /** Faint mono prefix inside the field, e.g. "https://". */
  prefix?: React.ReactNode;
  style?: React.CSSProperties;
  inputStyle?: React.CSSProperties;
}
export declare function Input(props: InputProps): React.ReactElement;
