import * as React from "react";
export interface Proposal {
  id: string;
  /** The proposed name or address scope */
  value: string;
  /** e.g. "name", "range" */
  kind: string;
  /** Where the registry saw it, e.g. "registrar match" */
  source: string;
}
export interface ProposalReviewProps {
  proposals: Proposal[];
  /** Per-row — there is deliberately no bulk confirm */
  onConfirm?: (p: Proposal) => void;
  /** Bulk decline of the checked ids */
  onDecline?: (ids: string[]) => void;
  style?: React.CSSProperties;
}
export function ProposalReview(props: ProposalReviewProps): JSX.Element;
