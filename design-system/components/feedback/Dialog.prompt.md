Modal dialog: ink border + 6px hard offset shadow, centered on a 32% ink overlay.

```jsx
<Dialog open={open} title="Delete target?" onClose={close}
  footer={<><Button variant="secondary" onClick={close}>Cancel</Button>
          <Button variant="danger" onClick={del}>Delete</Button></>}>
  Historical findings for acmecorp.io are kept.
</Dialog>
```
