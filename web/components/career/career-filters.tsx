import Link from 'next/link'
import { buttonClass } from '@/components/ui/button'
import { hasActiveFilters, type CareerFilters as Filters } from '@/lib/career'
import { departmentLabel } from '@/lib/format'
import type { VacancyFilters } from '@/lib/api'

interface CareerFiltersProps {
  filters: Filters
  options: VacancyFilters
}

const CONTROL =
  'block w-full rounded-md border border-slate-400 bg-white px-3 py-2.5 text-base text-ink focus-visible:border-accent-text'
const LABEL = 'mb-1.5 block text-sm font-semibold text-primary'

interface Option {
  value: string
  label: string
}

function SelectField({ name, label, value, options }: { name: string; label: string; value: string; options: Option[] }) {
  const known = options.some((option) => option.value === value)
  return (
    <div>
      <label htmlFor={`filter-${name}`} className={LABEL}>
        {label}
      </label>
      <select id={`filter-${name}`} name={name} defaultValue={known ? value : ''} className={CONTROL}>
        <option value="">All</option>
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  )
}

export function CareerFilters({ filters, options }: CareerFiltersProps) {
  return (
    <form method="get" action="/career" role="search" aria-label="Search vacancies" className="rounded-lg border border-line bg-surface p-4 sm:p-6">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div className="sm:col-span-2 lg:col-span-4">
          <label htmlFor="filter-q" className={LABEL}>
            Keyword
          </label>
          <input
            id="filter-q"
            name="q"
            type="search"
            defaultValue={filters.q}
            maxLength={100}
            placeholder="Job title, rank or Vacancy ID"
            className={CONTROL}
          />
        </div>
        <SelectField
          name="rank"
          label="Rank"
          value={filters.rank}
          options={options.ranks.map((rank) => ({ value: rank.id, label: rank.name }))}
        />
        <SelectField
          name="department"
          label="Department"
          value={filters.department}
          options={options.departments.map((department) => ({ value: department, label: departmentLabel(department) }))}
        />
        <SelectField
          name="vessel_type"
          label="Vessel type"
          value={filters.vessel_type}
          options={options.vessel_types.map((type) => ({ value: type.id, label: type.name }))}
        />
        <SelectField
          name="location"
          label="Location"
          value={filters.location}
          options={options.locations.map((location) => ({ value: location, label: location }))}
        />
      </div>
      <div className="mt-5 flex flex-wrap items-center gap-3">
        <button type="submit" className={buttonClass('primary')}>
          Search vacancies
        </button>
        {hasActiveFilters(filters) ? (
          <Link href="/career" className="text-sm font-semibold text-accent-text underline underline-offset-2">
            Clear filters
          </Link>
        ) : null}
      </div>
    </form>
  )
}
