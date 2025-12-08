# Prompt 010: Frontend Types & API Client Update

## Objective
Update frontend_v2 types and API client to align with the new entity hierarchy: User → Companies → Challenges → Analysis.

## Working Directory
```bash
cd ../frontend_v2
```

## Context
The backend has evolved to a challenge-centric model where:
- **Submission** = Entry point / audit trail (admin tracking only)
- **Company** = The business entity users care about
- **Challenge** = The strategic question being asked (growth, transform, etc.)
- **Analysis** = The 12-framework answer to that challenge via wizard

## Tasks

### 1. Update Domain Types (`src/lib/types/domain.ts`)

#### Add Challenge Types (if not complete):
```typescript
export type ChallengeCategory = 'growth' | 'transform' | 'transition' | 'compete' | 'funding'

export type ChallengeType =
  // Growth
  | 'growth_organic' | 'growth_geographic' | 'growth_segment' | 'growth_product' | 'growth_channel'
  // Transform
  | 'transform_digital' | 'transform_model' | 'transform_culture' | 'transform_operational'
  // Transition
  | 'transition_succession' | 'transition_exit' | 'transition_merger' | 'transition_turnaround'
  // Compete
  | 'compete_differentiate' | 'compete_defend' | 'compete_reposition'
  // Funding
  | 'funding_raise' | 'funding_debt' | 'funding_ipo'

export interface Challenge {
  id: string
  company_id: string
  challenge_category: ChallengeCategory
  challenge_type: ChallengeType
  business_challenge: string
  created_at: string
  updated_at: string
  // Derived from related analyses
  latest_analysis?: Analysis
  analysis_count?: number
}
```

#### Update Company Type:
```typescript
export interface Company {
  id: string
  name: string
  cnpj?: string
  website?: string
  industry?: string
  sector?: string
  company_size?: string
  location?: string
  target_market?: string
  funding_stage?: string
  foundation_year?: string
  headquarters?: string
  target_audience?: string
  value_proposition?: string
  employees_range?: string
  revenue_estimate?: string
  business_model?: string
  market_share_status?: string
  digital_maturity?: number
  competitors?: string[]
  strengths?: string[]
  weaknesses?: string[]
  enrichment_status: 'pending' | 'processing' | 'completed' | 'failed'
  owner_ids?: string[]  // Multiple owners allowed
  created_at: string
  updated_at: string
  // Derived
  challenges?: Challenge[]
}
```

#### Remove from Analysis type:
- `is_blurred` (removed in v2_013)
- Any version-related fields

### 2. Update API Client (`src/lib/api.ts`)

#### Add Challenge API:
```typescript
export const challengesApi = {
  // Get challenge types for dropdown
  getTypes: () => api.get<{ categories: Record<ChallengeCategory, ChallengeType[]> }>('/challenges/types'),

  // Create challenge for existing company
  create: (data: {
    company_id: string
    challenge_category: ChallengeCategory
    challenge_type: ChallengeType
    business_challenge: string
  }) => api.post<Challenge>('/challenges', data),

  // Update challenge
  update: (id: string, data: Partial<Challenge>) => api.put<Challenge>(`/challenges/${id}`, data),

  // Delete challenge
  delete: (id: string) => api.delete(`/challenges/${id}`),

  // List challenges for company
  listByCompany: (companyId: string) => api.get<Challenge[]>(`/companies/${companyId}/challenges`),
}
```

#### Update Company API:
```typescript
export const companiesApi = {
  // Existing...
  list: () => api.get<Company[]>('/companies'),
  getById: (id: string) => api.get<Company>(`/companies/${id}`),

  // Add these:
  update: (id: string, data: Partial<Company>) => api.put<Company>(`/companies/${id}`, data),
  delete: (id: string) => api.delete(`/companies/${id}`),
  reEnrich: (id: string) => api.post(`/companies/${id}/re-enrich`),

  // Get with challenges included
  getWithChallenges: (id: string) => api.get<Company & { challenges: Challenge[] }>(`/companies/${id}?include=challenges`),
}
```

#### Update Wizard API:
```typescript
export const wizardApi = {
  // Start wizard from challenge
  start: (data: { company_id: string; challenge_id: string }) =>
    api.post<{ analysis_id: string }>('/wizard/start', data),

  // Get current wizard state
  getState: (analysisId: string) =>
    api.get<WizardState>(`/analyses/${analysisId}/wizard`),

  // Generate framework output
  generate: (analysisId: string, input?: { context?: string }) =>
    api.post<WizardState>(`/analyses/${analysisId}/wizard/generate`, input),

  // Approve current step
  approve: (analysisId: string) =>
    api.post<WizardState>(`/analyses/${analysisId}/wizard/approve`),

  // Request refinement
  refine: (analysisId: string, data: { context: string }) =>
    api.post<WizardState>(`/analyses/${analysisId}/wizard/refine`, data),

  // Admin: Generate all frameworks at once
  generateAll: (analysisId: string) =>
    api.post(`/admin/analyses/${analysisId}/wizard/generate-all`),
}
```

### 3. Add React Query Hooks (`src/lib/hooks/use-challenges.ts`)

```typescript
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { challengesApi } from '../api'
import { toast } from 'sonner'

export function useChallengeTypes() {
  return useQuery({
    queryKey: ['challengeTypes'],
    queryFn: () => challengesApi.getTypes(),
    staleTime: Infinity, // Types don't change
  })
}

export function useChallengesByCompany(companyId: string) {
  return useQuery({
    queryKey: ['challenges', companyId],
    queryFn: () => challengesApi.listByCompany(companyId),
    enabled: !!companyId,
  })
}

export function useCreateChallenge() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: challengesApi.create,
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['challenges', data.company_id] })
      toast.success('Desafio criado com sucesso')
    },
    onError: () => toast.error('Erro ao criar desafio'),
  })
}

export function useUpdateChallenge() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Challenge> }) =>
      challengesApi.update(id, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['challenges', data.company_id] })
      toast.success('Desafio atualizado')
    },
    onError: () => toast.error('Erro ao atualizar desafio'),
  })
}

export function useDeleteChallenge() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: challengesApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['challenges'] })
      toast.success('Desafio excluído')
    },
    onError: () => toast.error('Erro ao excluir desafio'),
  })
}
```

### 4. Update Company Hooks (`src/lib/hooks/use-companies.ts`)

Add these mutations:
```typescript
export function useUpdateCompany() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<Company> }) =>
      companiesApi.update(id, data),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['companies'] })
      queryClient.invalidateQueries({ queryKey: ['company', data.id] })
      toast.success('Empresa atualizada')
    },
    onError: () => toast.error('Erro ao atualizar empresa'),
  })
}

export function useDeleteCompany() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: companiesApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['companies'] })
      toast.success('Empresa excluída')
    },
    onError: () => toast.error('Erro ao excluir empresa'),
  })
}

export function useReEnrichCompany() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: companiesApi.reEnrich,
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ['company', id] })
      toast.success('Enriquecimento iniciado')
    },
    onError: () => toast.error('Erro ao iniciar enriquecimento'),
  })
}
```

## Verification
- [ ] All types compile without errors
- [ ] API client exports all new functions
- [ ] Hooks properly invalidate caches
- [ ] No references to removed fields (is_blurred, version fields)

## Files Modified
- `src/lib/types/domain.ts`
- `src/lib/api.ts`
- `src/lib/hooks/use-challenges.ts` (new or update)
- `src/lib/hooks/use-companies.ts`
- `src/lib/hooks/use-wizard.ts`
