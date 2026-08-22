import * as React from "react";
export interface TreeNode {
  id: string;
  label: string;
  /** Muted count on the right */
  count?: number;
  /** Severity dot */
  sev?: "critical" | "high" | "medium" | "low" | "info";
  children?: TreeNode[];
}
export interface TreeViewProps {
  nodes: TreeNode[];
  defaultOpen?: string[];
  onSelect?: (node: TreeNode) => void;
  selectedId?: string;
  style?: React.CSSProperties;
}
export function TreeView(props: TreeViewProps): JSX.Element;
