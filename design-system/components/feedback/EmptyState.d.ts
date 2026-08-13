import * as React from "react";
/** Dashed-border empty state: fact + next action. */
export interface EmptyStateProps {
  /** The fact: "No targets yet." */
  title: React.ReactNode;
  /** The next action, plainly: "Add a domain or CIDR range to start scanning." */
  detail?: React.ReactNode;
  /** Usually one primary Button. */
  action?: React.ReactNode;
  style?: React.CSSProperties;
}
export declare function EmptyState(props: EmptyStateProps): React.ReactElement;
