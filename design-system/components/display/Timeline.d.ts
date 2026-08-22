import * as React from "react";
export interface TimelineEvent {
  title: string;
  /** Second line; set mono for technical detail */
  detail?: string;
  /** Terse relative time ("4m") or ISO */
  time: string;
  tone?: "neutral" | "accent" | "ok" | "warn" | "danger";
  mono?: boolean;
  /** Overrides the tone dot color (e.g. var(--drift-gain-solid)) */
  dotColor?: string;
  /** Extra block under the detail (e.g. an inline DiffView) */
  content?: React.ReactNode;
}
export interface TimelineGroup {
  id: string;
  /** Mono header, e.g. a batch timestamp */
  label: string;
  meta?: string;
  events: TimelineEvent[];
  defaultCollapsed?: boolean;
}
export interface TimelineProps {
  /** Newest first by convention */
  events?: TimelineEvent[];
  /** Collapsible per-batch sections; overrides events */
  groups?: TimelineGroup[];
  style?: React.CSSProperties;
}
export function Timeline(props: TimelineProps): JSX.Element;
