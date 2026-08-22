import * as React from "react";
export interface TabItem { id: string; label: string; count?: number; }
export interface TabsProps {
  tabs: TabItem[];
  active?: string;
  onChange?: (id: string) => void;
  style?: React.CSSProperties;
}
export function Tabs(props: TabsProps): JSX.Element;
