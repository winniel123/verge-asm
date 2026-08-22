import * as React from "react";
export interface LogoProps {
  /** Glyph size in px. Default 20; works alone at >=16 */
  size?: number;
  /** Include the "verge ASM" wordmark. Default true */
  withWordmark?: boolean;
  /** Override the derived wordmark px size */
  wordmarkSize?: number;
  /** Light-on-dark treatment (inverted ink panels, dark chrome) */
  inverted?: boolean;
  /** Wrap the glyph in a rounded azure tile (favicon/app-tile) */
  tile?: boolean;
  style?: React.CSSProperties;
}
export function Logo(props: LogoProps): JSX.Element;
