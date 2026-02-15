# 🔥 BuildBurn — CI/CD Cost Attribution Engine

Stop guessing where your CI budget goes. BuildBurn analyzes your GitHub Actions runs and tells you **exactly** which workflows, jobs, and steps are burning money — then shows you how to cut 30-50%.

## 🚀 Quick Start

```bash
# Install
go install github.com/buildburn-cli/buildburn@latest
# Or download binary from Releases

# Analyze last 7 days
export GITHUB_TOKEN=ghp_xxx
buildburn -repo your-org/your-repo

# JSON output for CI integration
buildburn -repo your-org/your-repo -days 30 -format json
```

## Example Output

```
🔥 BuildBurn Report (last 7 days)
══════════════════════════════════════════
  CI minutes: 4820 | Cost: $58.56 | Monthly: $250.97

📊 Cost by Workflow:
  CI Tests                              $32.40 (55%)
  Deploy Production                     $18.16 (31%)
  Nightly E2E                           $8.00  (14%)

🗑️  Waste:
  [retry    ] Failed: test-integration       $4.80
  [cache-miss] Restore npm cache             $1.20
  [slow-deps ] npm install (6.2m)            $0.02

💡 Suggestions:
  1. Fix 3 issues to save ~$6.02/week
  2. macOS runners cost 10x Linux — switch where possible
```

## 📊 Why Teams Pay for BuildBurn

| Pain | Impact |
|------|--------|
| CI bill jumped from $3K to $18K | No visibility into what changed |
| Cache misses silently double build time | Nobody monitors cache hit rates |
| macOS runners used for Linux-compatible jobs | 10x cost for no reason |
| Flaky tests retry 3x per PR | Each retry burns full job cost |

## 💰 Pricing

| Feature | Free | Pro $49/mo | Enterprise $499/mo |
|---------|------|------------|--------------------|
| Single repo analysis | ✅ | ✅ | ✅ |
| Multi-repo / org-wide | ❌ | ✅ (up to 20) | ✅ Unlimited |
| Cost-by-workflow breakdown | ✅ | ✅ | ✅ |
| Waste detection | ✅ (top 3) | ✅ Full | ✅ Full |
| Optimization suggestions | ✅ Basic | ✅ + auto-fix YAML | ✅ + custom rules |
| Slack/PR comment alerts | ❌ | ✅ | ✅ |
| Budget threshold alerts | ❌ | ✅ | ✅ |
| Trend analysis & forecasting | ❌ | ✅ | ✅ |
| Team/repo cost chargeback | ❌ | ❌ | ✅ |
| PDF/CSV export | ❌ | ✅ | ✅ |
| SSO / SAML | ❌ | ❌ | ✅ |
| SLA & support | Community | Email | Dedicated |

## License

BSL 1.1 — Free for teams < 10 devs. Commercial license required for larger teams.
