# Contributing

Thanks for your interest in contributing.

## Ways to contribute

- Report bugs.
- Suggest enhancements.
- Improve documentation.
- Submit code changes.

## Before you start

- Search existing issues and pull requests before opening a new one.
- For larger changes, open an issue first to discuss the proposal.
- Keep changes focused and small when possible.

## Development workflow

1. Fork the repository.
2. Create a branch for your change.
3. Make your changes.
4. Run relevant tests and checks.
5. Open a pull request with a clear description.

## Pull request guidelines

- Use a clear title and description.
- Link related issues when applicable.
- Include tests or explain why they are not needed.
- Update documentation when behavior changes.
- Be responsive to review feedback.

## Commit guidelines

- Write concise, descriptive commit messages.
- Prefer small, logical commits.

## Code style

- Follow the existing code style and project structure.
- Avoid unrelated refactors in the same pull request.

## Reporting bugs

When reporting a bug, include:

- What happened.
- What you expected to happen.
- Steps to reproduce the issue.
- Environment details such as OS, runtime, and version.

## Suggesting enhancements

When suggesting an enhancement, include:

- The problem the change solves.
- The proposed solution.
- Any alternatives considered.

## Code of conduct

Please be respectful and constructive in all interactions.

## Questions

If you are unsure where to start, open an issue to discuss the change before implementing it.


## GitHub repository settings

After the new workflow passes on a pull request, configure the `main` ruleset
to require pull requests and the **Required CI** check from GitHub Actions.
Block force pushes and branch deletion. Protect version tags matching
`v*.*.*` so only release maintainers can create them. Configure the GitHub
`release` environment to require approval before publication.

## Release qualification

The repository and combined stack are alpha. Follow the local
[release policy](VERSIONING.md) for independent v0 prerelease milestones,
immutable existing tags, beta/v1 gates and compatibility notes. The
[shared qualification plan](https://github.com/hexxla/mosaic/blob/main/docs/RELEASE_QUALIFICATION.md)
defines candidate checks, the 24-hour beta workload and 14-day operational
pilot. Ordinary CI or a commit does not authorize tagging or publication.
