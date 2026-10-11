# Security Policy

## Supported Versions

This project is in early development (pre-v1.0). Only the latest release
receives security fixes.

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| < 0.0.x | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability, please **do not** open a public
GitHub issue or pull request. Instead, use GitHub's private vulnerability
reporting:

1. Go to this repository's **Security** tab
2. Click **Report a vulnerability**
   (direct link: `https://github.com/LarsArtmann/go-sse/security/advisories/new`)
3. Include a description of the vulnerability and steps to reproduce

Reports are private by default. Expect acknowledgment within 72 hours and
coordination on responsible disclosure once a patch is ready — including a
CVE request through GitHub's advisory system where appropriate.

## Scope

This is a **library** — it does not serve HTTP requests on its own and never
parses requests from untrusted clients beyond the documented
`Last-Event-ID` header handling. Security concerns include:

- Incorrect wire-format encoding that could confuse client-side parsing
  (e.g., injection of `event:`/`data:`/`id:` field boundaries through
  payload data)
- Validation gaps in `ParseEventID` / `LastEventIDFromRequest` that let
  malformed header values corrupt the reconnection replay protocol
- Concurrency violations in `Stream`, `Broadcaster`, or `Shutdown` that
  could panic a consumer's HTTP server
- Unbounded-resource behavior (memory growth per subscriber) that enables
  denial of service in consumer deployments

Out of scope: vulnerabilities in consumer applications built on this
library, and generic Go runtime issues (report those upstream).

## Security Practices

- The library performs **no logging** of event payloads or headers — nothing
  sensitive leaks through go-sse itself.
- Keep stream endpoints authenticated and authorized like any other HTTP
  route; go-sse is transport-only and adds no auth layer.
- Treat `Last-Event-ID` request values as untrusted input: parse them with
  `sse.ParseEventID` (rejects NUL, newline, and CR per the spec's
  Last-Event-ID value space) before feeding them to a store query.
- Set CORS headers explicitly on SSE endpoints that browsers call
  cross-origin (see the examples for the pattern).
- Broadcast buffers are bounded (`DefaultSubscriberBuffer`); subscribers
  that cannot keep up drop events rather than growing memory unboundedly —
  size consumer channels deliberately.
