import * as React from "react";
export interface ButtonProps {
  /** primary = the one filled action on screen. Default "primary" */
  variant?: "primary" | "secondary" | "ghost" | "danger";
  /** sm 30 / md 36 / lg 44 (marketing). Default "md" */
  size?: "sm" | "md" | "lg";
  /** Optional leading icon node, e.g. <Icon name="play" size={14}/> */
  icon?: React.ReactNode;
  disabled?: boolean;
  type?: "button" | "submit";
  onClick?: () => void;
  style?: React.CSSProperties;
  children?: React.ReactNode;
}
export function Button(props: ButtonProps): JSX.Element;
