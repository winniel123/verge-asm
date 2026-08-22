Data table — micro-label heads, hairline row separators, hover sunken, selected = fill + accent bar. Density is a feature: use dense on big lists. Columns holding DropdownMenu/Popover need clip:false. sortable (+sortValue) per column; maxHeight scrolls the body under a sticky header.
```jsx
<Table columns={[{key:"asset",label:"Asset",mono:true},{key:"sev",label:"Severity",render:(r)=><SeverityBadge level={r.sev} size="sm"/>}]} rows={rows} onRowClick={open} />
```
