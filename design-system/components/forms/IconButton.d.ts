import * as React from "react";
/** Square icon-only button. */
export interface IconButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  /** ghost=borderless (default), secondary=ink outline. @default "ghost" */
  variant?: "ghost" | "secondary";
  /** Square side: sm=26, md=32, lg=40. @default "md" */
  size?: "sm" | "md" | "lg";
  /** Accessible name; also the title tooltip. Required. */
  label: string;
  /** The icon (16px Lucide SVG or unicode glyph). */
  children?: React.ReactNode;
}
export declare function IconButton(props: IconButtonProps): React.ReactElement;
