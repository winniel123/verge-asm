import React from "react";
import { Button } from "./Button.jsx";
import { DropdownMenu } from "../feedback/DropdownMenu.jsx";
import { Icon } from "../media/Icon.jsx";

export function SplitButton({ children, onClick, items = [], variant = "secondary", size = "md", icon, style }) {
  return (
    <span style={{ display: "inline-flex", ...style }}>
      <Button variant={variant} size={size} icon={icon} onClick={onClick}
        style={{ borderTopRightRadius: 0, borderBottomRightRadius: 0, borderRight: variant === "secondary" ? "none" : undefined }}>
        {children}
      </Button>
      <DropdownMenu align="end" items={items} trigger={
        <Button variant={variant} size={size} aria-label="More formats"
          style={{ borderTopLeftRadius: 0, borderBottomLeftRadius: 0, padding: "0 8px", borderLeft: variant === "secondary" ? "1px solid var(--border-strong)" : "1px solid rgba(255,255,255,0.25)" }}>
          <Icon name="chevron-down" size={13} />
        </Button>
      } />
    </span>
  );
}
