import React from "react";
import { Popover } from "../feedback/Popover.jsx";
import { Checkbox } from "../forms/Checkbox.jsx";
import { Button } from "../forms/Button.jsx";
import { Icon } from "../media/Icon.jsx";

export function ColumnPicker({ columns = [], visible = [], onChange, label = "Columns", size = "sm", align = "end" }) {
  const toggle = (key) => {
    if (!onChange) return;
    onChange(columns.map((c) => c.key).filter((k) => (k === key ? !(visible.indexOf(key) !== -1) : visible.indexOf(k) !== -1)));
  };
  return (
    <Popover width={210} align={align} trigger={<Button variant="secondary" size={size} icon={<Icon name="columns-3" size={13} />}>{label}</Button>}>
      <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
        {columns.map((c) => (
          <div key={c.key} style={{ padding: "5px 4px", borderRadius: 8 }}>
            <Checkbox label={c.label} disabled={c.locked} checked={c.locked || visible.indexOf(c.key) !== -1}
              onChange={() => toggle(c.key)} />
          </div>
        ))}
      </div>
    </Popover>
  );
}
