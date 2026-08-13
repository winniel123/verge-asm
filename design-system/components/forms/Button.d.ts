import * as React from "react";
/** Primary action control. Sharp corners, ink fill.
 * @startingPoint section="Components" subtitle="Primary / secondary / ghost / danger actions" viewport="700x220"
 */
export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  /** Visual weight. primary=solid ink, secondary=ink outline, ghost=borderless, danger=solid red. @default "primary" */
  variant?: "primary" | "secondary" | "ghost" | "danger";
  /** sm=26px (inline in tables), md=32px, lg=40px (marketing). @default "md" */
  size?: "sm" | "md" | "lg";
  /** Optional leading icon node (16px Lucide SVG). */
  icon?: React.ReactNode;
  children?: React.ReactNode;
}
export declare function Button(props: ButtonProps): React.ReactElement;
