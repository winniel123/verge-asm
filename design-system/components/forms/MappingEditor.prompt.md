# MappingEditor
Two-sided attribute mapping: free mono text (IdP claim) -> closed Select (app field), per-row remove, add button.
Duplicate targets are flagged with a plain sentence ("the last assertion wins; remove one") — never auto-resolved.
Controlled: pass mappings + onChange.
```jsx
<MappingEditor mappings={maps} onChange={setMaps} toOptions={["Email", "Display name", "Org role", "Ignore"]} />
```
