import * as React from "react";
/** The typed brand mark: "Verge" bold sans + "ASM" mono ink chip. No logo exists — do not add one. */
export interface WordmarkProps {
  /** sm=13px (footer), md=15px (nav), lg=28px (marketing). @default "md" */
  size?: "sm" | "md" | "lg";
  /** Inverted palette for ink surfaces. @default false */
  onInk?: boolean;
  style?: React.CSSProperties;
}
export declare function Wordmark(props: WordmarkProps): React.ReactElement;
