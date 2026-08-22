Named filter sets above a data view (chip row; active = accent tint). Show Save view only when filters drift from the active set.
```jsx
<SavedViews views={[{id:"all",label:"All assets",count:1284},{id:"ranges",label:"Ranges"}]} activeId={view} onSelect={setView} dirty={dirty} onSave={save} />
```
