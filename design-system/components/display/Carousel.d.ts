import * as React from "react";
export interface CarouselProps {
  /** Slides — each child owns its own content and height */
  children?: React.ReactNode;
  ariaLabel?: string;
  /** Advance on a timer; pauses on hover/focus. Default false */
  autoAdvance?: boolean;
  /** Ms between advances, default 6000 */
  interval?: number;
  /** Wrap past the ends. Default false */
  loop?: boolean;
  showArrows?: boolean;
  showDots?: boolean;
  style?: React.CSSProperties;
}
export function Carousel(props: CarouselProps): JSX.Element;
