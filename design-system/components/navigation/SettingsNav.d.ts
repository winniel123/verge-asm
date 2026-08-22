import * as React from "react";
export interface SettingsNavItem { id: string; label: string; icon?: string; }
export interface SettingsNavSection { label: string; items: SettingsNavItem[]; }
export interface SettingsNavProps {
  sections: SettingsNavSection[];
  active?: string;
  onNavigate?: (id: string) => void;
  style?: React.CSSProperties;
}
export function SettingsNav(props: SettingsNavProps): JSX.Element;
