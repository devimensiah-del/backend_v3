# Report V2 Templates - Premium A4 PDF Reports

## Overview

Report V2 is a complete redesign of IMENSIAH's strategic analysis reports, optimized for professional PDF generation with premium McKinsey/BCG-style consulting aesthetics.

### Design Philosophy

- **Premium consulting style**: Ultra-clean, minimal layouts with high whitespace
- **A4 Portrait format**: 595px × 842px (210mm × 297mm @ 72 DPI)
- **Navy + Gold branding**: #0A101D (Navy) + #B89E68 (Gold)
- **Strict one-page constraint**: Each framework fits exactly one A4 page
- **PDF-optimized**: Inline styles, system fonts, tight content control

---

## Template Structure (16 Pages)

| # | File | Content | Data Source |
|---|------|---------|-------------|
| 01 | `01_cover.html` | Premium cover with logo | `.CompanyName`, `.Industry`, `.Market`, `.Date` |
| 02 | `02_exec_summary.html` | Executive summary | `.Synthesis` |
| 03 | `03_toc.html` | Table of contents | Static |
| 04 | `04_pestel.html` | PESTEL Analysis | `.PESTEL` |
| 05 | `05_porter.html` | Porter's 5 Forces | `.Porter` |
| 06 | `06_swot.html` | SWOT Analysis | `.SWOT` |
| 07 | `07_tam_sam_som.html` | Market sizing | `.TamSamSom` |
| 08 | `08_ocean.html` | Blue Ocean Strategy | `.BlueOcean` |
| 09 | `09_okrs.html` | OKRs | `.OKRs` |
| 10 | `10_business_model.html` | Balanced Scorecard | `.BSC` |
| 11 | `11_competitive_analysis.html` | Benchmarking | `.Benchmarking` |
| 12 | `12_financial_projections.html` | Financial scenarios | `.Scenarios` |
| 13 | `13_gtm_strategy.html` | Growth strategy | `.GrowthHacking` |
| 14 | `14_risk_assessment.html` | Risk assessment | `.Scenarios` |
| 15 | `15_roadmap.html` | Strategic roadmap | `.DecisionMatrix` |
| 16 | `16_appendix.html` | Methodology & references | Static + `.Date` |

---

## Technical Specifications

### Page Dimensions
```
Width:  595px (8.27 inches)
Height: 842px (11.7 inches)
Margins: 40px all sides
Safe content area: 515px × 762px
```

### Typography
- **Font stack**: System fonts (-apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif)
- **Headings**: 24-28px, weight 300 (light)
- **Body text**: 10-12px, line-height 1.4-1.6
- **Labels**: 9-10px, uppercase, letter-spacing 0.15em

### Color Palette
```css
/* Primary */
--navy-900: #0A101D;      /* Navy primary */
--gold-500: #B89E68;      /* Gold accent */

/* Neutrals */
--surface-paper: #F7F6F4; /* Background */
--line-color: #E5E0D6;    /* Borders */
--text-primary: #111827;  /* Body text */
--text-secondary: #52525B; /* Secondary text */
--text-tertiary: #71717A;  /* Tertiary text */

/* Semantic */
--success: #10b981;       /* Green (strengths, growth) */
--warning: #f59e0b;       /* Amber (caution) */
--danger: #ef4444;        /* Red (risks, threats) */
--info: #3b82f6;          /* Blue (opportunities) */
```

---

## Implementation Guide

### Step 1: Update Gotenberg Configuration

Edit `backend_v3/infrastructure/gotenberg.go`:

```go
// Change from A4 Landscape to A4 Portrait
data.Set("paperWidth", "8.27")   // was 11.7
data.Set("paperHeight", "11.7")  // was 8.27
data.Set("marginTop", "0")
data.Set("marginBottom", "0")
data.Set("marginLeft", "0")
data.Set("marginRight", "0")
data.Set("printBackground", "true")
```

### Step 2: Update Report Service

Edit `backend_v3/domain/report/service.go`:

```go
// Change template directory constant
const templateDir = "templates/report_v2/"  // was "templates/report/"
```

Or create a configuration flag:
```go
func (s *ReportService) SetTemplateVersion(version string) {
    if version == "v2" {
        s.templateDir = "templates/report_v2/"
    } else {
        s.templateDir = "templates/report/"
    }
}
```

### Step 3: Template Path Resolution

Ensure template loading handles the new directory:

```go
templatePaths := []string{
    filepath.Join("backend_v3", templateDir, filename),
    filepath.Join(templateDir, filename),
    filename,
}
```

---

## Content Constraints

### Maximum Content Per Framework

To ensure single-page fit, content is strictly limited:

| Framework | Content Type | Max Items | Max Length |
|-----------|-------------|-----------|------------|
| **PESTEL** | Lists (6 factors) | 5 items/factor | 2 lines/item |
| **Porter** | Paragraphs (5 forces) | 1 paragraph/force | 4 lines max |
| **SWOT** | Lists (4 quadrants) | 6 items/quadrant | 2 lines/item |
| **TAM-SAM-SOM** | Text values | 3 values | 100 chars each |
| **Blue Ocean** | Lists (4 actions) | 5 items/action | 2 lines/item |
| **OKRs** | Nested lists | 5 objectives | 4 KRs/objective |
| **BSC** | Lists (4 perspectives) | 6 items/perspective | 2 lines/item |
| **Benchmarking** | Lists (3 sections) | 5 items/section | 1-2 lines/item |
| **Scenarios** | Paragraphs (3 scenarios) | 1 paragraph/scenario | 150 words max |
| **Growth** | Lists (3 sections) | 5 items/section | 1-2 lines/item |
| **Risk** | Mixed | 2 risk boxes + signals | Varies |
| **Roadmap** | Lists + paragraph | 5 alternatives + recommendation | Varies |

### CSS Line Clamping

Templates use `-webkit-line-clamp` for overflow control:

```css
.item {
    display: -webkit-box;
    -webkit-line-clamp: 2;       /* Max 2 lines */
    -webkit-box-orient: vertical;
    overflow: hidden;
}
```

---

## Go Template Variables

### Global Context (Available to All Templates)
```go
.CompanyName  // string
.Industry     // string
.Market       // string (default: "Brazil")
.Date         // string (formatted: "January 2006")
.Version      // string (default: "1.0")
```

### Framework-Specific Data

#### PESTEL
```go
.PESTEL.Political      // []string
.PESTEL.Economic       // []string
.PESTEL.Social         // []string
.PESTEL.Technological  // []string
.PESTEL.Environmental  // []string
.PESTEL.Legal          // []string
.PESTEL.Summary        // string
```

#### Porter
```go
.Porter.CompetitiveRivalry  // string
.Porter.ThreatNewEntrants   // string
.Porter.ThreatSubstitutes   // string
.Porter.SupplierPower       // string
.Porter.BuyerPower          // string
.Porter.OverallAttractiveness // string
```

#### SWOT
```go
.SWOT.Strengths      // []string
.SWOT.Weaknesses     // []string
.SWOT.Opportunities  // []string
.SWOT.Threats        // []string
```

#### TAM-SAM-SOM
```go
.TamSamSom.TAM          // string
.TamSamSom.SAM          // string
.TamSamSom.SOM          // string
.TamSamSom.Assumptions  // []string
.TamSamSom.CAGR         // string
```

#### Blue Ocean
```go
.BlueOcean.Eliminate      // []string
.BlueOcean.Reduce         // []string
.BlueOcean.Raise          // []string
.BlueOcean.Create         // []string
.BlueOcean.NewValueCurve  // string
```

#### OKRs
```go
.OKRs.Objectives  // []OKRObjective
    .Title        // string
    .KeyResults   // []string
```

#### Balanced Scorecard
```go
.BSC.Financial      // []string
.BSC.Customer       // []string
.BSC.Internal       // []string
.BSC.LearningGrowth // []string
```

#### Benchmarking
```go
.Benchmarking.CompetitorsAnalyzed  // []string
.Benchmarking.PerformanceGaps      // []string
.Benchmarking.BestPractices        // []string
```

#### Scenarios (Financial Projections + Risk Assessment)
```go
.Scenarios.ScenarioOptimistic     // string
.Scenarios.ScenarioRealist        // string
.Scenarios.ScenarioPessimistic    // string
.Scenarios.EarlyWarningSignals    // []string
```

#### Growth Hacking
```go
.GrowthHacking.Hypotheses   // []string
.GrowthHacking.Experiments  // []string
.GrowthHacking.KeyMetrics   // []string
```

#### Decision Matrix
```go
.DecisionMatrix.Alternatives        // []string
.DecisionMatrix.Criteria            // []string
.DecisionMatrix.FinalRecommendation // string
```

#### Synthesis (Executive Summary)
```go
.Synthesis.ExecutiveSummary      // string
.Synthesis.KeyFindings           // []string
.Synthesis.StrategicPriorities   // []string
.Synthesis.Roadmap               // []string
.Synthesis.OverallRecommendation // string
```

---

## Testing Checklist

### Visual Testing
- [ ] All pages render at exactly 842px height
- [ ] Content fits within safe area (40px margins)
- [ ] No text overflow or clipping
- [ ] Colors match brand guidelines
- [ ] Typography is crisp and readable at 10px

### PDF Generation Testing
```bash
# Test individual template
curl -X POST http://localhost:3000/forms/chromium/convert/html \
  -F 'files=@01_cover.html' \
  -F 'paperWidth=8.27' \
  -F 'paperHeight=11.7' \
  -F 'printBackground=true' \
  -o test_cover.pdf

# Test full report
# Use your report service's test suite
go test ./domain/report -v -run TestGenerateReport
```

### Content Testing
- [ ] Test with minimal data (1-2 items per list)
- [ ] Test with maximum data (5-6 items per list)
- [ ] Test with long text strings (check truncation)
- [ ] Test with special characters (accents, quotes)
- [ ] Test with missing/nil data fields

---

## Migration Path

### Option A: Gradual Rollout
1. Keep both `report/` and `report_v2/` directories
2. Add feature flag: `USE_REPORT_V2=true`
3. Test v2 on non-production reports
4. Switch production once validated

### Option B: Direct Replacement
1. Backup current templates: `mv templates/report templates/report_v1_backup`
2. Deploy v2: `mv templates/report_v2 templates/report`
3. Update Gotenberg config
4. Deploy and monitor

---

## Troubleshooting

### Issue: Content Overflows Page
**Solution**: Review content length, reduce font size, or increase line-clamp

### Issue: PDF Renders Blank
**Solution**: Check Gotenberg logs, verify HTML is valid, ensure CSS is embedded

### Issue: Colors Look Washed Out
**Solution**: Add `-webkit-print-color-adjust: exact;` to body CSS

### Issue: Fonts Don't Match
**Solution**: Verify system font stack is available on server

### Issue: Page Breaks Mid-Content
**Solution**: Add `page-break-inside: avoid;` to container divs

---

## Performance Notes

- **Template parsing**: ~5-10ms per template
- **PDF generation**: ~2-3 seconds for 16-page report
- **File size**: ~200-400KB per PDF (depends on content)
- **Gotenberg timeout**: 60 seconds (configurable)

---

## Future Enhancements

### Planned Features
- [ ] Dynamic page numbers in footer
- [ ] Custom color themes per client
- [ ] Multi-language support (English/Portuguese)
- [ ] Interactive PDF elements (links, bookmarks)
- [ ] Chart/graph generation for financial data
- [ ] Client logo integration on cover

### Optimization Opportunities
- [ ] Template caching in production
- [ ] Parallel PDF generation for speed
- [ ] Compression for smaller file sizes
- [ ] Pre-rendered static pages for common sections

---

## Support

For questions or issues with Report V2 templates:
- **Technical docs**: See `backend_v3/docs/`
- **Bug reports**: Create GitHub issue with `report-v2` label
- **Design feedback**: Contact design team

---

**Version**: 2.0.0
**Last Updated**: November 2025
**Maintained by**: IMENSIAH Engineering Team
