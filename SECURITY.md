# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.2.x   | :white_check_mark: |
| 0.1.x   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability, please report it privately to avoid exposing it to the public.

**Do not open a public issue.**

Use GitHub's private **[Report a vulnerability](https://github.com/hexxla/mosaic/security/advisories/new)** form. If that form is unavailable, contact the maintainers privately through the [Hexxla organisation](https://github.com/hexxla) rather than opening a public issue.

Please include:
- A description of the vulnerability
- Steps to reproduce the vulnerability
- Affected versions
- Any potential impact or exploit

Maintainers will acknowledge and triage reports as promptly as possible, work with reporters to understand and remediate confirmed issues, and coordinate disclosure after a fix is available.

## Security Best Practices

This project follows security best practices including:
- Automated dependency scanning via Dependabot
- Security scanning via gosec
- Secret scanning in CI/CD pipeline
- Regular dependency updates

For repository security expectations, see [`AGENTS.md`](AGENTS.md) and [`.cursor/rules/security.mdc`](.cursor/rules/security.mdc). The `make ci` pipeline runs vulnerability, static-security, and secret checks.
