import * as React from "react";
export interface Org { id: string; name: string; /** Asset count shown muted */ assets?: number; }
export interface OrgSwitcherProps {
  orgs: Org[];
  active?: string;
  onChange?: (id: string) => void;
  style?: React.CSSProperties;
}
export function OrgSwitcher(props: OrgSwitcherProps): JSX.Element;
