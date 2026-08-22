import * as React from "react";
export interface BreadcrumbItem {
  label: string;
  href?: string;
  onClick?: (e: React.MouseEvent) => void;
}
export interface BreadcrumbProps {
  /** Last item = current page */
  items: BreadcrumbItem[];
  style?: React.CSSProperties;
}
export function Breadcrumb(props: BreadcrumbProps): JSX.Element;
