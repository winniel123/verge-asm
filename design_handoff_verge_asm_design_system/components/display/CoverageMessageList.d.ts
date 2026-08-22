import * as React from "react";
export interface CoverageMessage {
  id?: string;
  /** gap = expected observation missing; stale / not-evaluable / silent = currency states */
  kind: "gap" | "stale" | "not-evaluable" | "silent";
  /** GapBadge label override, e.g. "no address" */
  badge?: string;
  /** StalenessBadge bound, e.g. "9d" */
  bound?: string;
  subject: string;
  text: string;
  when?: string;
  iso?: string;
}
export interface CoverageMessageListProps { messages: CoverageMessage[]; style?: React.CSSProperties; }
export function CoverageMessageList(props: CoverageMessageListProps): JSX.Element;
