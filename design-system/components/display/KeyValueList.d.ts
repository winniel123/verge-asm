import * as React from "react";
export interface KeyValueItem {
  /** Micro-label key, e.g. "First seen" */
  k: string;
  v: React.ReactNode;
  /** Default true — technical values are mono; set false for prose values */
  mono?: boolean;
  /** Grid columns to span */
  span?: number;
}
export interface KeyValueListProps {
  items: KeyValueItem[];
  /** Default 2 */
  columns?: number;
  /** Sunken rounded background (default true) */
  sunken?: boolean;
  style?: React.CSSProperties;
}
export function KeyValueList(props: KeyValueListProps): JSX.Element;
