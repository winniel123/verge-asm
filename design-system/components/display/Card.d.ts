import * as React from "react";
/** Flat white panel; the container for every dashboard module.
 * @startingPoint section="Components" subtitle="Panel with eyebrow, title, action slot" viewport="700x260"
 */
export interface CardProps {
  /** Card heading, 14px semibold sentence case. */
  title?: React.ReactNode;
  /** Mono micro-label above the title. */
  eyebrow?: React.ReactNode;
  /** Right-aligned header slot (link, Button size="sm"). */
  action?: React.ReactNode;
  /** Ink outline instead of hairline — for the one panel that matters most. @default false */
  emphasized?: boolean;
  /** Set false when the body is a Table (flush edges). @default true */
  pad?: boolean;
  style?: React.CSSProperties;
  bodyStyle?: React.CSSProperties;
  children?: React.ReactNode;
}
export declare function Card(props: CardProps): React.ReactElement;
