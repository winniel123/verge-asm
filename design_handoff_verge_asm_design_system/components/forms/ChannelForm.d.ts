import * as React from "react";
export interface Channel {
  url: string;
  /** Message classes this channel receives */
  classes: string[];
  secretSet?: boolean;
}
export interface ChannelFormProps {
  channel?: Channel;
  onSubmit?: (channel: Channel) => void;
  style?: React.CSSProperties;
}
export function ChannelForm(props: ChannelFormProps): JSX.Element;
