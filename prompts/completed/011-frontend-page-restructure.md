# Prompt 011: Frontend Page Restructure

## Objective
Restructure the frontend_v2 pages to be challenge-centric. Users navigate: Dashboard → Companies → Challenges → Wizard/Analysis.

## Working Directory
```bash
cd ../frontend_v2
```

## Key Principle
- **Users see Companies + Challenges** (not submissions)
- **Admin sees Submissions** (audit trail only)
- **Wizard is accessed FROM a Challenge**

## Tasks

### 1. Update Route Structure

#### Target Structure:
```
src/app/
├── (public)/
│   ├── page.tsx                    # Landing (keep as-is)
│   ├── submit/page.tsx             # Public submission (keep as-is)
│   ├── obrigado/page.tsx           # Thank you (keep as-is)
│   ├── login/page.tsx              # Login (keep as-is)
│   ├── signup/page.tsx             # Signup (keep as-is)
│   ├── forgot-password/page.tsx    # Password recovery (keep as-is)
│   ├── reset-password/page.tsx     # Password reset (keep as-is)
│   ├── report/[code]/page.tsx      # Public report via access code (keep as-is)
│   ├── privacy/page.tsx            # Privacy (keep as-is)
│   └── terms/page.tsx              # Terms (keep as-is)
│
├── (dashboard)/
│   ├── layout.tsx                  # User dashboard layout
│   ├── dashboard/
│   │   ├── page.tsx                # Dashboard home = Companies list
│   │   ├── companies/
│   │   │   └── [id]/page.tsx       # Company detail with challenges
│   │   └── settings/page.tsx       # User settings (keep as-is)
│   │
│   ├── wizard/
│   │   └── [analysisId]/page.tsx   # Wizard interface (move from submissions)
│   │
│   └── analysis/
│       └── [id]/page.tsx           # View completed analysis
│
├── (admin)/
│   ├── layout.tsx                  # Admin layout
│   └── admin/
│       ├── dashboard/page.tsx      # Admin home (metrics)
│       ├── submissions/page.tsx    # All submissions (audit trail)
│       ├── companies/
│       │   ├── page.tsx            # All companies
│       │   └── [id]/page.tsx       # Admin company view
│       ├── analyses/
│       │   ├── page.tsx            # All analyses
│       │   └── [id]/page.tsx       # Admin analysis (visibility controls)
│       ├── frameworks/page.tsx     # Framework config
│       └── macro/page.tsx          # Macro indicators
│
└── wizard/
    └── [analysisId]/page.tsx       # Standalone wizard (legacy, redirect to dashboard version)
```

### 2. DELETE These Pages (User shouldn't see submissions)

```bash
# Remove user-facing submission pages
rm -rf src/app/(dashboard)/dashboard/submissions/
rm -rf src/app/(dashboard)/submissions/
```

### 3. Update Dashboard Home (`src/app/(dashboard)/dashboard/page.tsx`)

Replace with Companies list:

```tsx
'use client'

import { useCompanies } from '@/lib/hooks/use-companies'
import { CompanyCard } from '@/components/features/company/company-card'
import { LoadingSpinner } from '@/components/shared/loading-spinner'
import { EmptyState } from '@/components/shared/empty-state'
import { Building2 } from 'lucide-react'

export default function DashboardPage() {
  const { data: companies, isLoading } = useCompanies()

  if (isLoading) return <LoadingSpinner />

  if (!companies?.length) {
    return (
      <EmptyState
        icon={Building2}
        title="Nenhuma empresa ainda"
        description="Suas empresas aparecerão aqui após você enviar um diagnóstico."
        action={{
          label: "Iniciar Diagnóstico",
          href: "/submit"
        }}
      />
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-heading font-bold">Minhas Empresas</h1>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {companies.map((company) => (
          <CompanyCard key={company.id} company={company} />
        ))}
      </div>
    </div>
  )
}
```

### 4. Create Company Detail Page (`src/app/(dashboard)/dashboard/companies/[id]/page.tsx`)

Challenge-centric view:

```tsx
'use client'

import { useParams, useRouter } from 'next/navigation'
import { useCompany } from '@/lib/hooks/use-companies'
import { useChallengesByCompany } from '@/lib/hooks/use-challenges'
import { CompanyHeader } from '@/components/features/company/company-header'
import { ChallengeCard } from '@/components/dashboard/ChallengeCard'
import { NewChallengeModal } from '@/components/dashboard/NewChallengeModal'
import { LoadingSpinner } from '@/components/shared/loading-spinner'
import { Button } from '@/components/ui/button'
import { Plus } from 'lucide-react'
import { useState } from 'react'

export default function CompanyPage() {
  const { id } = useParams<{ id: string }>()
  const { data: company, isLoading: companyLoading } = useCompany(id)
  const { data: challenges, isLoading: challengesLoading } = useChallengesByCompany(id)
  const [showNewChallenge, setShowNewChallenge] = useState(false)

  if (companyLoading || challengesLoading) return <LoadingSpinner />
  if (!company) return <div>Empresa não encontrada</div>

  // Group challenges by status
  const completedChallenges = challenges?.filter(c => c.latest_analysis?.status === 'completed') || []
  const inProgressChallenges = challenges?.filter(c =>
    c.latest_analysis && c.latest_analysis.status !== 'completed'
  ) || []
  const pendingChallenges = challenges?.filter(c => !c.latest_analysis) || []

  return (
    <div className="space-y-8">
      <CompanyHeader company={company} />

      {/* Challenges Section */}
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-heading font-semibold">Desafios</h2>
          <Button onClick={() => setShowNewChallenge(true)}>
            <Plus className="mr-2 h-4 w-4" />
            Novo Desafio
          </Button>
        </div>

        {/* In Progress */}
        {inProgressChallenges.length > 0 && (
          <div className="space-y-3">
            <h3 className="text-sm font-medium text-text-secondary uppercase tracking-widest">
              Em Andamento
            </h3>
            <div className="grid gap-4 md:grid-cols-2">
              {inProgressChallenges.map(challenge => (
                <ChallengeCard key={challenge.id} challenge={challenge} />
              ))}
            </div>
          </div>
        )}

        {/* Pending (no analysis) */}
        {pendingChallenges.length > 0 && (
          <div className="space-y-3">
            <h3 className="text-sm font-medium text-text-secondary uppercase tracking-widest">
              Aguardando Análise
            </h3>
            <div className="grid gap-4 md:grid-cols-2">
              {pendingChallenges.map(challenge => (
                <ChallengeCard key={challenge.id} challenge={challenge} />
              ))}
            </div>
          </div>
        )}

        {/* Completed */}
        {completedChallenges.length > 0 && (
          <div className="space-y-3">
            <h3 className="text-sm font-medium text-text-secondary uppercase tracking-widest">
              Concluídos
            </h3>
            <div className="grid gap-4 md:grid-cols-2">
              {completedChallenges.map(challenge => (
                <ChallengeCard key={challenge.id} challenge={challenge} />
              ))}
            </div>
          </div>
        )}

        {/* Empty State */}
        {!challenges?.length && (
          <div className="text-center py-12 text-text-secondary">
            <p>Nenhum desafio criado ainda.</p>
            <Button
              variant="outline"
              className="mt-4"
              onClick={() => setShowNewChallenge(true)}
            >
              Criar Primeiro Desafio
            </Button>
          </div>
        )}
      </div>

      <NewChallengeModal
        companyId={id}
        open={showNewChallenge}
        onOpenChange={setShowNewChallenge}
      />
    </div>
  )
}
```

### 5. Update ChallengeCard (`src/components/dashboard/ChallengeCard.tsx`)

```tsx
'use client'

import { Challenge } from '@/lib/types/domain'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { useRouter } from 'next/navigation'
import { useStartWizard } from '@/lib/hooks/use-wizard'
import {
  TrendingUp, RefreshCw, ArrowRight, Play, Eye,
  Pencil, Trash2, MoreVertical
} from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

const categoryIcons: Record<string, typeof TrendingUp> = {
  growth: TrendingUp,
  transform: RefreshCw,
  transition: ArrowRight,
  compete: Target,
  funding: DollarSign,
}

const categoryLabels: Record<string, string> = {
  growth: 'Crescimento',
  transform: 'Transformação',
  transition: 'Transição',
  compete: 'Competição',
  funding: 'Captação',
}

interface ChallengeCardProps {
  challenge: Challenge
  onEdit?: () => void
  onDelete?: () => void
}

export function ChallengeCard({ challenge, onEdit, onDelete }: ChallengeCardProps) {
  const router = useRouter()
  const startWizard = useStartWizard()

  const Icon = categoryIcons[challenge.challenge_category] || TrendingUp
  const analysis = challenge.latest_analysis

  const getStatus = () => {
    if (!analysis) return { label: 'Sem análise', variant: 'outline' as const }
    if (analysis.status === 'completed') return { label: 'Concluído', variant: 'default' as const }
    if (analysis.status === 'failed') return { label: 'Erro', variant: 'destructive' as const }
    // In progress - show step count
    const currentStep = analysis.current_step || 1
    return { label: `Passo ${currentStep}/12`, variant: 'secondary' as const }
  }

  const status = getStatus()

  const handleAction = () => {
    if (!analysis) {
      // Start new analysis
      startWizard.mutate(
        { company_id: challenge.company_id, challenge_id: challenge.id },
        { onSuccess: (data) => router.push(`/dashboard/wizard/${data.analysis_id}`) }
      )
    } else if (analysis.status === 'completed') {
      // View analysis
      router.push(`/dashboard/analysis/${analysis.id}`)
    } else {
      // Continue wizard
      router.push(`/dashboard/wizard/${analysis.id}`)
    }
  }

  const getActionLabel = () => {
    if (!analysis) return 'Iniciar Análise'
    if (analysis.status === 'completed') return 'Ver Resultado'
    return 'Continuar'
  }

  return (
    <Card className="relative group hover:border-gold-500/50 transition-colors">
      <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-2">
        <div className="flex items-center gap-2">
          <div className="p-2 bg-navy-900/5 rounded">
            <Icon className="h-4 w-4 text-navy-700" />
          </div>
          <div>
            <p className="text-xs text-text-secondary uppercase tracking-wider">
              {categoryLabels[challenge.challenge_category]}
            </p>
            <CardTitle className="text-base mt-0.5">
              {challenge.challenge_type.replace(/_/g, ' ').replace(/^\w/, c => c.toUpperCase())}
            </CardTitle>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Badge variant={status.variant}>{status.label}</Badge>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={onEdit}>
                <Pencil className="mr-2 h-4 w-4" />
                Editar
              </DropdownMenuItem>
              <DropdownMenuItem onClick={onDelete} className="text-red-600">
                <Trash2 className="mr-2 h-4 w-4" />
                Excluir
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>

      <CardContent>
        <p className="text-sm text-text-secondary line-clamp-2 mb-4">
          {challenge.business_challenge}
        </p>

        <Button
          onClick={handleAction}
          className="w-full"
          variant={analysis?.status === 'completed' ? 'outline' : 'default'}
          disabled={startWizard.isPending}
        >
          {startWizard.isPending ? (
            'Iniciando...'
          ) : (
            <>
              {!analysis && <Play className="mr-2 h-4 w-4" />}
              {analysis?.status === 'completed' && <Eye className="mr-2 h-4 w-4" />}
              {getActionLabel()}
            </>
          )}
        </Button>
      </CardContent>
    </Card>
  )
}
```

### 6. Move Wizard to New Location

Move `src/app/(dashboard)/submissions/[id]/wizard/page.tsx` to `src/app/(dashboard)/wizard/[analysisId]/page.tsx`

Update the wizard page to use `analysisId` param instead of submission ID.

### 7. Update Navigation Sidebar

Update `src/components/shared/user-sidebar.tsx`:

```tsx
const navItems = [
  { href: '/dashboard', label: 'Empresas', icon: Building2 },
  { href: '/dashboard/settings', label: 'Configurações', icon: Settings },
]
// Remove submissions link
```

## Verification
- [ ] Dashboard shows companies list
- [ ] Company page shows challenges organized by status
- [ ] Challenge card has correct actions (Start/Continue/View)
- [ ] New challenge modal works
- [ ] Wizard accessible from challenge
- [ ] No references to user-facing submission pages
- [ ] Admin still has submission access

## Files Modified/Created
- `src/app/(dashboard)/dashboard/page.tsx` - Updated to companies list
- `src/app/(dashboard)/dashboard/companies/[id]/page.tsx` - New challenge-centric view
- `src/app/(dashboard)/wizard/[analysisId]/page.tsx` - Moved wizard
- `src/components/dashboard/ChallengeCard.tsx` - Enhanced
- `src/components/shared/user-sidebar.tsx` - Updated nav

## Files Deleted
- `src/app/(dashboard)/dashboard/submissions/` - Entire directory
- `src/app/(dashboard)/submissions/` - Entire directory
