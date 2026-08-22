# CertificateCard
IdP/SP signing certificate: name, role, issuer, algorithm, expiry, fingerprint (CopyValue).
Expiry is operational state on Badge tones — ok / warn (<=30d) / danger (expired) — never the severity ramp.
Expired adds a plain-language consequence line; onReplace shows the replace button.
```jsx
<CertificateCard name="idp-signing-2026" issuer="CN=Okta, O=acmecorp" algorithm="RSA-SHA256"
  notAfter="2026-09-14" daysLeft={23} fingerprint="SHA256:7f:2a:91:c4..." onReplace={rotate} />
```
