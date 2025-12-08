<objective>
Execute cleanup and consistency fixes based on all audit findings.

This is Phase 6 of the comprehensive codebase audit. Now that all audits are complete, systematically fix issues found: remove dead code, eliminate deprecated markers, ensure consistency across all layers.
</objective>

<context>
Read all audit files in @./docs/audit/ including:
- PHASE1-SUMMARY.md through PHASE5-SUMMARY.md
- All individual audit files (001-019)

Use these findings to guide cleanup work.
</context>

<cleanup_tasks>

**1. Dead Code Removal**
- Remove any unused functions, types, or variables
- Delete any files that are no longer referenced
- Remove commented-out code blocks
- Clean up unused imports

**2. Deprecated/Backwards Compatible Removal**
- Find all code marked "deprecated" and remove it
- Find all "backwards compatible" shims and remove them
- Update any code that depended on deprecated functionality
- Remove any TODO comments about deprecation

**3. Old Documentation Cleanup**
- Remove outdated inline comments
- Delete old documentation that no longer applies
- Update comments that reference removed functionality

**4. Consistency Enforcement**
Apply consistent patterns across ALL layers:

**Naming Conventions:**
- Function names: consistent casing and verb patterns
- Variable names: consistent abbreviations and styles
- File names: consistent casing and suffixes

**Error Handling:**
- All domains use pkg/errors consistently
- All handlers translate errors the same way
- Error messages follow same format

**Logging:**
- All packages use pkg/logging
- Log levels used consistently
- Context fields follow same patterns

**Testing:**
- Test file naming consistent (*_test.go)
- Test function naming consistent (Test*)
- Mock patterns consistent

**File Organization:**
- Each domain has same file structure
- No files in wrong directories
- Clear separation of concerns

**5. Layer Separation**
Ensure clear separation:
- Models contain only data structures
- Repositories contain only data access
- Services contain only business logic
- Handlers contain only HTTP concerns
</cleanup_tasks>

<output>
As you make changes, track them:

Create `./docs/audit/020-cleanup-log.md` with:
- Each file modified and what was changed
- Code removed (brief description)
- Consistency fixes applied
- Any issues that couldn't be fixed automatically

At the end, create `./docs/audit/PHASE6-SUMMARY.md` with:
- Total files modified
- Categories of changes made
- Remaining issues that need manual review
- Verification steps completed
</output>

<constraints>
- Only remove code confirmed unused in audits
- Run tests after each major change category
- Don't remove code that might be called dynamically
- Preserve any public API compatibility
- If unsure, document rather than delete
</constraints>

<verification>
After each change category:
- Run `go build` to ensure compilation
- Run `go test ./...` to verify tests pass

Before completing:
- Full build passes
- All tests pass
- No "deprecated" or "backwards compatible" markers remain
- Codebase follows consistent patterns
</verification>
