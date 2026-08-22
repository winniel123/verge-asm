import * as React from "react";
import { DropdownMenuItem } from "../feedback/DropdownMenu";
export interface SplitButtonProps {
  /** The primary action label */
  children?: React.ReactNode;
  onClick?: () => void;
  /** Menu of alternates (DropdownMenu item shape, "-" separators ok) */
  items: Array<DropdownMenuItem | "-">;
  variant?: "primary" | "secondary";
  size?: "sm" | "md" | "lg";
  icon?: React.ReactNode;
  style?: React.CSSProperties;
}
export function SplitButton(props: SplitButtonProps): JSX.Element;
