import { spawnSync } from 'node:child_process'
import { mkdirSync, writeFileSync } from 'node:fs'

const api = process.env.API_URL ?? 'http://localhost:8080'
const site = process.env.SITE_URL ?? 'http://localhost:3000'
const threshold = Number(process.env.PERFORMANCE_THRESHOLD ?? '0.85')
const runs = Number(process.env.LH_RUNS ?? '3')

async function firstSlug(path) {
  const res = await fetch(`${api}/api/public/v1${path}`)
  if (!res.ok) throw new Error(`${path} answered ${res.status}`)
  const body = await res.json()
  const slug = body.data?.[0]?.slug
  if (!slug) throw new Error(`${path} returned no items; load the demo seed first`)
  return slug
}

async function firstCmsPage() {
  const res = await fetch(`${api}/api/public/v1/sitemap-data`)
  if (!res.ok) throw new Error(`sitemap-data answered ${res.status}`)
  const body = await res.json()
  const page = body.data?.pages?.[0]
  if (!page) throw new Error('No published CMS page found; load the demo seed first')
  return page.path
}

const urls = [
  `${site}/`,
  `${site}${await firstCmsPage()}`,
  `${site}/news/${await firstSlug('/news')}`,
  `${site}/career`,
  `${site}/contact`,
]

let failed = false
for (const form of ['mobile', 'desktop']) {
  const outputDir = `.lighthouseci/${form}`
  mkdirSync(outputDir, { recursive: true })
  const config = {
    ci: {
      collect: {
        url: urls,
        numberOfRuns: runs,
        settings: { ...(form === 'desktop' ? { preset: 'desktop' } : {}), chromeFlags: '--no-sandbox --headless=new' },
      },
      assert: {
        assertions: {
          'categories:performance': ['error', { minScore: threshold, aggregationMethod: 'median-run' }],
          'categories:accessibility': ['warn', { minScore: 0.9, aggregationMethod: 'median-run' }],
          'categories:seo': ['warn', { minScore: 0.9, aggregationMethod: 'median-run' }],
          'categories:best-practices': ['warn', { minScore: 0.9, aggregationMethod: 'median-run' }],
        },
      },
      upload: { target: 'filesystem', outputDir },
    },
  }
  const configPath = `.lighthouseci/rc-${form}.json`
  writeFileSync(configPath, JSON.stringify(config, null, 2))
  const result = spawnSync('npx', ['--yes', '@lhci/cli@0.15.1', 'autorun', `--config=${configPath}`], { stdio: 'inherit' })
  if (result.status !== 0) failed = true
}
process.exit(failed ? 1 : 0)
