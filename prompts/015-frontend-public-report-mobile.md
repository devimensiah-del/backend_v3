# Prompt 015: Public Report & Mobile Responsive Design

## Objective
Finalize public report viewing via access code and ensure mobile-first responsive design across all pages.

## Working Directory
```bash
cd ../frontend_v2
```

## Tasks

### 1. Update Public Report Page (`src/app/(public)/report/[code]/page.tsx`)

```tsx
'use client'

import { useParams } from 'next/navigation'
import { usePublicReport } from '@/lib/hooks/use-report'
import { FrameworkViewer } from '@/components/features/analysis/framework-viewer'
import { LoadingSpinner } from '@/components/shared/loading-spinner'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Building2, Calendar, AlertCircle } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { ptBR } from 'date-fns/locale'
import Image from 'next/image'

export default function PublicReportPage() {
  const { code } = useParams<{ code: string }>()
  const { data: report, isLoading, error } = usePublicReport(code)

  if (isLoading) {
    return (
      <div className="min-h-screen bg-surface-paper flex items-center justify-center">
        <LoadingSpinner />
      </div>
    )
  }

  if (error || !report) {
    return (
      <div className="min-h-screen bg-surface-paper flex items-center justify-center p-4">
        <Card className="max-w-md w-full">
          <CardContent className="py-12 text-center">
            <AlertCircle className="mx-auto h-12 w-12 text-red-500 mb-4" />
            <h1 className="text-xl font-heading font-bold mb-2">
              Relatório Não Encontrado
            </h1>
            <p className="text-text-secondary">
              O código de acesso é inválido ou o relatório não está mais disponível.
            </p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-surface-paper">
      {/* Header */}
      <header className="bg-white border-b border-line sticky top-0 z-10">
        <div className="max-w-5xl mx-auto px-4 py-4 flex items-center justify-between">
          <Image
            src="/images/landing/logo.png"
            alt="IMENSIAH"
            width={120}
            height={32}
            className="h-8 w-auto"
          />
          <Badge variant="outline">Relatório Público</Badge>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-5xl mx-auto px-4 py-8">
        {/* Report Header */}
        <div className="mb-8">
          <div className="flex flex-col sm:flex-row sm:items-center gap-4 mb-4">
            <div className="p-3 bg-navy-900/5 rounded-lg w-fit">
              <Building2 className="h-8 w-8 text-navy-700" />
            </div>
            <div>
              <h1 className="text-2xl sm:text-3xl font-heading font-bold text-text-primary">
                {report.company_name}
              </h1>
              <p className="text-text-secondary mt-1">
                Análise Estratégica Completa
              </p>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-4 text-sm text-text-secondary">
            {report.challenge_type && (
              <Badge>{report.challenge_type.replace(/_/g, ' ')}</Badge>
            )}
            <div className="flex items-center gap-1">
              <Calendar className="h-4 w-4" />
              <span>
                {formatDistanceToNow(new Date(report.created_at), {
                  addSuffix: true,
                  locale: ptBR
                })}
              </span>
            </div>
          </div>
        </div>

        {/* Framework Results */}
        <FrameworkViewer results={report.framework_results} isPublic />
      </main>

      {/* Footer */}
      <footer className="border-t border-line bg-white mt-12">
        <div className="max-w-5xl mx-auto px-4 py-6 text-center text-sm text-text-secondary">
          <p>
            Relatório gerado por{' '}
            <a href="/" className="text-navy-700 hover:underline font-medium">
              IMENSIAH
            </a>
          </p>
          <p className="mt-1">
            Inteligência Artificial + Inteligência Humana para Decisões Estratégicas
          </p>
        </div>
      </footer>
    </div>
  )
}
```

### 2. Update Framework Viewer for Mobile (`src/components/features/analysis/framework-viewer.tsx`)

```tsx
'use client'

import { FrameworkResults } from '@/lib/types/domain'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import ReactMarkdown from 'react-markdown'
import { useState } from 'react'
import { useMediaQuery } from '@/lib/hooks/use-media-query'

interface FrameworkViewerProps {
  results: FrameworkResults
  isPublic?: boolean
}

const frameworkOrder = [
  { key: 'pestel', name: 'PESTEL' },
  { key: 'porter', name: 'Porter 7 Forças' },
  { key: 'tam_sam_som', name: 'TAM SAM SOM' },
  { key: 'swot', name: 'SWOT' },
  { key: 'benchmarking', name: 'Benchmarking' },
  { key: 'blue_ocean', name: 'Blue Ocean' },
  { key: 'growth_hacking', name: 'Growth Hacking' },
  { key: 'scenarios', name: 'Cenários' },
  { key: 'okrs', name: 'OKRs' },
  { key: 'bsc', name: 'BSC' },
  { key: 'decision_matrix', name: 'Matriz Decisão' },
  { key: 'synthesis', name: 'Síntese Executiva' },
]

export function FrameworkViewer({ results, isPublic = false }: FrameworkViewerProps) {
  const [activeTab, setActiveTab] = useState('pestel')
  const isMobile = useMediaQuery('(max-width: 768px)')

  // Filter to only show frameworks with results
  const availableFrameworks = frameworkOrder.filter(
    f => results[f.key as keyof FrameworkResults]
  )

  if (isMobile) {
    // Accordion for mobile
    return (
      <Accordion type="single" collapsible className="space-y-2">
        {availableFrameworks.map((framework) => {
          const content = results[framework.key as keyof FrameworkResults]
          if (!content) return null

          return (
            <AccordionItem
              key={framework.key}
              value={framework.key}
              className="bg-white border border-line rounded-lg px-4"
            >
              <AccordionTrigger className="hover:no-underline">
                <span className="font-heading font-semibold">{framework.name}</span>
              </AccordionTrigger>
              <AccordionContent>
                <div className="prose prose-sm max-w-none pb-4">
                  <ReactMarkdown>
                    {typeof content === 'string' ? content : JSON.stringify(content, null, 2)}
                  </ReactMarkdown>
                </div>
              </AccordionContent>
            </AccordionItem>
          )
        })}
      </Accordion>
    )
  }

  // Tabs for desktop
  return (
    <Card>
      <CardHeader className="pb-0">
        <CardTitle>Frameworks de Análise</CardTitle>
      </CardHeader>
      <CardContent className="pt-4">
        <Tabs value={activeTab} onValueChange={setActiveTab}>
          <TabsList className="flex flex-wrap h-auto gap-1 bg-transparent p-0 mb-6">
            {availableFrameworks.map((framework) => (
              <TabsTrigger
                key={framework.key}
                value={framework.key}
                className="data-[state=active]:bg-navy-900 data-[state=active]:text-white px-3 py-1.5 text-sm"
              >
                {framework.name}
              </TabsTrigger>
            ))}
          </TabsList>

          {availableFrameworks.map((framework) => {
            const content = results[framework.key as keyof FrameworkResults]
            if (!content) return null

            return (
              <TabsContent key={framework.key} value={framework.key}>
                <div className="prose max-w-none">
                  <ReactMarkdown>
                    {typeof content === 'string' ? content : JSON.stringify(content, null, 2)}
                  </ReactMarkdown>
                </div>
              </TabsContent>
            )
          })}
        </Tabs>
      </CardContent>
    </Card>
  )
}
```

### 3. Create Media Query Hook (`src/lib/hooks/use-media-query.ts`)

```typescript
import { useState, useEffect } from 'react'

export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(false)

  useEffect(() => {
    const media = window.matchMedia(query)

    // Set initial value
    setMatches(media.matches)

    // Listen for changes
    const listener = (e: MediaQueryListEvent) => setMatches(e.matches)
    media.addEventListener('change', listener)

    return () => media.removeEventListener('change', listener)
  }, [query])

  return matches
}
```

### 4. Mobile Responsive Updates

#### Update Wizard Header for Mobile (`src/components/features/wizard/wizard-progress.tsx`)

```tsx
// Add responsive wrapper
<div className="flex flex-col items-center">
  <p className="text-sm font-medium text-text-primary mb-2">
    Passo {currentStep} de {totalSteps}
  </p>

  {/* Hide dots on small screens, show progress bar instead */}
  <div className="hidden sm:flex items-center gap-1">
    {/* ... existing dots ... */}
  </div>

  {/* Mobile: Simple progress bar */}
  <div className="sm:hidden w-32 h-1.5 bg-gray-200 rounded-full overflow-hidden">
    <div
      className="h-full bg-navy-700 transition-all"
      style={{ width: `${(currentStep / totalSteps) * 100}%` }}
    />
  </div>
</div>
```

#### Update Company Card for Mobile (`src/components/features/company/company-card.tsx`)

```tsx
'use client'

import { Company } from '@/lib/types/domain'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Building2, ChevronRight } from 'lucide-react'
import Link from 'next/link'

interface CompanyCardProps {
  company: Company
}

export function CompanyCard({ company }: CompanyCardProps) {
  const challengeCount = company.challenges?.length || 0
  const completedCount = company.challenges?.filter(
    c => c.latest_analysis?.status === 'completed'
  ).length || 0

  return (
    <Link href={`/dashboard/companies/${company.id}`}>
      <Card className="hover:border-gold-500/50 transition-colors cursor-pointer">
        <CardContent className="p-4 sm:p-6">
          <div className="flex items-start gap-3 sm:gap-4">
            <div className="p-2 sm:p-3 bg-navy-900/5 rounded-lg shrink-0">
              <Building2 className="h-5 w-5 sm:h-6 sm:w-6 text-navy-700" />
            </div>

            <div className="flex-1 min-w-0">
              <h3 className="font-heading font-semibold text-text-primary truncate">
                {company.name}
              </h3>

              <p className="text-sm text-text-secondary mt-0.5 truncate">
                {company.industry || 'Sem indústria'}
              </p>

              <div className="flex flex-wrap items-center gap-2 mt-3">
                <Badge variant={
                  company.enrichment_status === 'completed' ? 'default' :
                  company.enrichment_status === 'failed' ? 'destructive' :
                  'outline'
                }>
                  {company.enrichment_status === 'completed' ? 'Enriquecido' :
                   company.enrichment_status === 'failed' ? 'Erro' :
                   'Pendente'}
                </Badge>

                {challengeCount > 0 && (
                  <span className="text-xs text-text-secondary">
                    {completedCount}/{challengeCount} análises
                  </span>
                )}
              </div>
            </div>

            <ChevronRight className="h-5 w-5 text-text-tertiary shrink-0 hidden sm:block" />
          </div>
        </CardContent>
      </Card>
    </Link>
  )
}
```

#### Update Dashboard Layout for Mobile (`src/app/(dashboard)/layout.tsx`)

```tsx
'use client'

import { UserSidebar } from '@/components/shared/user-sidebar'
import { AppHeader } from '@/components/shared/app-header'
import { useState } from 'react'
import { Menu, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const [sidebarOpen, setSidebarOpen] = useState(false)

  return (
    <div className="min-h-screen bg-surface-paper">
      {/* Mobile Header */}
      <header className="lg:hidden sticky top-0 z-30 bg-white border-b border-line px-4 py-3 flex items-center justify-between">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => setSidebarOpen(true)}
        >
          <Menu className="h-5 w-5" />
        </Button>
        <span className="font-heading font-bold">IMENSIAH</span>
        <div className="w-10" /> {/* Spacer */}
      </header>

      {/* Mobile Sidebar Overlay */}
      {sidebarOpen && (
        <div
          className="lg:hidden fixed inset-0 bg-black/50 z-40"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside className={cn(
        'fixed inset-y-0 left-0 z-50 w-64 bg-white border-r border-line transform transition-transform lg:translate-x-0',
        sidebarOpen ? 'translate-x-0' : '-translate-x-full'
      )}>
        <div className="lg:hidden p-4 flex justify-end">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setSidebarOpen(false)}
          >
            <X className="h-5 w-5" />
          </Button>
        </div>
        <UserSidebar onNavigate={() => setSidebarOpen(false)} />
      </aside>

      {/* Main Content */}
      <main className="lg:ml-64 min-h-screen">
        <div className="hidden lg:block">
          <AppHeader />
        </div>
        <div className="p-4 sm:p-6 lg:p-8">
          {children}
        </div>
      </main>
    </div>
  )
}
```

### 5. Global Responsive Utilities

Add to `src/app/globals.css`:

```css
/* Mobile-first responsive utilities */
@layer utilities {
  /* Touch-friendly tap targets */
  .touch-target {
    @apply min-h-[44px] min-w-[44px];
  }

  /* Safe area for notch devices */
  .safe-area-inset {
    padding-top: env(safe-area-inset-top);
    padding-bottom: env(safe-area-inset-bottom);
    padding-left: env(safe-area-inset-left);
    padding-right: env(safe-area-inset-right);
  }

  /* Hide scrollbar but allow scroll */
  .scrollbar-hide {
    -ms-overflow-style: none;
    scrollbar-width: none;
  }
  .scrollbar-hide::-webkit-scrollbar {
    display: none;
  }

  /* Text truncation */
  .truncate-2 {
    overflow: hidden;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  .truncate-3 {
    overflow: hidden;
    display: -webkit-box;
    -webkit-line-clamp: 3;
    -webkit-box-orient: vertical;
  }
}

/* Responsive container */
.container-responsive {
  @apply max-w-7xl mx-auto px-4 sm:px-6 lg:px-8;
}
```

## Mobile Checklist

Apply these patterns throughout:

- [ ] **Touch targets**: Minimum 44x44px for all interactive elements
- [ ] **Font sizes**: Minimum 16px for body text (prevents zoom on iOS)
- [ ] **Spacing**: Use `p-4 sm:p-6 lg:p-8` pattern
- [ ] **Navigation**: Hamburger menu on mobile, sidebar on desktop
- [ ] **Cards**: Stack vertically on mobile, grid on desktop
- [ ] **Modals**: Full-screen on mobile, centered on desktop
- [ ] **Tables**: Convert to cards/lists on mobile
- [ ] **Forms**: Single column on mobile
- [ ] **Buttons**: Full-width on mobile when appropriate

## Verification
- [ ] Public report loads with access code
- [ ] Report displays all frameworks
- [ ] Mobile navigation works (hamburger menu)
- [ ] Wizard is usable on mobile
- [ ] Company/challenge cards stack on mobile
- [ ] Touch targets are adequate size
- [ ] No horizontal scroll on mobile

## Files Created/Modified
- `src/app/(public)/report/[code]/page.tsx`
- `src/components/features/analysis/framework-viewer.tsx`
- `src/lib/hooks/use-media-query.ts` (new)
- `src/app/(dashboard)/layout.tsx`
- `src/components/features/company/company-card.tsx`
- `src/app/globals.css`
