import * as React from "react";
export interface ErrorStateProps {
  /** Lucide name. Default "alert-triangle" */
  icon?: string;
  /** The fact: "Batch failed to start." */
  message: string;
  /** What to do about it */
  detail?: string;
  retryLabel?: string;
  onRetry?: () => void;
  style?: React.CSSProperties;
}
export function ErrorState(props: ErrorStateProps): JSX.Element;
