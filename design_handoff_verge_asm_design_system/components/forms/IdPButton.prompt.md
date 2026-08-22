# IdPButton
Federated sign-in button for auth cards. Ships neutral letter marks, NOT brand logos — swap the mark for the
provider's official SVG in production. Full-width by default; pair with a hairline "or" divider under the
credentials form.
```jsx
<IdPButton provider="okta" onClick={startSso} />
<IdPButton provider="saml" label="Continue with SSO" onClick={startSso} />
```
