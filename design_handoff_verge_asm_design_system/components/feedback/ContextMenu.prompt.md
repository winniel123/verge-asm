Right-click menu, same item grammar as DropdownMenu. Wrapper mode for elements; controlled mode pairs with Table onRowContextMenu. Destructive items still confirm via ConfirmDialog.
```jsx
<ContextMenu open={!!ctx} x={ctx?.x} y={ctx?.y} onClose={()=>setCtx(null)} items={[{label:"Annotate",icon:"pencil",onSelect:...},"-",{label:"Descope seed",icon:"trash-2",tone:"danger",onSelect:...}]} />
```
