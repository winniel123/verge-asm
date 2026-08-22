import * as React from "react";
export interface TopNavItem { id: string; label: string; count?: number; }
export interface TopNavProps {
  /** Default: dashboard / inventory / signals / graph / reports */
  items?: TopNavItem[];
  active?: string;
  onNavigate?: (id: string) => void;
  /** Org context chip (MSPs run many orgs) */
  orgName?: string;
  /** Pass orgs to render the interactive OrgSwitcher instead of the static chip */
  orgs?: Array<{ id: string; name: string; assets?: number }>;
  activeOrg?: string;
  onOrgChange?: (id: string) => void;
  /** Open-source identity in the chrome. Default "v0.9.2" */
  version?: string;
  /** Signed-in user (Avatar initials). Default "Ola P\u00e9rez" */
  user?: string;
  /** Shows the pulsing scan status */
  scanRunning?: boolean;
  /** Renders the theme toggle */
  onToggleTheme?: () => void;
  /** Renders the search trigger (⌘K palette) */
  onOpenPalette?: () => void;
  /** Renders the inbox icon + popover (MessageList shape); unread messages show a dot */
  messages?: Array<{ id: string; cls: string; text: string; time: string; unread?: boolean }>;
  /** "All messages" / message click destination */
  onOpenAllMessages?: () => void;
  dark?: boolean;
  style?: React.CSSProperties;
}
export function TopNav(props: TopNavProps): JSX.Element;
