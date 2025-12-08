# Prompt 014: Admin Features & Generate-All

## Objective
Implement admin-specific features: generate-all bypass for wizard, visibility toggles, and submission audit view.

## Working Directory
```bash
cd ../frontend_v2
```

## Key Admin Features
1. **Submissions List** - Audit trail (admin only)
2. **Generate-All** - Bypass wizard, generate all 12 frameworks at once
3. **Visibility Toggles** - Control `is_public` and `is_visible_to_user`
4. **Retry Enrichment** - Re-trigger company enrichment

## Tasks

### 1. Update Admin Submissions Page (`src/app/(admin)/admin/submissions/page.tsx`)

```tsx
'use client'

import { useAdminSubmissions } from '@/lib/hooks/use-admin'
import { DataTable } from '@/components/ui/data-table'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { LoadingSpinner } from '@/components/shared/loading-spinner'
import { formatDistanceToNow } from 'date-fns'
import { ptBR } from 'date-fns/locale'
import Link from 'next/link'
import { Eye, Building2 } from 'lucide-react'

const columns = [
  {
    accessorKey: 'company_name',
    header: 'Empresa',
    cell: ({ row }) => (
      <div>
        <p className="font-medium">{row.original.company_name}</p>
        <p className="text-xs text-text-secondary">{row.original.contact_email}</p>
      </div>
    ),
  },
  {
    accessorKey: 'created_at',
    header: 'Enviado',
    cell: ({ row }) => (
      <span className="text-sm text-text-secondary">
        {formatDistanceToNow(new Date(row.original.created_at), {
          addSuffix: true,
          locale: ptBR
        })}
      </span>
    ),
  },
  {
    accessorKey: 'company_id',
    header: 'Empresa Criada',
    cell: ({ row }) => (
      row.original.company_id ? (
        <Link href={`/admin/companies/${row.original.company_id}`}>
          <Button variant="ghost" size="sm">
            <Building2 className="mr-2 h-4 w-4" />
            Ver Empresa
          </Button>
        </Link>
      ) : (
        <Badge variant="outline">Pendente</Badge>
      )
    ),
  },
  {
    accessorKey: 'user_id',
    header: 'Usuário',
    cell: ({ row }) => (
      row.original.user_id ? (
        <Badge variant="default">Vinculado</Badge>
      ) : (
        <Badge variant="outline">Anônimo</Badge>
      )
    ),
  },
  {
    id: 'actions',
    cell: ({ row }) => (
      <Link href={`/admin/submissions/${row.original.id}`}>
        <Button variant="ghost" size="sm">
          <Eye className="h-4 w-4" />
        </Button>
      </Link>
    ),
  },
]

export default function AdminSubmissionsPage() {
  const { data: submissions, isLoading } = useAdminSubmissions()

  if (isLoading) return <LoadingSpinner />

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-heading font-bold">Submissões</h1>
        <p className="text-text-secondary">
          Audit trail de todas as submissões do sistema
        </p>
      </div>

      <DataTable
        columns={columns}
        data={submissions || []}
        searchKey="company_name"
        searchPlaceholder="Buscar por empresa..."
      />
    </div>
  )
}
```

### 2. Update Admin Analysis Page with Generate-All (`src/app/(admin)/admin/analyses/[id]/page.tsx`)

```tsx
'use client'

import { useParams } from 'next/navigation'
import { useAdminAnalysis, useGenerateAll, useToggleVisibility } from '@/lib/hooks/use-admin'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { LoadingSpinner } from '@/components/shared/loading-spinner'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Zap, Eye, EyeOff, Globe, Lock,
  Loader2, AlertCircle, Check, RefreshCw
} from 'lucide-react'
import { FrameworkViewer } from '@/components/features/analysis/framework-viewer'

export default function AdminAnalysisPage() {
  const { id } = useParams<{ id: string }>()
  const { data: analysis, isLoading, refetch } = useAdminAnalysis(id)
  const generateAll = useGenerateAll()
  const toggleVisibility = useToggleVisibility()

  if (isLoading) return <LoadingSpinner />
  if (!analysis) return <div>Análise não encontrada</div>

  const handleGenerateAll = () => {
    generateAll.mutate(id, {
      onSuccess: () => refetch()
    })
  }

  const handleToggle = (field: 'is_public' | 'is_visible_to_user', value: boolean) => {
    toggleVisibility.mutate(
      { analysisId: id, field, value },
      { onSuccess: () => refetch() }
    )
  }

  const isCompleted = analysis.status === 'completed'
  const isProcessing = analysis.status === 'processing' || generateAll.isPending

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-heading font-bold">Análise</h1>
          <p className="text-text-secondary">ID: {analysis.id}</p>
        </div>

        <Badge
          variant={
            analysis.status === 'completed' ? 'default' :
            analysis.status === 'failed' ? 'destructive' :
            'secondary'
          }
        >
          {analysis.status}
        </Badge>
      </div>

      {/* Admin Controls */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Zap className="h-5 w-5 text-gold-500" />
            Controles Admin
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          {/* Generate All Button */}
          {!isCompleted && (
            <div className="space-y-2">
              <p className="text-sm text-text-secondary">
                Gerar todos os 12 frameworks de uma vez (bypass wizard)
              </p>
              <Button
                onClick={handleGenerateAll}
                disabled={isProcessing}
                className="btn-architect"
              >
                {isProcessing ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Gerando...
                  </>
                ) : (
                  <>
                    <Zap className="mr-2 h-4 w-4" />
                    Gerar Todos
                  </>
                )}
              </Button>

              {generateAll.isPending && (
                <Alert>
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription>
                    Processando 12 frameworks. Isso pode levar alguns minutos...
                  </AlertDescription>
                </Alert>
              )}
            </div>
          )}

          {/* Visibility Controls */}
          {isCompleted && (
            <div className="space-y-4 pt-4 border-t border-line">
              <h4 className="font-medium">Visibilidade</h4>

              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  {analysis.is_visible_to_user ? (
                    <Eye className="h-4 w-4 text-green-500" />
                  ) : (
                    <EyeOff className="h-4 w-4 text-gray-400" />
                  )}
                  <Label htmlFor="visible_user">Visível para Usuário</Label>
                </div>
                <Switch
                  id="visible_user"
                  checked={analysis.is_visible_to_user}
                  onCheckedChange={(checked) => handleToggle('is_visible_to_user', checked)}
                  disabled={toggleVisibility.isPending}
                />
              </div>

              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  {analysis.is_public ? (
                    <Globe className="h-4 w-4 text-green-500" />
                  ) : (
                    <Lock className="h-4 w-4 text-gray-400" />
                  )}
                  <Label htmlFor="public">Relatório Público</Label>
                </div>
                <Switch
                  id="public"
                  checked={analysis.is_public}
                  onCheckedChange={(checked) => handleToggle('is_public', checked)}
                  disabled={toggleVisibility.isPending}
                />
              </div>

              {analysis.access_code && (
                <div className="pt-2">
                  <p className="text-xs text-text-secondary">Código de Acesso:</p>
                  <code className="text-sm bg-gray-100 px-2 py-1 rounded">
                    {analysis.access_code}
                  </code>
                </div>
              )}
            </div>
          )}

          {/* Progress for in-progress */}
          {!isCompleted && analysis.current_step && (
            <div className="pt-4 border-t border-line">
              <p className="text-sm text-text-secondary">
                Progresso: Passo {analysis.current_step}/12
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Framework Results */}
      {isCompleted && analysis.framework_results && (
        <FrameworkViewer results={analysis.framework_results} />
      )}
    </div>
  )
}
```

### 3. Add Admin Hooks (`src/lib/hooks/use-admin.ts`)

```typescript
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { adminApi } from '../api'
import { toast } from 'sonner'

export function useAdminSubmissions() {
  return useQuery({
    queryKey: ['admin', 'submissions'],
    queryFn: () => adminApi.listSubmissions(),
  })
}

export function useAdminCompanies() {
  return useQuery({
    queryKey: ['admin', 'companies'],
    queryFn: () => adminApi.listCompanies(),
  })
}

export function useAdminAnalysis(id: string) {
  return useQuery({
    queryKey: ['admin', 'analysis', id],
    queryFn: () => adminApi.getAnalysis(id),
    enabled: !!id,
  })
}

export function useGenerateAll() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (analysisId: string) => adminApi.generateAll(analysisId),
    onSuccess: (_, analysisId) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'analysis', analysisId] })
      toast.success('Geração iniciada. Acompanhe o progresso.')
    },
    onError: () => toast.error('Erro ao iniciar geração'),
  })
}

export function useToggleVisibility() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      analysisId,
      field,
      value
    }: {
      analysisId: string
      field: 'is_public' | 'is_visible_to_user'
      value: boolean
    }) => adminApi.updateVisibility(analysisId, { [field]: value }),
    onSuccess: (_, { analysisId }) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'analysis', analysisId] })
      toast.success('Visibilidade atualizada')
    },
    onError: () => toast.error('Erro ao atualizar visibilidade'),
  })
}

export function useRetryEnrichment() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (companyId: string) => adminApi.retryEnrichment(companyId),
    onSuccess: (_, companyId) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'companies'] })
      queryClient.invalidateQueries({ queryKey: ['company', companyId] })
      toast.success('Enriquecimento reiniciado')
    },
    onError: () => toast.error('Erro ao reiniciar enriquecimento'),
  })
}
```

### 4. Update Admin API (`src/lib/api.ts`)

Add to `adminApi`:

```typescript
export const adminApi = {
  // Existing...
  listSubmissions: () => api.get<Submission[]>('/admin/submissions'),
  listCompanies: () => api.get<Company[]>('/admin/companies'),
  getAnalysis: (id: string) => api.get<Analysis>(`/admin/analyses/${id}`),

  // Add these:
  generateAll: (analysisId: string) =>
    api.post(`/admin/analyses/${analysisId}/wizard/generate-all`),

  updateVisibility: (analysisId: string, data: {
    is_public?: boolean
    is_visible_to_user?: boolean
  }) => api.patch(`/admin/analyses/${analysisId}/visibility`, data),

  retryEnrichment: (companyId: string) =>
    api.post(`/admin/companies/${companyId}/retry-enrichment`),
}
```

### 5. Update Admin Company Page (`src/app/(admin)/admin/companies/[id]/page.tsx`)

Add retry enrichment button:

```tsx
'use client'

import { useParams } from 'next/navigation'
import { useAdminCompany, useRetryEnrichment } from '@/lib/hooks/use-admin'
import { useChallengesByCompany } from '@/lib/hooks/use-challenges'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { LoadingSpinner } from '@/components/shared/loading-spinner'
import { ChallengeCard } from '@/components/dashboard/ChallengeCard'
import { RefreshCw, Loader2, Building2, Globe, MapPin } from 'lucide-react'
import Link from 'next/link'

export default function AdminCompanyPage() {
  const { id } = useParams<{ id: string }>()
  const { data: company, isLoading: companyLoading } = useAdminCompany(id)
  const { data: challenges, isLoading: challengesLoading } = useChallengesByCompany(id)
  const retryEnrichment = useRetryEnrichment()

  if (companyLoading || challengesLoading) return <LoadingSpinner />
  if (!company) return <div>Empresa não encontrada</div>

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-4">
          <div className="p-3 bg-navy-900/5 rounded-lg">
            <Building2 className="h-8 w-8 text-navy-700" />
          </div>
          <div>
            <h1 className="text-2xl font-heading font-bold">{company.name}</h1>
            <div className="flex items-center gap-2 mt-1">
              <Badge variant={
                company.enrichment_status === 'completed' ? 'default' :
                company.enrichment_status === 'failed' ? 'destructive' :
                'secondary'
              }>
                {company.enrichment_status}
              </Badge>
            </div>
          </div>
        </div>

        <Button
          onClick={() => retryEnrichment.mutate(id)}
          disabled={retryEnrichment.isPending || company.enrichment_status === 'processing'}
          variant="outline"
        >
          {retryEnrichment.isPending ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="mr-2 h-4 w-4" />
          )}
          Re-enriquecer
        </Button>
      </div>

      {/* Company Info */}
      <Card>
        <CardHeader>
          <CardTitle>Informações</CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {company.website && (
            <div>
              <p className="text-xs text-text-secondary">Website</p>
              <a href={company.website} target="_blank" className="text-sm hover:underline">
                {company.website}
              </a>
            </div>
          )}
          {company.industry && (
            <div>
              <p className="text-xs text-text-secondary">Indústria</p>
              <p className="text-sm">{company.industry}</p>
            </div>
          )}
          {company.location && (
            <div>
              <p className="text-xs text-text-secondary">Localização</p>
              <p className="text-sm">{company.location}</p>
            </div>
          )}
          {company.company_size && (
            <div>
              <p className="text-xs text-text-secondary">Tamanho</p>
              <p className="text-sm">{company.company_size}</p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Challenges */}
      <div className="space-y-4">
        <h2 className="text-xl font-heading font-semibold">Desafios</h2>
        {challenges?.length ? (
          <div className="grid gap-4 md:grid-cols-2">
            {challenges.map(challenge => (
              <ChallengeCard
                key={challenge.id}
                challenge={challenge}
                showAdminActions
              />
            ))}
          </div>
        ) : (
          <p className="text-text-secondary">Nenhum desafio criado.</p>
        )}
      </div>
    </div>
  )
}
```

## Verification
- [ ] Admin submissions shows all with company link
- [ ] Admin can generate-all for any analysis
- [ ] Visibility toggles work
- [ ] Retry enrichment works
- [ ] Access code displays for public analyses

## Files Created/Modified
- `src/app/(admin)/admin/submissions/page.tsx`
- `src/app/(admin)/admin/analyses/[id]/page.tsx`
- `src/app/(admin)/admin/companies/[id]/page.tsx`
- `src/lib/hooks/use-admin.ts`
- `src/lib/api.ts` (adminApi additions)
