Manage the three exclusion kinds — exact name, subtree (renders *. prefix), address scope.
```jsx
<ExclusionEditor exclusions={xs} onAdd={(kind,v)=>add({kind,value:v})} onRemove={drop} />
```
