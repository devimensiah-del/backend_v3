# Prompt 012: Wizard Human-in-the-Loop Refinement

## Objective
Refine the wizard to be a true human-in-the-loop experience with proper progress persistence, step navigation, and error recovery.

## Working Directory
```bash
cd ../frontend_v2
```

## Key Requirements
- User CAN go back to previous steps (view only, not edit)
- User CANNOT skip steps
- Progress persists (resume later)
- Error recovery: retry failed step
- Context input for refinement (not direct text editing)
- Progress indicator: "Passo 1 de 12", "Passo 2 de 12", etc.

## Tasks

### 1. Update Wizard State Types (`src/lib/types/domain.ts`)

```typescript
export interface WizardStep {
  framework_key: string
  framework_name: string
  status: 'pending' | 'generating' | 'awaiting_review' | 'approved' | 'failed'
  output?: string
  error?: string
  approved_at?: string
  refinement_count: number
}

export interface WizardState {
  analysis_id: string
  challenge_id: string
  company_id: string
  current_step: number // 1-12
  total_steps: number  // 12
  steps: WizardStep[]
  status: 'in_progress' | 'completed' | 'failed'
  can_go_back: boolean
  can_skip: boolean // Always false
  started_at: string
  completed_at?: string
}
```

### 2. Create Wizard Page (`src/app/(dashboard)/wizard/[analysisId]/page.tsx`)

```tsx
'use client'

import { useParams, useRouter } from 'next/navigation'
import { useWizardState, useGenerateStep, useApproveStep, useRefineStep } from '@/lib/hooks/use-wizard'
import { WizardProgress } from '@/components/features/wizard/wizard-progress'
import { WizardStep } from '@/components/features/wizard/wizard-step'
import { WizardCompletion } from '@/components/features/wizard/wizard-completion'
import { LoadingSpinner } from '@/components/shared/loading-spinner'
import { AlertCircle, ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import Link from 'next/link'

export default function WizardPage() {
  const { analysisId } = useParams<{ analysisId: string }>()
  const router = useRouter()
  const { data: wizardState, isLoading, error, refetch } = useWizardState(analysisId)

  const generateStep = useGenerateStep(analysisId)
  const approveStep = useApproveStep(analysisId)
  const refineStep = useRefineStep(analysisId)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <LoadingSpinner />
      </div>
    )
  }

  if (error) {
    return (
      <div className="max-w-2xl mx-auto py-12">
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            Erro ao carregar wizard. Por favor, tente novamente.
          </AlertDescription>
        </Alert>
        <Button onClick={() => refetch()} className="mt-4">
          Tentar Novamente
        </Button>
      </div>
    )
  }

  if (!wizardState) return null

  // Completed state
  if (wizardState.status === 'completed') {
    return <WizardCompletion wizardState={wizardState} />
  }

  const currentStep = wizardState.steps[wizardState.current_step - 1]
  const isGenerating = generateStep.isPending
  const isApproving = approveStep.isPending
  const isRefining = refineStep.isPending
  const isBusy = isGenerating || isApproving || isRefining

  return (
    <div className="min-h-screen bg-surface-paper">
      {/* Header */}
      <header className="sticky top-0 z-10 bg-white border-b border-line">
        <div className="max-w-4xl mx-auto px-4 py-4 flex items-center justify-between">
          <Link href={`/dashboard/companies/${wizardState.company_id}`}>
            <Button variant="ghost" size="sm">
              <ArrowLeft className="mr-2 h-4 w-4" />
              Voltar
            </Button>
          </Link>

          <WizardProgress
            currentStep={wizardState.current_step}
            totalSteps={wizardState.total_steps}
            steps={wizardState.steps}
          />

          <div className="w-24" /> {/* Spacer for centering */}
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-4xl mx-auto px-4 py-8">
        <WizardStep
          step={currentStep}
          stepNumber={wizardState.current_step}
          totalSteps={wizardState.total_steps}
          isGenerating={isGenerating}
          isApproving={isApproving}
          isRefining={isRefining}
          onGenerate={(context) => generateStep.mutate({ context })}
          onApprove={() => approveStep.mutate()}
          onRefine={(context) => refineStep.mutate({ context })}
          onRetry={() => generateStep.mutate({})}
          previousSteps={wizardState.steps.slice(0, wizardState.current_step - 1)}
        />
      </main>
    </div>
  )
}
```

### 3. Create WizardProgress Component (`src/components/features/wizard/wizard-progress.tsx`)

```tsx
'use client'

import { WizardStep } from '@/lib/types/domain'
import { cn } from '@/lib/utils'
import { Check, Circle, Loader2, AlertCircle } from 'lucide-react'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface WizardProgressProps {
  currentStep: number
  totalSteps: number
  steps: WizardStep[]
}

export function WizardProgress({ currentStep, totalSteps, steps }: WizardProgressProps) {
  return (
    <div className="flex flex-col items-center">
      <p className="text-sm font-medium text-text-primary mb-2">
        Passo {currentStep} de {totalSteps}
      </p>

      <div className="flex items-center gap-1">
        <TooltipProvider>
          {steps.map((step, index) => {
            const stepNum = index + 1
            const isActive = stepNum === currentStep
            const isCompleted = step.status === 'approved'
            const isFailed = step.status === 'failed'
            const isGenerating = step.status === 'generating'

            return (
              <Tooltip key={step.framework_key}>
                <TooltipTrigger asChild>
                  <div
                    className={cn(
                      'w-3 h-3 rounded-full transition-all',
                      isCompleted && 'bg-green-500',
                      isFailed && 'bg-red-500',
                      isGenerating && 'bg-gold-500 animate-pulse',
                      isActive && !isCompleted && !isFailed && !isGenerating && 'bg-navy-700',
                      !isActive && !isCompleted && !isFailed && !isGenerating && 'bg-gray-200'
                    )}
                  />
                </TooltipTrigger>
                <TooltipContent>
                  <p>{step.framework_name}</p>
                  <p className="text-xs text-gray-400">
                    {isCompleted && 'Aprovado'}
                    {isFailed && 'Erro'}
                    {isGenerating && 'Gerando...'}
                    {isActive && !isCompleted && !isFailed && !isGenerating && 'Atual'}
                    {!isActive && !isCompleted && !isFailed && !isGenerating && 'Pendente'}
                  </p>
                </TooltipContent>
              </Tooltip>
            )
          })}
        </TooltipProvider>
      </div>
    </div>
  )
}
```

### 4. Create WizardStep Component (`src/components/features/wizard/wizard-step.tsx`)

```tsx
'use client'

import { WizardStep as WizardStepType } from '@/lib/types/domain'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Loader2, Check, RefreshCw, AlertCircle,
  ChevronDown, ChevronUp, Sparkles
} from 'lucide-react'
import { useState } from 'react'
import { cn } from '@/lib/utils'
import ReactMarkdown from 'react-markdown'

interface WizardStepProps {
  step: WizardStepType
  stepNumber: number
  totalSteps: number
  isGenerating: boolean
  isApproving: boolean
  isRefining: boolean
  onGenerate: (context?: string) => void
  onApprove: () => void
  onRefine: (context: string) => void
  onRetry: () => void
  previousSteps: WizardStepType[]
}

export function WizardStep({
  step,
  stepNumber,
  totalSteps,
  isGenerating,
  isApproving,
  isRefining,
  onGenerate,
  onApprove,
  onRefine,
  onRetry,
  previousSteps,
}: WizardStepProps) {
  const [refinementContext, setRefinementContext] = useState('')
  const [showPrevious, setShowPrevious] = useState(false)

  const isBusy = isGenerating || isApproving || isRefining
  const hasOutput = step.status === 'awaiting_review' && step.output
  const hasFailed = step.status === 'failed'

  return (
    <div className="space-y-6">
      {/* Previous Steps (Collapsible) */}
      {previousSteps.length > 0 && (
        <div>
          <Button
            variant="ghost"
            onClick={() => setShowPrevious(!showPrevious)}
            className="text-text-secondary"
          >
            {showPrevious ? <ChevronUp className="mr-2 h-4 w-4" /> : <ChevronDown className="mr-2 h-4 w-4" />}
            Ver passos anteriores ({previousSteps.length})
          </Button>

          {showPrevious && (
            <div className="mt-4 space-y-4">
              {previousSteps.map((prevStep, index) => (
                <Card key={prevStep.framework_key} className="bg-gray-50 border-gray-200">
                  <CardHeader className="py-3">
                    <CardTitle className="text-sm flex items-center gap-2">
                      <Check className="h-4 w-4 text-green-500" />
                      {index + 1}. {prevStep.framework_name}
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="py-3">
                    <div className="prose prose-sm max-w-none text-text-secondary">
                      <ReactMarkdown>{prevStep.output || ''}</ReactMarkdown>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Current Step */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-3">
            <span className="flex items-center justify-center w-8 h-8 rounded-full bg-navy-900 text-white text-sm font-bold">
              {stepNumber}
            </span>
            {step.framework_name}
          </CardTitle>
        </CardHeader>

        <CardContent className="space-y-6">
          {/* Pending State - Generate Button */}
          {step.status === 'pending' && (
            <div className="text-center py-8">
              <Sparkles className="mx-auto h-12 w-12 text-gold-500 mb-4" />
              <p className="text-text-secondary mb-6">
                Clique para gerar a análise do framework {step.framework_name}.
              </p>
              <Button onClick={() => onGenerate()} disabled={isBusy} size="lg">
                {isGenerating ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Gerando...
                  </>
                ) : (
                  'Gerar Análise'
                )}
              </Button>
            </div>
          )}

          {/* Generating State */}
          {step.status === 'generating' && (
            <div className="text-center py-12">
              <Loader2 className="mx-auto h-12 w-12 text-gold-500 animate-spin mb-4" />
              <p className="text-text-secondary">
                Gerando análise com IA...
              </p>
              <p className="text-xs text-text-tertiary mt-2">
                Isso pode levar alguns segundos.
              </p>
            </div>
          )}

          {/* Failed State */}
          {hasFailed && (
            <div className="space-y-4">
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  {step.error || 'Ocorreu um erro ao gerar a análise.'}
                </AlertDescription>
              </Alert>
              <Button onClick={onRetry} disabled={isBusy}>
                <RefreshCw className="mr-2 h-4 w-4" />
                Tentar Novamente
              </Button>
            </div>
          )}

          {/* Output Review State */}
          {hasOutput && (
            <div className="space-y-6">
              {/* Generated Output */}
              <div className="prose prose-sm max-w-none bg-white border border-line rounded-lg p-6">
                <ReactMarkdown>{step.output}</ReactMarkdown>
              </div>

              {/* Refinement Section */}
              <div className="space-y-3">
                <Label htmlFor="refinement" className="label-editorial">
                  Fornecer Contexto Adicional (Opcional)
                </Label>
                <Textarea
                  id="refinement"
                  value={refinementContext}
                  onChange={(e) => setRefinementContext(e.target.value)}
                  placeholder="Se desejar refinar a análise, descreva o que gostaria de ajustar ou adicionar contexto específico da sua empresa..."
                  className="min-h-[100px] input-editorial"
                  disabled={isBusy}
                />
                <p className="text-xs text-text-tertiary">
                  Refinamentos feitos: {step.refinement_count}
                </p>
              </div>

              {/* Action Buttons */}
              <div className="flex flex-col sm:flex-row gap-3">
                <Button
                  onClick={() => {
                    if (refinementContext.trim()) {
                      onRefine(refinementContext)
                      setRefinementContext('')
                    }
                  }}
                  variant="outline"
                  disabled={isBusy || !refinementContext.trim()}
                  className="flex-1"
                >
                  {isRefining ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Refinando...
                    </>
                  ) : (
                    <>
                      <RefreshCw className="mr-2 h-4 w-4" />
                      Refinar
                    </>
                  )}
                </Button>

                <Button
                  onClick={onApprove}
                  disabled={isBusy}
                  className="flex-1 btn-architect"
                >
                  {isApproving ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Aprovando...
                    </>
                  ) : (
                    <>
                      <Check className="mr-2 h-4 w-4" />
                      {stepNumber === totalSteps ? 'Finalizar' : 'Aprovar e Continuar'}
                    </>
                  )}
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
```

### 5. Create WizardCompletion Component (`src/components/features/wizard/wizard-completion.tsx`)

```tsx
'use client'

import { WizardState } from '@/lib/types/domain'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Check, Eye, Share2, Download } from 'lucide-react'
import Link from 'next/link'
import confetti from 'canvas-confetti'
import { useEffect } from 'react'

interface WizardCompletionProps {
  wizardState: WizardState
}

export function WizardCompletion({ wizardState }: WizardCompletionProps) {
  useEffect(() => {
    // Celebration confetti
    confetti({
      particleCount: 100,
      spread: 70,
      origin: { y: 0.6 }
    })
  }, [])

  return (
    <div className="min-h-screen bg-surface-paper flex items-center justify-center p-4">
      <Card className="max-w-lg w-full text-center">
        <CardContent className="py-12 space-y-6">
          <div className="mx-auto w-16 h-16 rounded-full bg-green-100 flex items-center justify-center">
            <Check className="h-8 w-8 text-green-600" />
          </div>

          <div>
            <h1 className="text-2xl font-heading font-bold text-text-primary mb-2">
              Análise Concluída!
            </h1>
            <p className="text-text-secondary">
              Todos os 12 frameworks foram analisados e aprovados com sucesso.
            </p>
          </div>

          <div className="flex flex-col sm:flex-row gap-3 pt-4">
            <Link href={`/dashboard/analysis/${wizardState.analysis_id}`} className="flex-1">
              <Button className="w-full btn-architect">
                <Eye className="mr-2 h-4 w-4" />
                Ver Análise Completa
              </Button>
            </Link>

            <Link href={`/dashboard/companies/${wizardState.company_id}`} className="flex-1">
              <Button variant="outline" className="w-full">
                Voltar à Empresa
              </Button>
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
```

### 6. Update Wizard Hooks (`src/lib/hooks/use-wizard.ts`)

```typescript
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { wizardApi } from '../api'
import { toast } from 'sonner'

export function useWizardState(analysisId: string) {
  return useQuery({
    queryKey: ['wizard', analysisId],
    queryFn: () => wizardApi.getState(analysisId),
    enabled: !!analysisId,
    refetchInterval: (data) => {
      // Poll while generating
      if (data?.steps.some(s => s.status === 'generating')) {
        return 2000 // 2 seconds
      }
      return false
    },
  })
}

export function useStartWizard() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: wizardApi.start,
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['challenges'] })
      toast.success('Análise iniciada')
    },
    onError: () => toast.error('Erro ao iniciar análise'),
  })
}

export function useGenerateStep(analysisId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input?: { context?: string }) => wizardApi.generate(analysisId, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wizard', analysisId] })
    },
    onError: () => toast.error('Erro ao gerar análise'),
  })
}

export function useApproveStep(analysisId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => wizardApi.approve(analysisId),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['wizard', analysisId] })
      if (data.status === 'completed') {
        toast.success('Análise concluída!')
      } else {
        toast.success('Passo aprovado')
      }
    },
    onError: () => toast.error('Erro ao aprovar passo'),
  })
}

export function useRefineStep(analysisId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: { context: string }) => wizardApi.refine(analysisId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['wizard', analysisId] })
      toast.success('Refinamento solicitado')
    },
    onError: () => toast.error('Erro ao refinar'),
  })
}
```

## Verification
- [ ] Wizard loads with correct step
- [ ] Progress indicator shows "Passo X de 12"
- [ ] Generate button works for pending steps
- [ ] Output displays with markdown rendering
- [ ] Refinement input works
- [ ] Approve advances to next step
- [ ] Failed state shows retry button
- [ ] Completion screen shows on finish
- [ ] Previous steps viewable (collapsed)
- [ ] Resume works after leaving page

## Files Created/Modified
- `src/app/(dashboard)/wizard/[analysisId]/page.tsx`
- `src/components/features/wizard/wizard-progress.tsx`
- `src/components/features/wizard/wizard-step.tsx`
- `src/components/features/wizard/wizard-completion.tsx`
- `src/lib/hooks/use-wizard.ts`
- `src/lib/types/domain.ts`

## Dependencies to Add
```bash
npm install canvas-confetti react-markdown
npm install -D @types/canvas-confetti
```
