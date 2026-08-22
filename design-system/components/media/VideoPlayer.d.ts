import * as React from "react";
export interface VideoPlayerProps {
  src: string;
  poster?: string;
  /** Shown in the control bar and as the region label */
  label?: string;
  /** CSS aspect-ratio, default "16 / 9" */
  aspect?: string;
  style?: React.CSSProperties;
}
export function VideoPlayer(props: VideoPlayerProps): JSX.Element;
