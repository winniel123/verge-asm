\u2318K palette — grouped commands, arrow-key nav, Enter runs. The consumer binds the global shortcut.
```jsx
<CommandPalette open={open} onClose={close} groups={[
  { label: "Screens", items: [{ label: "Signals", icon: "shield-alert", onSelect: () => go("signals") }] },
  { label: "Actions", items: [{ label: "Run scan", icon: "play", hint: "\u2318R", onSelect: run }] },
]} />
```
