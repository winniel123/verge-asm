Modal dialog — r24 panel, shadow-lg, vg-pop-in entrance, scrim without blur.
```jsx
<Dialog open title="Add seed" description="Domain or CIDR range." onClose={close}
  footer={<><Button variant="ghost" onClick={close}>Cancel</Button><Button>Add seed</Button></>}>
  <Input label="Seed" mono placeholder="acmecorp.io" />
</Dialog>
```
