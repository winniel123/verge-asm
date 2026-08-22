Action menu on a trigger (usually an ellipsis IconButton). Destructive items get tone:"danger", separated by "-".
```jsx
<DropdownMenu trigger={<IconButton icon="ellipsis" label="Actions" />} items={[
  { label: "Annotate — accept risk", icon: "pencil", onSelect: fn },
  { label: "Copy asset", icon: "copy" },
  "-",
  { label: "Descope seed", icon: "trash-2", tone: "danger" },
]} />
```
