import * as React from "react";
export interface IdPButtonProps {
  /** Neutral letter mark + default label per provider. */
  provider?: "okta" | "entra" | "google" | "github" | "saml" | "oidc";
  /** Overrides "Continue with {Provider}". */
  label?: string;
  onClick?: () => void;
  /** Full-width (default true) — auth cards. */
  full?: boolean;
  style?: React.CSSProperties;
}
export declare function IdPButton(props: IdPButtonProps): JSX.Element;
