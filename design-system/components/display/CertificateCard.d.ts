import * as React from "react";
export interface CertificateCardProps {
  /** Certificate CN / friendly name (mono). */
  name: string;
  /** What the cert is for, e.g. "IdP signing" (default) or "SP signing". */
  role?: string;
  issuer?: string;
  algorithm?: string;
  /** Display date the cert expires. */
  notAfter?: string;
  /** Days until expiry; <=0 renders expired (danger), <=30 expiring (warn), else valid (ok). */
  daysLeft?: number;
  /** e.g. "SHA256:7f:2a:..." — rendered as a CopyValue. */
  fingerprint?: string;
  /** Shows a "Replace certificate" button when provided. */
  onReplace?: () => void;
  style?: React.CSSProperties;
}
export declare function CertificateCard(props: CertificateCardProps): JSX.Element;
