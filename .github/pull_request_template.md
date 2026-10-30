<!--
Thank you for contributing to paykit-go!
Before opening your PR:
- Format your PR title using Conventional Commits (e.g., feat(mpesa): ..., fix(transport): ...; see CONTRIBUTING.md).
- Ensure your branch is rebased on main and `mise run ci` (or `go test -race ./...`) passes cleanly.
- Exported types, functions, and methods should include conventional Godoc comments.
- Fill in the relevant sections below. You are encouraged to adapt or delete optional sections to reduce clutter and keep the review focused.
-->

<!-- Link the issue(s) resolved or related to this PR. E.g., Closes #123, Resolves #456, or Relates to #789 -->

Closes #

---

## Summary

<!--
Provide a concise overview of the circumstances leading to the PR (bug, issue, or feature driver)
and the high-level scope of the change.
-->

---

## Background & Context (Optional)

<!--
Optional: Detail background context, provider specifications, edge-case analysis, or architectural motivations.
Helpful for complex fixes, refactors, or security hardening passes.
Delete this section if not applicable to reduce clutter.
-->

---

## Key Changes

<!--
Break down distinct changes or functional areas, answering three core questions for each:
1. What changed?: The specific capability, component, or behavior modified.
2. How did it change?: Technical mechanics, architecture, and implementation details.
3. Why did it change?: Author intent, design trade-offs, and problem solved.
-->

### 1. [Component / Area Name]

- **What changed?**:
- **How did it change?**:
- **Why did it change?**:

---

## Architecture & Public API Impact (Optional)

<!--
Optional: Detail any public API changes, new capability interfaces, exported symbol additions or modifications,
ADR alignment (e.g., ADR 0001), breaking changes, or backward compatibility considerations.
Delete this section if not applicable to reduce clutter.
-->

---

## Verification & Testing

<!--
Provide concrete evidence of validation. Detail the automated test commands run,
test results, and any manual verification performed (see CONTRIBUTING.md for evidence expectations).

Examples:
- `mise run check`: Passed with zero static analysis or formatting warnings.
- `mise run test`: All unit/integration tests passed.
- `mise run pre-push` / `mise run ci`: Full suite passed cleanly.
-->

---

## Acceptance Evidence

<!--
Attach tangible evidence of verification matching the modified area(s) (see CONTRIBUTING.md):
- Automated Test Output: `go test -v -race ./...` test runner summary or coverage output.
- Provider Sandbox / Simulator Logs: Sanitized request/response exchanges against provider sandbox environments (e.g., Daraja, Airtel, Pesapal).
- Deterministic Fixture Assertions: Test execution demonstrating mock fixture pass rates.
-->

---

## Notes & Follow-ups (Optional)

<!--
Optional: Any ancillary context, follow-up items, provider limitations, or coordination notes.
Delete this section if not applicable to reduce clutter.
-->
