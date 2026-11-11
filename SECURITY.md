# Security Policy

## Supported Versions

**Maturity: alpha.** The support entries below concern security-report
handling; they do not certify production readiness or a stable API. Qualification
and release promotion follow [VERSIONING.md](VERSIONING.md). Existing tags
remain immutable. Fixes may require upgrading to a newer prerelease; no alpha
long-term-support or automatic backport commitment is made.

| Version | Supported          |
| ------- | ------------------ |
| 0.3.x   | :white_check_mark: |
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

For repository security expectations, see [`AGENTS.md`](AGENTS.md). `task ci` runs vulnerability and static checks. GitHub Actions runs secret scanning in a separate job; the optional local hook scans staged changes.
