The app header: wordmark, section links (active = 2px accent underline), live status, right actions. Sits on a 2px ink rule.

```jsx
<TopNav
  items={[{label:"Dashboard",active:true},{label:"Inventory"},{label:"Findings"},{label:"Graph"},{label:"Reports"}]}
  status={<StatusDot pulse label="scan running" />}
  right={<Button size="sm">+ New scan</Button>} />
```
