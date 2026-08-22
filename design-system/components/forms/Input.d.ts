import * as React from "react";
export interface InputProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "size" | "prefix"> {
  label?: string;
  /** Helper line under the field */
  hint?: string;
  /** Error message; also turns the border danger */
  error?: string;
  /** Render the value in Geist Mono — REQUIRED for technical values (domains, CIDRs, ports) */
  mono?: boolean;
  /** Leading node, e.g. <Icon name="search" size={14}/> */
  prefix?: React.ReactNode;
  size?: "sm" | "md";
  style?: React.CSSProperties;
  inputStyle?: React.CSSProperties;
}
export function Input(props: InputProps): JSX.Element;
