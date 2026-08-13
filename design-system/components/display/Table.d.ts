import * as React from "react";
/** The workhorse: dense data table with mono head rule and row states.
 * @startingPoint section="Components" subtitle="Dense data table with severity rows" viewport="700x280"
 */
export interface TableColumn<Row = any> {
  key: string;
  /** Header text (renders as mono micro-label). */
  label?: React.ReactNode;
  width?: number | string;
  align?: "left" | "right" | "center";
  /** Mono 12px cells — hosts, IPs, timestamps. */
  mono?: boolean;
  /** Muted text color. */
  muted?: boolean;
  nowrap?: boolean;
  /** Custom cell renderer; wins over row[key]. */
  render?: (row: Row) => React.ReactNode;
}
export interface TableProps<Row = any> {
  columns: TableColumn<Row>[];
  rows: Row[];
  /** Field used as React key + selection identity. @default "id" */
  rowKey?: string;
  /** Makes rows hoverable/clickable. */
  onRowClick?: (row: Row) => void;
  /** Row key with accent-soft fill + inset accent bar. */
  selectedKey?: string | number;
  /** 6px row padding instead of 9. @default false */
  dense?: boolean;
  style?: React.CSSProperties;
}
export declare function Table(props: TableProps): React.ReactElement;
