import * as React from "react";
export interface ReachLegBadgeProps {
  /** One Reach leg — never an Exposure value, which needs both legs */
  state?: "reached" | "not-reached" | "never-looked" | "stopped-looking";
  /** The vantage class the leg was read from; it decides the tone of `reached` */
  legClass?: "internal" | "internet";
  size?: "md" | "sm";
  style?: React.CSSProperties;
}
export function ReachLegBadge(props: ReachLegBadgeProps): JSX.Element;
