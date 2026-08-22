import * as React from "react";
export interface VantageCardProps {
  /** e.g. "eu-west-1" */
  name: string;
  /** internet | internal | unverified — re-verified each batch */
  vantageClass?: "internet" | "internal" | "unverified";
  /** Resolver identity, e.g. "9.9.9.9" */
  resolver?: string;
  availability?: "available" | "degraded" | "unavailable" | "unverified";
  /** e.g. "34ms" */
  latency?: string;
  style?: React.CSSProperties;
}
export function VantageCard(props: VantageCardProps): JSX.Element;
