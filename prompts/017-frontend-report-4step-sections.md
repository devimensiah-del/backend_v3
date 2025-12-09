<objective>
Restructure the public report page in frontend_v2 to organize the 11 frameworks into 4 strategic sections following the 10XMentorAI methodology:

- Parte I: Onde Estamos? (Current Situation Analysis)
- Parte II: Onde Queremos Ir? (Strategic Positioning)
- Parte III: Como Chegar La? (Execution Planning)
- Parte IV: O Que Fazer Agora? (Immediate Actions)

This creates a clear strategic narrative that helps users understand the logical flow of the analysis.
</objective>

<context>
The report page at `/report/[code]/page.tsx` currently displays frameworks in a flat sequence. User feedback indicates the old report structure (Santapele PDF) was clearer because it grouped frameworks into 4 strategic questions.

Framework to Section Mapping:
- **Parte I (Onde Estamos?)**: PESTEL, Porter, SWOT
- **Parte II (Onde Queremos Ir?)**: TAM-SAM-SOM, Benchmarking, Blue Ocean
- **Parte III (Como Chegar La?)**: OKRs, Growth Hacking, BSC
- **Parte IV (O Que Fazer Agora?)**: Scenarios, Decision Matrix, (Synthesis as conclusion)

@frontend_v2/src/app/(public)/report/[code]/page.tsx - Main report page (1727 lines)
@frontend_v2/src/components/report/ - Framework view components
</context>

<design_principles>
CRITICAL: The new divider sections MUST match the existing report's premium visual design.

Study these existing patterns from the report page before implementing:

1. **Typography Scale**:
   - Section labels: `text-[10px] font-bold uppercase tracking-[0.2em]` with gold-500
   - Main headings: `text-3xl lg:text-5xl font-medium tracking-tight`
   - Body text: `text-lg leading-relaxed`

2. **Spacing System**:
   - Section padding: `px-6 lg:px-24 py-24` (generous whitespace)
   - Content max-width: `max-w-6xl mx-auto` for content, `max-w-4xl` for centered text

3. **Color Palette**:
   - Primary dark: `bg-navy-900`, `text-navy-900`
   - Accent: `text-gold-500`, `bg-gold-500`, `border-gold-500`
   - Light backgrounds: `bg-surface-paper`, `bg-gray-50`
   - Muted text: `text-gray-300` (on dark), `text-gray-600` (on light)

4. **Visual Elements**:
   - Gold accent bars: `w-1.5 h-48 bg-gradient-to-b from-gold-500 to-gold-600`
   - Card borders: `border border-gray-200`
   - Icon containers: `w-10 h-10 rounded-full bg-gold-500/10`

5. **Mobile Responsiveness** (CRITICAL):
   - All text sizes use responsive classes: `text-3xl lg:text-5xl`
   - Padding adjusts: `px-6 lg:px-24`
   - Grid layouts stack on mobile: `grid md:grid-cols-2 lg:grid-cols-3`
   - Touch-friendly tap targets: minimum 44px height for interactive elements
</design_principles>

<requirements>
1. Create a `StrategicPartDivider` component that displays part headers
2. Modify the `buildSections()` function to:
   - Reorder frameworks into the 4-part structure
   - Insert divider sections between each part
3. Update the navigation dots to show part names instead of just framework names
4. Ensure Synthesis appears FIRST (as executive summary) and the strategic narrative appears in the dividers
5. Handle cases where `strategic_narrative` field may not exist (backward compatibility)
</requirements>

<implementation>
**Step 1: Create StrategicPartDivider Component**

Add this component inside `page.tsx` (after the existing section components, around line 1500).

The divider should feel like a "chapter opener" - premium, centered, with breathing room.
Match the Hero section's visual gravitas but slightly smaller scale.

```tsx
function StrategicPartDivider({
  partNumber,
  title,
  subtitle,
  narrative,
  icon: Icon,
  variant = 'light'
}: {
  partNumber: string
  title: string
  subtitle: string
  narrative?: string
  icon: React.ComponentType<{ className?: string }>
  variant?: 'light' | 'dark'
}) {
  const isDark = variant === 'dark'

  return (
    <div className={cn(
      // Mobile-first responsive padding and height
      'min-h-[60vh] md:min-h-[50vh] flex flex-col justify-center',
      'px-6 md:px-12 lg:px-24 py-16 md:py-20 lg:py-24',
      isDark ? 'bg-navy-900' : 'bg-surface-paper'
    )}>
      {/* Subtle gold accent bar - matches hero section */}
      {isDark && (
        <div className="absolute left-0 top-1/4 w-1 md:w-1.5 h-32 md:h-48 bg-gradient-to-b from-gold-500 to-gold-600" />
      )}

      <div className="max-w-4xl mx-auto text-center w-full">
        {/* Icon - responsive sizing */}
        <div className={cn(
          'inline-flex items-center justify-center rounded-full mb-6',
          'w-14 h-14 md:w-16 md:h-16 lg:w-20 lg:h-20',
          isDark ? 'bg-gold-500/20' : 'bg-gold-50 border border-gold-200'
        )}>
          <Icon className={cn(
            'w-6 h-6 md:w-8 md:h-8 lg:w-10 lg:h-10',
            isDark ? 'text-gold-500' : 'text-gold-600'
          )} />
        </div>

        {/* Part label - matches existing section number style */}
        <div className={cn(
          'text-[10px] md:text-xs font-bold uppercase tracking-[0.2em] md:tracking-[0.3em] mb-3 md:mb-4',
          isDark ? 'text-gold-500' : 'text-gold-600'
        )}>
          {partNumber}
        </div>

        {/* Main title - responsive typography matching hero */}
        <h2 className={cn(
          'text-2xl md:text-3xl lg:text-5xl font-medium tracking-tight mb-3 md:mb-4',
          isDark ? 'text-white' : 'text-navy-900'
        )}>
          {title}
        </h2>

        {/* Subtitle - frameworks covered */}
        <p className={cn(
          'text-base md:text-lg lg:text-xl mb-6 md:mb-8 max-w-2xl mx-auto',
          isDark ? 'text-gray-300' : 'text-gray-600'
        )}>
          {subtitle}
        </p>

        {/* Narrative quote block - if available from AI */}
        {narrative && (
          <div className={cn(
            'max-w-2xl mx-auto p-4 md:p-6 border-l-4 text-left rounded-r-lg',
            isDark
              ? 'border-gold-500 bg-white/5 backdrop-blur-sm'
              : 'border-gold-500 bg-gold-50/50'
          )}>
            <p className={cn(
              'text-sm md:text-base leading-relaxed',
              isDark ? 'text-gray-200' : 'text-navy-800'
            )}>
              "{narrative}"
            </p>
          </div>
        )}

        {/* Scroll hint for mobile */}
        <div className="mt-8 md:mt-12 flex justify-center">
          <ChevronDown className={cn(
            'w-5 h-5 animate-bounce',
            isDark ? 'text-gray-500' : 'text-gray-400'
          )} />
        </div>
      </div>
    </div>
  )
}
```

**Step 2: Define Strategic Part Configuration**

Create a constant for the 4 parts (add near the top of the file, after imports):

```tsx
const STRATEGIC_PARTS = {
  parte1: {
    number: 'Parte I',
    title: 'Onde Estamos?',
    subtitle: 'Analise da Situacao Atual',
    description: 'PESTEL, Porter 7 Forcas, SWOT',
    icon: Search, // or Compass
    variant: 'light' as const,
    frameworks: ['pestel', 'porter', 'swot'],
  },
  parte2: {
    number: 'Parte II',
    title: 'Onde Queremos Ir?',
    subtitle: 'Posicionamento Estrategico',
    description: 'TAM-SAM-SOM, Benchmarking, Blue Ocean',
    icon: Target,
    variant: 'dark' as const,
    frameworks: ['tam-sam-som', 'benchmarking', 'blue-ocean'],
  },
  parte3: {
    number: 'Parte III',
    title: 'Como Chegar La?',
    subtitle: 'Planejamento Estrategico',
    description: 'OKRs, Growth Hacking, BSC',
    icon: Rocket,
    variant: 'light' as const,
    frameworks: ['okrs', 'growth-hacking', 'bsc'],
  },
  parte4: {
    number: 'Parte IV',
    title: 'O Que Fazer Agora?',
    subtitle: 'Decisao e Acao',
    description: 'Cenarios, Matriz de Decisao',
    icon: Zap,
    variant: 'dark' as const,
    frameworks: ['scenarios', 'decision'],
  },
}
```

**Step 3: Modify buildSections() Function**

Restructure to insert dividers. The order should be:

1. Hero (id: 'hero')
2. Synthesis (id: 'synthesis') - Executive Summary first
3. Parte I Divider (id: 'parte-1')
4. PESTEL, Porter, SWOT
5. Parte II Divider (id: 'parte-2')
6. TAM-SAM-SOM, Benchmarking, Blue Ocean
7. Parte III Divider (id: 'parte-3')
8. OKRs, Growth Hacking, BSC
9. Parte IV Divider (id: 'parte-4')
10. Scenarios, Decision Matrix

**Step 4: Update normalizeSynthesis()**

Around line 1507, add handling for the new `strategic_narrative` field:

```tsx
function normalizeSynthesis(raw: any) {
  return {
    // ... existing fields unchanged ...
    executiveSummary: raw.executiveSummary || raw.executive_summary,
    keyFindings: raw.keyFindings || raw.key_findings,
    strategicPriorities: raw.strategicPriorities || raw.strategic_priorities,
    overallRecommendation: raw.overallRecommendation || raw.overall_recommendation,
    roadmap: raw.roadmap,
    centralChallenge: raw.centralChallenge || raw.central_challenge,
    mainFindings: raw.mainFindings || raw.main_findings,
    importantNotes: raw.importantNotes || raw.important_notes,
    // NEW: Strategic narrative for 4-part structure
    strategicNarrative: raw.strategic_narrative || raw.strategicNarrative || null,
  }
}
```

**Step 5: Add Icons Import**

At the top of the file, ensure these icons are imported from lucide-react:
- `Search` (for Parte I - Onde Estamos) - ADD THIS
- `Target` (already imported - for Parte II)
- `Rocket` (already imported - for Parte III)
- `Zap` (already imported - for Parte IV)
- `ChevronDown` (already imported - for scroll hint)
</implementation>

<mobile_responsive_checklist>
Before finalizing, verify these mobile breakpoints work correctly:

1. **Text sizing**: All headings use `text-2xl md:text-3xl lg:text-5xl` pattern
2. **Padding**: Uses `px-6 md:px-12 lg:px-24` pattern
3. **Icon sizing**: Uses `w-14 h-14 md:w-16 md:h-16 lg:w-20 lg:h-20`
4. **Touch targets**: Buttons and interactive elements are at least 44px
5. **Narrative block**: Readable on small screens with proper padding
6. **No horizontal scroll**: Nothing overflows on 320px width
7. **Navigation dots**: Still accessible on mobile (consider hiding on very small screens)
</mobile_responsive_checklist>

<constraints>
- DO NOT delete or rename any existing components
- DO NOT change the framework section components (SWOTSection, PESTELSection, etc.)
- DO NOT break the navigation dots functionality
- DO NOT remove the blur/premium overlay logic
- DO NOT use any colors outside the existing palette (navy, gold, gray, white)
- Ensure backward compatibility: if strategic_narrative is missing, dividers still render with generic text
- Keep the same visual design language (navy, gold, white color scheme)
- All new styles must be responsive (mobile-first approach)
</constraints>

<output>
Modify the file:
- `./src/app/(public)/report/[code]/page.tsx` - Add StrategicPartDivider, modify buildSections()
</output>

<verification>
Before declaring complete:

1. **Build check**:
   - Run `npm run build` in frontend_v2 to ensure no TypeScript errors
   - Run `npm run lint` to check for any linting issues

2. **Desktop verification** (test at 1440px width):
   - All 11 frameworks still render correctly
   - Part dividers appear between framework groups
   - Navigation dots work and show part names
   - Dividers have the same visual quality as Hero section

3. **Mobile verification** (test at 375px and 320px width):
   - All text is readable without horizontal scrolling
   - Part dividers stack properly
   - Icons are appropriately sized
   - Touch targets are large enough
   - Narrative blocks don't overflow

4. **Tablet verification** (test at 768px width):
   - Layout transitions smoothly between mobile and desktop
   - No awkward spacing or sizing

5. **Backward compatibility**:
   - Test with an existing report that doesn't have strategic_narrative
   - Dividers should show with generic subtitle text
</verification>

<success_criteria>
- Report page compiles without errors
- 4 strategic part dividers are visible between framework groups
- Dividers match the premium visual quality of the Hero section
- Frameworks are reordered into the correct strategic phases
- Navigation dots show both part names and framework names
- Existing reports (without strategic_narrative) render correctly
- **Mobile layout is fully responsive at 320px, 375px, 768px widths**
- **No horizontal scrolling on any screen size**
- **All interactive elements have minimum 44px touch targets**
- No visual regressions in existing framework sections
</success_criteria>
