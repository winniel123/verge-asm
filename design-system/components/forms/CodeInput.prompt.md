TOTP/MFA verify field: segmented digits, auto-advance, paste-distributes. Codes are never masked; enrollment secrets belong in SecretInput.
```jsx
<CodeInput label="Verification code" value={code} onChange={setCode} onComplete={verify} hint="From your authenticator app" />
```
