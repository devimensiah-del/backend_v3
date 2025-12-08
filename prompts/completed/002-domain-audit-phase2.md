<objective>
Audit the remaining 3 domain packages for production MVP readiness: framework, analysis, and wizard.

This is Phase 2 of the comprehensive codebase audit. These domains are more complex and heavily related to the LLM package. Pay special attention to the integration between analysis/framework domains and the llm package.
</objective>

<context>
Read @CLAUDE.md for project conventions.
Read @./docs/audit/PHASE1-SUMMARY.md for Phase 1 findings to maintain consistency.

Domain packages to audit in this phase:
- `domain/framework/` - Dynamic framework configuration (v2+)
- `domain/analysis/` - 11 strategic frameworks execution
- `domain/wizard/` - Human-in-the-loop step-by-step analysis

The analysis domain executes 11 frameworks:
- Layer 1 (Environment): PESTEL, Porter's 7 Forces, TAM-SAM-SOM
- Layer 2 (Positioning): SWOT (with confidence/source), Benchmarking
- Layer 3 (Strategy): Blue Ocean, Growth Hacking, Scenarios
- Layer 4 (Execution): OKRs, BSC, Decision Matrix
- Final: Synthesis (executive summary)

These domains integrate with:
- `llm/` package for AI calls
- `llm/prompts.go` for framework prompts
</context>

<audit_checklist>
For EACH domain package, evaluate and document:

1. **File Organization** (same as Phase 1)
2. **Code Quality** (same as Phase 1)
3. **Error Handling** (same as Phase 1)
4. **Logging** (same as Phase 1)
5. **Testing** (same as Phase 1)

6. **LLM Integration** (CRITICAL for these domains)
   - How does the domain interact with the llm package?
   - Are prompts well-structured and maintainable?
   - Is error handling robust for LLM failures?
   - Are retries and fallbacks properly implemented?
   - Is there proper handling of rate limits?

7. **Business Logic Understanding**
   - Document the framework execution flow
   - How do the 11 frameworks relate to each other?
   - What is the wizard's role in human-in-the-loop?
   - How does framework configuration affect analysis?

8. **README/Documentation**
</audit_checklist>

<output>
Continue the audit documentation:

1. Create audit files:
   - `./docs/audit/006-framework-audit.md`
   - `./docs/audit/007-analysis-audit.md`
   - `./docs/audit/008-wizard-audit.md`

2. Create `./docs/audit/PHASE2-SUMMARY.md` with:
   - Overall findings for these complex domains
   - LLM integration concerns
   - Cross-domain dependencies
   - Critical issues requiring immediate attention
</output>

<constraints>
- Do NOT make any code changes - only audit and document
- Note current test coverage (wizard is at 0%, analysis at 9%)
- Pay special attention to llm package integration
- Flag any hardcoded configurations that should be dynamic
</constraints>

<verification>
Before completing:
- All 3 domains thoroughly reviewed
- LLM integration patterns documented
- Framework execution flow understood and documented
- Phase 2 summary includes integration concerns
</verification>
