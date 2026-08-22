# ConsentList
Install-time scope grants as a display, not checkboxes — grants are all-or-nothing; partial consent is a lie
the runtime can't keep. Reads get a quiet eye icon; writes get warn tint + a "writes" Tag and should stay rare.
```jsx
<ConsentList grants={[
  { scope: "Read signals", detail: "Message content mirrors the signal drawer." },
  { scope: "Write annotations", detail: "Ticket transitions propose an annotation — an operator confirms.", write: true },
]} />
```
