The action control: primary/secondary/ghost/danger, 26/32/40px, ink fill, sharp corners.

Primary action control — solid ink, sharp corners. Use `variant="secondary"` next to a primary, `ghost` for row actions, `danger` for destructive confirms.

```jsx
<Button>Run scan</Button>
<Button variant="secondary" icon={<PlusIcon/>}>Add target</Button>
<Button variant="danger" size="sm">Delete</Button>
```

Labels are imperative sentence case: "Add target", never "Add New Target!". sm=26 fits table rows; lg=40 is for marketing CTAs only.
