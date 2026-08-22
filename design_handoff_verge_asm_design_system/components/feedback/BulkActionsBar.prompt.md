Floating selection bar on inverted ink — pairs with Table selectable.
```jsx
<BulkActionsBar count={sel.length} itemLabel="assets" onClear={()=>setSel([])} actions={[
  { label: "Rescan", icon: "play", onClick: rescan },
  { label: "Annotate", icon: "pencil" },
  { label: "Remove", icon: "trash-2", tone: "danger" },
]} />
```
