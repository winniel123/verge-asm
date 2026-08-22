import * as React from "react";
export interface GraphNode {
  id: string;
  /** Mono label next to the node */
  label: string;
  type: "domain" | "subdomain" | "ip" | "service";
  /** Position in view coordinates */
  x: number;
  y: number;
  /** Severity halo for nodes with open signals */
  sev?: "critical" | "high" | "medium" | "low" | "info";
}
export interface GraphEdge { from: string; to: string; }
export interface GraphProps {
  nodes: GraphNode[];
  edges: GraphEdge[];
  /** Rendered height px. Default 560 */
  height?: number;
  /** Coordinate space. Default 1200\u00d7640 */
  viewWidth?: number;
  viewHeight?: number;
  selectedId?: string;
  onNodeSelect?: (node: GraphNode) => void;
  /** Zoom/reset control cluster (default true) */
  controls?: boolean;
  /** Overview minimap with the current viewport box (default false) */
  minimap?: boolean;
  style?: React.CSSProperties;
}
export function Graph(props: GraphProps): JSX.Element;
export interface GraphLegendProps { style?: React.CSSProperties; }
export function GraphLegend(props: GraphLegendProps): JSX.Element;
