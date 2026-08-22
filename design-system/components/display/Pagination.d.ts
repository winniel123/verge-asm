import * as React from "react";
export interface PaginationProps {
  /** 1-based current page */
  page: number;
  pageCount: number;
  onChange?: (page: number) => void;
  /** With totalItems, renders the mono "1\u201325 of 1,284" range */
  pageSize?: number;
  totalItems?: number;
  style?: React.CSSProperties;
}
export function Pagination(props: PaginationProps): JSX.Element;
