import * as React from "react";
export interface TableColumn {
  key: string;
  label: string;
  width?: number | string;
  align?: "left" | "right" | "center";
  /** Geist Mono cell — REQUIRED for technical values */
  mono?: boolean;
  /** Custom cell renderer */
  render?: (row: any, index: number) => React.ReactNode;
  /** false = no overflow clipping on this cell — required for floating children like DropdownMenu */
  clip?: boolean;
  /** Header click cycles asc \u2192 desc \u2192 off */
  sortable?: boolean;
  /** Custom sort accessor (e.g. severity rank, parsed relative time) */
  sortValue?: (row: any) => any;
}
/** @startingPoint section="Components" subtitle="Dense data table with micro-label heads" viewport="700x260" */
export interface TableProps {
  columns: TableColumn[];
  rows: any[];
  /** Single-row highlight + accent bar (detail-open state) */
  selectedIndex?: number;
  /** Fired on row right-click (default prevented) — pair with a controlled ContextMenu */
  onRowContextMenu?: (row: any, index: number, event: React.MouseEvent) => void;
  onRowClick?: (row: any, index: number) => void;
  /** 7px row padding instead of 10px */
  /** Legacy alias for density="compact" */
  dense?: boolean;
  /** Row density; "compact" tightens padding */
  density?: "comfortable" | "compact";
  /** Window rendering to visible rows (requires maxHeight + fixed rowHeight) */
  virtual?: boolean;
  /** Fixed row height px when virtual. Default 37 */
  rowHeight?: number;
  /** Wrap in card frame (default true) */
  framed?: boolean;
  /** Property to use as React key — required for selection */
  rowKey?: string;
  /** Checkbox column with header select-all; shift-click selects ranges */
  selectable?: boolean;
  selectedKeys?: string[];
  onSelectionChange?: (keys: string[]) => void;
  /** Initial sort, e.g. { key: "sev", dir: "asc" } */
  initialSort?: { key: string; dir: "asc" | "desc" };
  /** Scrollable body with sticky header */
  maxHeight?: number | string;
  style?: React.CSSProperties;
}
export function Table(props: TableProps): JSX.Element;
