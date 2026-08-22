FormField wraps any control with label/hint/error; FormErrorSummary lists errors at the top of a form and focuses fields on click. Errors name facts, never scold.
```jsx
<FormErrorSummary errors={[{label:"Channel URL",message:"not an https endpoint",fieldId:"url"}]} />
<FormField id="url" label="Channel URL" error={urlErr}><Input id="url" mono value={url} onChange={...} /></FormField>
```
