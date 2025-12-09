<objective>
Add a `strategic_narrative` field to the Synthesis prompt in `llm/prompts.go` that organizes the analysis into the 4-Step Strategic Path from the 10XMentorAI methodology.

This enables the frontend to display a clear strategic journey:
- Parte I: Onde Estamos? (PESTEL, Porter, SWOT)
- Parte II: Onde Queremos Ir? (TAM-SAM-SOM, Benchmarking, Blue Ocean)
- Parte III: Como Chegar La? (OKRs, Growth Hacking, BSC)
- Parte IV: O Que Fazer Agora? (Scenarios, Decision Matrix)
</objective>

<context>
The IMENSIAH platform generates strategic analysis reports using 11 frameworks. Currently, the Synthesis prompt produces a flat executive summary. Users have feedback that the old report structure (Santapele PDF) was clearer because it organized frameworks into 4 strategic questions.

The 10XMentorAI methodology (page 56 of their documentation) defines:
1. "Onde Estou?" - Where am I? (Strategic Analysis)
2. "Onde Quero Chegar?" - Where do I want to go? (Strategic Positioning)
3. "Como Chegar La?" - How to get there? (Strategic Planning)
4. "O Que Fazer Nesta Situacao?" - What to do now? (Strategic Decision)

@backend_v3/llm/prompts.go - The Synthesis prompt to modify (around line 786)
</context>

<requirements>
1. Add a new `strategic_narrative` object to the SynthesisPrompt JSON output
2. Each field should summarize the relevant frameworks for that strategic phase
3. Keep character limits tight for PDF layout (max 200 chars per phase)
4. The existing fields MUST remain unchanged (backward compatibility)
5. The new field is ADDITIVE only - nothing is removed or renamed
</requirements>

<implementation>
In `llm/prompts.go`, locate `SynthesisPrompt` (around line 786).

Add the following to the JSON schema section (before the closing `}`):

```go
  "strategic_narrative": {
    "parte_1_onde_estamos": "Sintese da situacao atual baseada em PESTEL, Porter e SWOT (max 200 chars)",
    "parte_2_onde_queremos_ir": "Sintese do posicionamento baseado em TAM-SAM-SOM, Benchmarking e Blue Ocean (max 200 chars)",
    "parte_3_como_chegar_la": "Sintese da execucao baseada em OKRs, Growth Hacking e BSC (max 200 chars)",
    "parte_4_o_que_fazer_agora": "Sintese das acoes imediatas baseada em Cenarios e Matriz de Decisao (max 200 chars)"
  },
```

Also add instruction text BEFORE the JSON schema explaining what the AI should generate:

```
NARRATIVA ESTRATEGICA (NOVO - 4 Perguntas Fundamentais):
Alem do sumario executivo, organize os insights nas 4 perguntas estrategicas:
1. **Parte I - Onde Estamos?**: Sintetize PESTEL + Porter + SWOT em um paragrafo
2. **Parte II - Onde Queremos Ir?**: Sintetize TAM-SAM-SOM + Benchmarking + Blue Ocean
3. **Parte III - Como Chegar La?**: Sintetize OKRs + Growth Hacking + BSC
4. **Parte IV - O Que Fazer Agora?**: Sintetize Cenarios + Matriz de Decisao

Cada sintese deve ter no MAXIMO 200 caracteres e conectar os frameworks da fase.
```
</implementation>

<constraints>
- DO NOT modify any existing JSON fields in the Synthesis output
- DO NOT change the structure of the `consistency_validation` field
- DO NOT add Portuguese accents that could cause encoding issues (use "sintese" not "sintese")
- The new field should be placed BEFORE `consistency_validation` in the JSON
- Use snake_case for JSON keys (parte_1_onde_estamos, not parte1OndeEstamos)
</constraints>

<output>
Modify the file:
- `./llm/prompts.go` - Add strategic_narrative to SynthesisPrompt
</output>

<verification>
Before declaring complete:
1. Run `go build` to ensure no syntax errors
2. Run `go test ./llm/...` if tests exist
3. Verify the JSON schema is valid (no trailing commas, proper quotes)
4. Confirm existing fields are unchanged by comparing before/after
</verification>

<success_criteria>
- SynthesisPrompt compiles without errors
- New `strategic_narrative` field is present in the JSON schema
- Existing fields (executive_summary, key_findings, etc.) are untouched
- Character limits (200 chars) are specified for each phase
- The 4 strategic questions map to the correct frameworks
</success_criteria>
