Bottom-right toast queue — concurrent toasts stack instead of last-one-wins; per-toast ttl.
```jsx
<ToastStack toasts={toasts} onDismiss={(id)=>setToasts(ts=>ts.filter(t=>t.id!==id))} />
```
