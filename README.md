# gitui

Lightweight Go + templ UI for GitHub pull requests and Actions.

Copy `.env.example` to `.env`, set a classic PAT with `repo`, `workflow`, and `read:org` scopes, and run:

```sh
chmod 600 .env
go run ./cmd/gitui
```

Authorize the token for organizations using SAML SSO when required. Never commit `.env`.
