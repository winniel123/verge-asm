import * as React from "react";
/** The open-source strip: version, license, links, GitHub stars. Every surface carries it. */
export interface FooterProps {
  /** @default "v0.9.2" */
  version?: string;
  /** @default "AGPL-3.0" */
  license?: string;
  links?: Array<{ label: string; href?: string }>;
  /** Star count string, e.g. "4,218". Omits the star block when absent. */
  stars?: string;
  right?: React.ReactNode;
  style?: React.CSSProperties;
}
export declare function Footer(props: FooterProps): React.ReactElement;
