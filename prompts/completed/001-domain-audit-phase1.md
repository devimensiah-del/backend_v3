<objective>
Audit the first 5 domain packages for production MVP readiness: submission, enrichment, challenge, company, and macroeconomics.

This is Phase 1 of a comprehensive codebase audit. The goal is to ensure each domain is organized, clean, well-documented, well-tested, and follows the codebase's error handling and logging conventions. This is for production MVP - it doesn't need to be perfect, just production-ready.
</objective>

<context>
Read @CLAUDE.md for project conventions and architecture overview.

Domain packages to audit in this phase:
- `domain/submission/` - Entry point for company data capture
- `domain/enrichment/` - Stateless enrichment service (Perplexity-only)
- `domain/challenge/` - Business challenge management
- `domain/company/` - Company management, verification & enriched data
- `domain/macroeconomics/` - SELIC, IPCA, USD/BRL indicators

Each domain should have:
- `model.go` - Domain entities and value objects
- `repository.go` - Database access with sqlx
- `service.go` - Business logic
- `README.md` - Domain-specific documentation (if applicable)
- Appropriate test files
</context>

<audit_checklist>
For EACH domain package, evaluate and document:

1. **File Organization**
   - Are all expected files present (model.go, repository.go, service.go)?
   - Is there unnecessary code duplication?
   - Are files appropriately sized (target: under 500 lines)?

2. **Code Quality**
   - Are naming conventions consistent?
   - Is the code readable and self-documenting?
   - Are there any "deprecated" or "backwards compatible" markers that should be removed?
   - Are there dead code, unused functions, or old inline comments to remove?

3. **Error Handling**
   - Does the domain use the `pkg/errors` package consistently?
   - Are errors wrapped with context?
   - Are appropriate error types used (domain errors vs infrastructure errors)?

4. **Logging**
   - Is `pkg/logging` used consistently?
   - Are log levels appropriate (debug, info, warn, error)?
   - Do logs include relevant context (IDs, operation names)?

5. **Testing**
   - What is the current test coverage?
   - Are there unit tests for critical business logic?
   - Are there integration tests where needed?
   - Are tests well-organized and maintainable?

6. **Business Logic Understanding**
   - Document what this domain does
   - How does it fit into the overall pipeline?
   - What are its dependencies and dependents?

7. **README/Documentation**
   - Is the README accurate and helpful?
   - Does it explain the domain's purpose and key concepts?
</audit_checklist>

<output>
Create an audit documentation folder and save findings:

1. Create `./docs/audit/` directory
2. For each domain, create a findings file:
   - `./docs/audit/001-submission-audit.md`
   - `./docs/audit/002-enrichment-audit.md`
   - `./docs/audit/003-challenge-audit.md`
   - `./docs/audit/004-company-audit.md`
   - `./docs/audit/005-macroeconomics-audit.md`

Each audit file should contain:
- Domain overview (business logic summary)
- File inventory
- Issues found (categorized by severity: critical, important, minor)
- Recommendations
- Action items (specific fixes needed)

Also create:
- `./docs/audit/PHASE1-SUMMARY.md` - Overall findings summary for Phase 1
</output>

<constraints>
- Do NOT make any code changes in this prompt - only audit and document
- Focus on production MVP readiness, not perfection
- Flag anything marked "deprecated" or "backwards compatible" for removal
- Note any inconsistencies with the codebase conventions from CLAUDE.md
</constraints>

<verification>
Before completing:
- Confirm all 5 domains have been thoroughly reviewed
- Each audit file is created with structured findings
- Phase 1 summary captures cross-cutting issues
- All issues are categorized by severity
</verification>
