Danger confirm for destructive acts — never fire a descope/remove directly from a menu. typedConfirm for the worst.
```jsx
<ConfirmDialog open={!!doomed} title="Descope seed" message="Removes it and its subjects from scope." typedConfirm={doomed.asset} confirmLabel="Descope seed" onConfirm={run} onClose={clear} />
```
