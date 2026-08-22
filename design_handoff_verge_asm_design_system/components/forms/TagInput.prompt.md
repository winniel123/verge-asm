Multi-value filter input — Enter/comma commits, Backspace removes, typeahead from suggestions. Values are mono Tags (sev:high, tag:edge).
```jsx
<TagInput label="Filters" values={f} onChange={setF} suggestions={["sev:critical","sev:high","tag:edge"]} placeholder="sev:high, tag:edge" />
```
