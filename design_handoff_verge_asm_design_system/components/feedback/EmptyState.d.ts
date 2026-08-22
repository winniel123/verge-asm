import * as React from "react";
export interface EmptyStateProps {
  /** Lucide name. Default "radar" */
  icon?: string;
  /** The fact: "No targets yet." */
  message: string;
  /** The next action, as prose: "Add a domain or CIDR range to start scanning." */
  detail?: string;
  /** Usually one primary or secondary Button */
  action?: React.ReactNode;
  style?: React.CSSProperties;
}
export function EmptyState(props: EmptyStateProps): JSX.Element;
