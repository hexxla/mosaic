# Versioning and release readiness

**Maturity: alpha.** The HexxlaDB–Mosaic–MCP Ratchet stack is alpha.
No general production sign-off or beta/v1 qualification is recorded.
Existing published tags remain intact historical references.

This document owns Mosaic release policy and MCP tool schemas, configuration, write/retry behavior and database interoperability.

## Release policy

Versions identify releases; maturity is an explicit decision. The minor number
has no ceiling: `v0.9.0` can become `v0.10.0` and later `v0.100.0`. No amount of
iteration automatically promotes a project to v1.

- Published tags are immutable. Preserve every tag and its contents. Never delete, move, reuse or
  reset a tag to make the project look younger; retained tags also support
  reproducible dependency resolution and recovery.
- Keep versions independent across the three repositories. Record the exact
  compatible combination when qualifying the stack.
- Commit normal development without tagging every change. Publish deliberate
  milestones as annotated tags on validated commits.
- During v0, use a minor release for a feature group or intentional compatibility
  break, and patches for compatible fixes. Record breaks and migration steps
  under `CHANGELOG.md` → `[Unreleased]` before release.
- Label new development milestones explicitly, for example `v0.N.0-alpha.1`,
  then `-alpha.2`, `-beta.1` and `-rc.1`. Here `N` is a chosen future minor;
  use a numeric minor in an actual tag. A prerelease sorts below its corresponding
  normal version, so choose a base above existing normal releases.
- Move to beta, release candidate or v1 only through the gates below. A new
  feature or compatibility break during qualification requires a new candidate
  and requalification of affected behavior.
- Use Git tags for pre-v1 distribution. GitHub Release pages and automatic
  binary publication are reserved for v1 unless explicitly reconsidered.
- Follow [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html).
  After v1, compatible additions increment minor, compatible fixes increment
  patch, and public compatibility breaks require a new major version.

## Maturity and evidence

| Stage | Meaning |
| --- | --- |
| Alpha (current) | Working software under active interface and behavior development; suitable for controlled evaluation with recoverable data. General production sign-off has not been given. |
| Beta | Intended initial scope is settled and the beta gates pass for a declared deployment profile. Changes focus on qualification and defect correction. |
| Release candidate | Intended v1 contracts and dependency versions are frozen; all v1 evidence is complete and final candidate checks pass. Only release-blocking corrections are admitted. |
| v1 | Maintainers accept the documented compatibility and support commitments for the qualified profile. Future growth remains possible under SemVer. |

CI is necessary evidence, but does not alone establish operating maturity.
Count an unrun, failed, skipped or unrecorded gate as unmet. Historical evidence
is useful background and must not be presented as a pass on a new candidate.
The [shared qualification plan](https://github.com/hexxla/mosaic/blob/main/docs/RELEASE_QUALIFICATION.md)
owns the stack workload, observation windows and report format. Run it against
explicit source SHAs and dependency versions; an unpublished local workspace
alone does not qualify the distributable stack. Repository promotion may happen
independently, but the combined stack remains alpha until its complete selected
combination passes the stack beta gates.

## Alpha snapshot dependency set

Mosaic `v0.4.0-alpha.2` uses the following independently versioned upstream
snapshots. These pins describe a reproducible alpha combination; they do not
certify beta/v1 gates or the operational pilot.

| Dependency | Published tag | Source commit |
| --- | --- | --- |
| HexxlaDB | `v0.8.0-alpha.1` | `96070acab89eebca49aab51fe598d572f42041f0` |
| MCP Ratchet | `v0.4.0-alpha.1` | `43d4ffb110d4ea0180d0d7104943a61a44bbdaa9` |

The canonical module path is `github.com/hexxla/mosaic`. The preserved
`v0.4.0-alpha.1` snapshot declared the former, unavailable
`github.com/sploitzberg/mosaic` path and is superseded for Go installation.
Use `v0.4.0-alpha.2` or later; existing tag contents are not rewritten.

Candidate validation must use normal module resolution with `GOWORK=off`.
Local replacement/workspace results remain additional integration evidence.

## Beta gates

Current status: **unmet / alpha**. The 43-tool advanced profile is functional
and tested, but feature coverage does not establish operational qualification.

| Gate | Pass condition |
| --- | --- |
| Scope | Settle the initial core/advanced tool inventories, schemas, capture/deletion policy, retrieval limits and local deployment contract. Declare platform, dataset, model/digest and resource/performance targets. |
| Candidate checks | `task ci` and `task build-all` pass. Review schemas and runtime `tools/list` for both profiles, tool safety classifications and coverage of every enabled mutation by bundled and custom Ratchet policies. |
| Integrated behavior | Pin the intended published/publishable HexxlaDB and Ratchet revisions, then exercise real MCP sessions: isolation, successful/denied/failed calls, cancellation, admission/draining, restart, durable/uncertain outcomes, idempotent retries, changefeed continuity, model changes and guarded metadata/facet writes. |
| Bounded transfer and failure | Exercise sparse/large data, byte/scan limits, oversized records, cursor tampering/expiry/foreign sessions, snapshot expiry and interrupted batch/export flows. Require explicit partial results/errors and no silent loss or duplicate logical writes. |
| Qualification | Complete the shared 24-hour beta workload with at least 10,000 completed operations, both profiles and enabled optional services. Verify a backup/restore and restart/upgrade rehearsal against an oracle; meet predeclared targets. |
| Review | No unresolved data-integrity, workflow enforcement, session-isolation, reachable security or uncontrolled-resource blocker in the supported loopback profile. Document remaining limitations and ownership. |

## V1 gates

Current status: **unmet**. All beta gates must remain satisfied.

| Gate | Pass condition |
| --- | --- |
| Contract commitment | Maintainers accept the supported MCP schemas/results, policy/configuration formats, error/retry behavior, budget semantics and database/model compatibility rules. Explain migrations for all pre-v1 changes. |
| Operational pilot | Complete the shared 14-day named-owner pilot with a real agent workflow, monitoring and at least one verified upgrade and full recovery-set restore/rollback drill. Maintainer-owned use is sufficient; external adoption is not required. |
| Deployment and support | Validate installation from the distributable source, document the supported local/loopback threat boundary, pinned dependency combination, model identity, backup/restore instructions, resource envelope, security reporting and maintenance/upgrade policy. |
| Final candidate | Freeze intended v1 contracts/dependencies; rerun CI, vulnerability and supported-platform checks on the selected sources. Review the qualification report and sign off the exact stack combination. |

Physical database formats follow HexxlaDB's own version/upgrade contract.
Mosaic's regional transfer is cell-only; qualification uses a native complete
backup for recovery. A remote, multi-tenant or distributed deployment is outside
the initial v1 claim. See the [capability contract](docs/mosaic/HEXXLA_API_SURFACE_COVERAGE.md)
and [shared qualification plan](docs/RELEASE_QUALIFICATION.md).
