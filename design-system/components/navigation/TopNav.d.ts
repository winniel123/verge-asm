import * as React from "react";
/** App chrome: wordmark, section nav, status + right slots; 2px ink bottom rule.
 * @startingPoint section="Components" subtitle="App header with wordmark and section nav" viewport="900x60"
 */
export interface TopNavItem { label: string; href?: string; active?: boolean; }
export interface TopNavProps {
  items?: TopNavItem[];
  onSelect?: (item: TopNavItem, index: number) => void;
  /** Live status slot, e.g. <StatusDot pulse label="scan running"/>. */
  status?: React.ReactNode;
  /** Far-right slot (buttons, org switcher). */
  right?: React.ReactNode;
  style?: React.CSSProperties;
}
export declare function TopNav(props: TopNavProps): React.ReactElement;
