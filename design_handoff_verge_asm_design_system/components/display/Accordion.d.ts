import * as React from "react";
export interface AccordionItem {
  id: string;
  title: string;
  content: React.ReactNode;
}
export interface AccordionProps {
  items: AccordionItem[];
  /** Allow several open at once (default single-open) */
  multiple?: boolean;
  /** Ids open on mount */
  defaultOpen?: string[];
  style?: React.CSSProperties;
}
export function Accordion(props: AccordionProps): JSX.Element;
