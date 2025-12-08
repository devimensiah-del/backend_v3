# Prompt 013: Company CRUD & Re-Enrichment

## Objective
Implement full Company CRUD operations: edit, delete, and re-enrich capabilities.

## Working Directory
```bash
cd ../frontend_v2
```

## Tasks

### 1. Create Company Header with Actions (`src/components/features/company/company-header.tsx`)

```tsx
'use client'

import { Company } from '@/lib/types/domain'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  MoreVertical, Pencil, Trash2, RefreshCw,
  Building2, Globe, MapPin, Users, Loader2
} from 'lucide-react'
import { useState } from 'react'
import { EditCompanyModal } from './edit-company-modal'
import { DeleteCompanyDialog } from './delete-company-dialog'
import { useReEnrichCompany } from '@/lib/hooks/use-companies'

interface CompanyHeaderProps {
  company: Company
}

const enrichmentStatusLabels: Record<string, { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline' }> = {
  pending: { label: 'Pendente', variant: 'outline' },
  processing: { label: 'Processando', variant: 'secondary' },
  completed: { label: 'Enriquecido', variant: 'default' },
  failed: { label: 'Erro', variant: 'destructive' },
}

export function CompanyHeader({ company }: CompanyHeaderProps) {
  const [showEdit, setShowEdit] = useState(false)
  const [showDelete, setShowDelete] = useState(false)
  const reEnrich = useReEnrichCompany()

  const enrichmentStatus = enrichmentStatusLabels[company.enrichment_status] || enrichmentStatusLabels.pending

  return (
    <div className="space-y-6">
      {/* Header with Actions */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-4">
          <div className="p-3 bg-navy-900/5 rounded-lg">
            <Building2 className="h-8 w-8 text-navy-700" />
          </div>
          <div>
            <h1 className="text-2xl font-heading font-bold text-text-primary">
              {company.name}
            </h1>
            <div className="flex items-center gap-2 mt-1">
              <Badge variant={enrichmentStatus.variant}>
                {enrichmentStatus.label}
              </Badge>
              {company.industry && (
                <span className="text-sm text-text-secondary">{company.industry}</span>
              )}
            </div>
          </div>
        </div>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="outline" size="icon">
              <MoreVertical className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={() => setShowEdit(true)}>
              <Pencil className="mr-2 h-4 w-4" />
              Editar Empresa
            </DropdownMenuItem>
            <DropdownMenuItem
              onClick={() => reEnrich.mutate(company.id)}
              disabled={reEnrich.isPending || company.enrichment_status === 'processing'}
            >
              {reEnrich.isPending ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <RefreshCw className="mr-2 h-4 w-4" />
              )}
              Re-enriquecer Dados
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={() => setShowDelete(true)}
              className="text-red-600"
            >
              <Trash2 className="mr-2 h-4 w-4" />
              Excluir Empresa
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {/* Company Info Grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {company.website && (
          <div className="flex items-center gap-2 text-sm text-text-secondary">
            <Globe className="h-4 w-4" />
            <a
              href={company.website}
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-navy-700 hover:underline truncate"
            >
              {company.website.replace(/^https?:\/\//, '')}
            </a>
          </div>
        )}
        {company.location && (
          <div className="flex items-center gap-2 text-sm text-text-secondary">
            <MapPin className="h-4 w-4" />
            <span>{company.location}</span>
          </div>
        )}
        {company.company_size && (
          <div className="flex items-center gap-2 text-sm text-text-secondary">
            <Users className="h-4 w-4" />
            <span>{company.company_size}</span>
          </div>
        )}
        {company.funding_stage && (
          <div className="flex items-center gap-2 text-sm text-text-secondary">
            <span className="font-medium">Stage:</span>
            <span>{company.funding_stage}</span>
          </div>
        )}
      </div>

      {/* Enriched Data Section (if completed) */}
      {company.enrichment_status === 'completed' && company.value_proposition && (
        <div className="bg-white border border-line rounded-lg p-4 space-y-3">
          <h3 className="text-sm font-medium uppercase tracking-widest text-text-secondary">
            Dados Enriquecidos
          </h3>
          <p className="text-sm text-text-primary">
            {company.value_proposition}
          </p>

          {company.competitors && company.competitors.length > 0 && (
            <div>
              <p className="text-xs font-medium text-text-secondary mb-1">Concorrentes:</p>
              <div className="flex flex-wrap gap-1">
                {company.competitors.map((comp, i) => (
                  <Badge key={i} variant="outline" className="text-xs">{comp}</Badge>
                ))}
              </div>
            </div>
          )}

          {company.strengths && company.strengths.length > 0 && (
            <div>
              <p className="text-xs font-medium text-text-secondary mb-1">Pontos Fortes:</p>
              <ul className="text-xs text-text-primary list-disc list-inside">
                {company.strengths.slice(0, 3).map((s, i) => (
                  <li key={i}>{s}</li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      <EditCompanyModal
        company={company}
        open={showEdit}
        onOpenChange={setShowEdit}
      />

      <DeleteCompanyDialog
        company={company}
        open={showDelete}
        onOpenChange={setShowDelete}
      />
    </div>
  )
}
```

### 2. Create Edit Company Modal (`src/components/features/company/edit-company-modal.tsx`)

```tsx
'use client'

import { Company } from '@/lib/types/domain'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { useUpdateCompany } from '@/lib/hooks/use-companies'
import { useForm } from 'react-hook-form'
import { Loader2 } from 'lucide-react'

interface EditCompanyModalProps {
  company: Company
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface FormData {
  name: string
  website?: string
  industry?: string
  location?: string
  company_size?: string
  target_market?: string
  value_proposition?: string
}

export function EditCompanyModal({ company, open, onOpenChange }: EditCompanyModalProps) {
  const updateCompany = useUpdateCompany()

  const { register, handleSubmit, formState: { errors } } = useForm<FormData>({
    defaultValues: {
      name: company.name,
      website: company.website || '',
      industry: company.industry || '',
      location: company.location || '',
      company_size: company.company_size || '',
      target_market: company.target_market || '',
      value_proposition: company.value_proposition || '',
    },
  })

  const onSubmit = (data: FormData) => {
    updateCompany.mutate(
      { id: company.id, data },
      { onSuccess: () => onOpenChange(false) }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Editar Empresa</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Nome da Empresa *</Label>
            <Input
              id="name"
              {...register('name', { required: 'Nome é obrigatório' })}
              className="input-editorial"
            />
            {errors.name && (
              <p className="text-xs text-red-500">{errors.name.message}</p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="website">Website</Label>
              <Input
                id="website"
                {...register('website')}
                placeholder="https://..."
                className="input-editorial"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="industry">Indústria</Label>
              <Input
                id="industry"
                {...register('industry')}
                className="input-editorial"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="location">Localização</Label>
              <Input
                id="location"
                {...register('location')}
                className="input-editorial"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="company_size">Tamanho</Label>
              <Input
                id="company_size"
                {...register('company_size')}
                placeholder="ex: 10-50 funcionários"
                className="input-editorial"
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="target_market">Mercado Alvo</Label>
            <Input
              id="target_market"
              {...register('target_market')}
              className="input-editorial"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="value_proposition">Proposta de Valor</Label>
            <Textarea
              id="value_proposition"
              {...register('value_proposition')}
              className="input-editorial min-h-[100px]"
            />
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancelar
            </Button>
            <Button type="submit" disabled={updateCompany.isPending}>
              {updateCompany.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Salvando...
                </>
              ) : (
                'Salvar'
              )}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
```

### 3. Create Delete Company Dialog (`src/components/features/company/delete-company-dialog.tsx`)

```tsx
'use client'

import { Company } from '@/lib/types/domain'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { useDeleteCompany } from '@/lib/hooks/use-companies'
import { useRouter } from 'next/navigation'
import { Loader2 } from 'lucide-react'

interface DeleteCompanyDialogProps {
  company: Company
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function DeleteCompanyDialog({ company, open, onOpenChange }: DeleteCompanyDialogProps) {
  const router = useRouter()
  const deleteCompany = useDeleteCompany()

  const handleDelete = () => {
    deleteCompany.mutate(company.id, {
      onSuccess: () => {
        onOpenChange(false)
        router.push('/dashboard')
      }
    })
  }

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Excluir Empresa</AlertDialogTitle>
          <AlertDialogDescription>
            Tem certeza que deseja excluir <strong>{company.name}</strong>?
            <br /><br />
            Esta ação irá excluir permanentemente:
            <ul className="list-disc list-inside mt-2 space-y-1">
              <li>Todos os desafios associados</li>
              <li>Todas as análises associadas</li>
              <li>Todos os dados enriquecidos</li>
            </ul>
            <br />
            <strong className="text-red-600">Esta ação não pode ser desfeita.</strong>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancelar</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleDelete}
            className="bg-red-600 hover:bg-red-700"
            disabled={deleteCompany.isPending}
          >
            {deleteCompany.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Excluindo...
              </>
            ) : (
              'Excluir Empresa'
            )}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
```

### 4. Create New Challenge Modal (`src/components/dashboard/NewChallengeModal.tsx`)

```tsx
'use client'

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useCreateChallenge, useChallengeTypes } from '@/lib/hooks/use-challenges'
import { useForm, Controller } from 'react-hook-form'
import { Loader2 } from 'lucide-react'
import { ChallengeCategory, ChallengeType } from '@/lib/types/domain'

interface NewChallengeModalProps {
  companyId: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface FormData {
  challenge_category: ChallengeCategory
  challenge_type: ChallengeType
  business_challenge: string
}

const categoryLabels: Record<ChallengeCategory, string> = {
  growth: 'Crescimento',
  transform: 'Transformação',
  transition: 'Transição',
  compete: 'Competição',
  funding: 'Captação',
}

const typeLabels: Record<ChallengeType, string> = {
  // Growth
  growth_organic: 'Crescimento Orgânico',
  growth_geographic: 'Expansão Geográfica',
  growth_segment: 'Novos Segmentos',
  growth_product: 'Novos Produtos',
  growth_channel: 'Novos Canais',
  // Transform
  transform_digital: 'Transformação Digital',
  transform_model: 'Mudança de Modelo',
  transform_culture: 'Transformação Cultural',
  transform_operational: 'Eficiência Operacional',
  // Transition
  transition_succession: 'Sucessão',
  transition_exit: 'Saída/Venda',
  transition_merger: 'Fusão/Aquisição',
  transition_turnaround: 'Turnaround',
  // Compete
  compete_differentiate: 'Diferenciação',
  compete_defend: 'Defesa de Mercado',
  compete_reposition: 'Reposicionamento',
  // Funding
  funding_raise: 'Captação de Investimento',
  funding_debt: 'Financiamento',
  funding_ipo: 'IPO',
}

export function NewChallengeModal({ companyId, open, onOpenChange }: NewChallengeModalProps) {
  const createChallenge = useCreateChallenge()
  const { data: challengeTypes } = useChallengeTypes()

  const { register, handleSubmit, control, watch, formState: { errors } } = useForm<FormData>()

  const selectedCategory = watch('challenge_category')

  const onSubmit = (data: FormData) => {
    createChallenge.mutate(
      { company_id: companyId, ...data },
      { onSuccess: () => onOpenChange(false) }
    )
  }

  // Get types for selected category
  const availableTypes = selectedCategory && challengeTypes?.categories
    ? challengeTypes.categories[selectedCategory] || []
    : []

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Novo Desafio</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label>Categoria do Desafio *</Label>
            <Controller
              name="challenge_category"
              control={control}
              rules={{ required: 'Selecione uma categoria' }}
              render={({ field }) => (
                <Select onValueChange={field.onChange} value={field.value}>
                  <SelectTrigger className="input-editorial">
                    <SelectValue placeholder="Selecione a categoria..." />
                  </SelectTrigger>
                  <SelectContent>
                    {Object.entries(categoryLabels).map(([value, label]) => (
                      <SelectItem key={value} value={value}>
                        {label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            />
            {errors.challenge_category && (
              <p className="text-xs text-red-500">{errors.challenge_category.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label>Tipo de Desafio *</Label>
            <Controller
              name="challenge_type"
              control={control}
              rules={{ required: 'Selecione o tipo' }}
              render={({ field }) => (
                <Select
                  onValueChange={field.onChange}
                  value={field.value}
                  disabled={!selectedCategory}
                >
                  <SelectTrigger className="input-editorial">
                    <SelectValue placeholder={selectedCategory ? "Selecione o tipo..." : "Primeiro selecione a categoria"} />
                  </SelectTrigger>
                  <SelectContent>
                    {availableTypes.map((type) => (
                      <SelectItem key={type} value={type}>
                        {typeLabels[type] || type}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            />
            {errors.challenge_type && (
              <p className="text-xs text-red-500">{errors.challenge_type.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="business_challenge">Descreva seu Desafio *</Label>
            <Textarea
              id="business_challenge"
              {...register('business_challenge', {
                required: 'Descreva seu desafio',
                minLength: { value: 20, message: 'Mínimo 20 caracteres' }
              })}
              placeholder="Ex: Queremos aumentar nossa participação de mercado em 50% nos próximos 2 anos..."
              className="input-editorial min-h-[120px]"
            />
            {errors.business_challenge && (
              <p className="text-xs text-red-500">{errors.business_challenge.message}</p>
            )}
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              Cancelar
            </Button>
            <Button type="submit" disabled={createChallenge.isPending}>
              {createChallenge.isPending ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Criando...
                </>
              ) : (
                'Criar Desafio'
              )}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
```

## Verification
- [ ] Company header shows all info
- [ ] Edit modal pre-fills data
- [ ] Update saves and refreshes
- [ ] Delete shows confirmation
- [ ] Delete removes and redirects
- [ ] Re-enrich triggers enrichment
- [ ] New challenge modal works
- [ ] Challenge types load dynamically

## Files Created
- `src/components/features/company/company-header.tsx`
- `src/components/features/company/edit-company-modal.tsx`
- `src/components/features/company/delete-company-dialog.tsx`
- `src/components/dashboard/NewChallengeModal.tsx`
