import type { ReactNode } from 'react'
import type { MasterOptions, MasterRecord, MasterSegment } from '@/api/masters'
import { Badge } from '@/components/ui/Layout'
import type { Option } from '@/components/ui/Field'
import { titleCase } from '@/lib/format'

export type FieldType = 'text' | 'textarea' | 'email' | 'url' | 'number' | 'decimal' | 'color' | 'checkbox' | 'select' | 'imo' | 'logo' | 'reference'

export interface FieldDef {
  name: string
  label: string
  type: FieldType
  required?: boolean
  maxLength?: number
  min?: number
  max?: number
  hint?: string
  options?: Option[]
  source?: (o: MasterOptions) => Option[]
  upper?: boolean
  createOnly?: boolean
  defaultValue?: string | boolean
}

export interface ColumnDefSpec {
  id: string
  header: string
  sortable: boolean
  render: (row: MasterRecord) => ReactNode
}

export interface FilterSpec {
  name: string
  label: string
  options?: Option[]
  source?: (o: MasterOptions) => Option[]
}

export interface MasterConfig {
  segment: MasterSegment
  title: string
  singular: string
  description: string
  columns: ColumnDefSpec[]
  fields: FieldDef[]
  filters: FilterSpec[]
  readonlyOnSystem?: boolean
}

const opts = (values: readonly string[], label: (v: string) => string = titleCase): Option[] => values.map((v) => ({ value: v, label: label(v) }))

export const companyTypes = opts(['company', 'principal', 'both'])
export const departments = opts(['deck', 'engine', 'catering_other'], (v) => (v === 'catering_other' ? 'Catering/Other' : titleCase(v)))
export const vesselStatuses = opts(['active', 'laid_up', 'sold'])
export const allowanceBases: Option[] = [
  { value: 'fixed_month', label: 'Fixed per month' },
  { value: 'fixed_day', label: 'Fixed per day' },
  { value: 'percent_basic', label: 'Percentage of basic' },
]
export const certificateCategories: Option[] = [
  { value: 'coc', label: 'COC' },
  { value: 'cop', label: 'COP' },
  { value: 'endorsement', label: 'Endorsement' },
  { value: 'other', label: 'Other' },
]
const yesNo: Option[] = [
  { value: 'true', label: 'Yes' },
  { value: 'false', label: 'No' },
]

function text(row: MasterRecord, key: string): string {
  const v = row[key]
  return typeof v === 'string' || typeof v === 'number' ? String(v) : ''
}

function labelOf(options: Option[], value: string): string {
  return options.find((o) => o.value === value)?.label ?? value
}

const byName = (items: { id: string; name: string }[]): Option[] => items.map((i) => ({ value: i.id, label: i.name }))

const principals = (o: MasterOptions) => byName(o.companies.filter((c) => c.type !== 'company'))
const owners = (o: MasterOptions) => byName(o.companies.filter((c) => c.type !== 'principal'))
const vesselTypes = (o: MasterOptions) => byName(o.vessel_types)

const col = (id: string, header: string, sortable = true, render?: (row: MasterRecord) => ReactNode): ColumnDefSpec => ({
  id,
  header,
  sortable,
  render: render ?? ((row) => text(row, id) || '—'),
})

const yesNoCell = (key: string) => (row: MasterRecord) => (row[key] === true ? 'Yes' : 'No')

export const masterConfigs: Record<MasterSegment, MasterConfig> = {
  companies: {
    segment: 'companies',
    title: 'Companies & Principals',
    singular: 'Company',
    description: 'Employers, ship owners and principals used in vessels, contracts and quotations.',
    columns: [
      col('code', 'Code'),
      col('name', 'Name'),
      col('type', 'Type', true, (r) => labelOf(companyTypes, text(r, 'type'))),
      col('country', 'Country'),
      col('email', 'Email', false),
      col('phone', 'Phone', false),
    ],
    filters: [{ name: 'type', label: 'Type', options: companyTypes }],
    fields: [
      { name: 'code', label: 'Code', type: 'text', required: true, maxLength: 20, upper: true },
      { name: 'name', label: 'Name', type: 'text', required: true, maxLength: 200 },
      { name: 'type', label: 'Type', type: 'select', required: true, options: companyTypes, defaultValue: 'company' },
      { name: 'country', label: 'Country', type: 'text', maxLength: 80 },
      { name: 'address', label: 'Full address', type: 'textarea', maxLength: 500 },
      { name: 'phone', label: 'Phone', type: 'text', maxLength: 40 },
      { name: 'email', label: 'Email', type: 'email' },
      { name: 'website', label: 'Website', type: 'url', hint: 'Including https://' },
      { name: 'logo_file_id', label: 'Logo', type: 'logo', hint: 'Converted to WebP; SVG is sanitized.' },
    ],
  },
  vessels: {
    segment: 'vessels',
    title: 'Vessels',
    singular: 'Vessel',
    description: 'Fleet register with IMO numbers, owners and principals.',
    columns: [
      col('name', 'Name'),
      col('imo_number', 'IMO'),
      col('flag', 'Flag'),
      col('grt', 'GRT'),
      col('year_built', 'Built'),
      col('status', 'Status', true, (r) => labelOf(vesselStatuses, text(r, 'status'))),
    ],
    filters: [
      { name: 'status', label: 'Status', options: vesselStatuses },
      { name: 'vessel_type_id', label: 'Vessel type', source: vesselTypes },
      { name: 'principal_id', label: 'Principal', source: principals },
    ],
    fields: [
      { name: 'name', label: 'Name', type: 'text', required: true, maxLength: 120 },
      { name: 'imo_number', label: 'IMO number', type: 'imo', required: true, hint: '7 digits with a valid check digit.' },
      { name: 'flag', label: 'Flag', type: 'text', maxLength: 80 },
      { name: 'vessel_type_id', label: 'Vessel type', type: 'reference', source: vesselTypes },
      { name: 'grt', label: 'GRT', type: 'decimal' },
      { name: 'hp_kw', label: 'HP / kW', type: 'text', maxLength: 40, hint: 'e.g. 8000 HP' },
      { name: 'year_built', label: 'Year built', type: 'number', min: 1900, max: 2100 },
      { name: 'owner_company_id', label: 'Owner company', type: 'reference', source: owners },
      { name: 'principal_id', label: 'Principal', type: 'reference', source: principals },
      { name: 'status', label: 'Status', type: 'select', required: true, options: vesselStatuses, defaultValue: 'active' },
    ],
  },
  ranks: {
    segment: 'ranks',
    title: 'Ranks',
    singular: 'Rank',
    description: 'Ranks grouped by department; the order here is used in every dropdown.',
    columns: [
      col('name', 'Name'),
      col('short_code', 'Code'),
      col('department', 'Department', true, (r) => labelOf(departments, text(r, 'department'))),
      col('is_officer', 'Officer', false, yesNoCell('is_officer')),
    ],
    filters: [
      { name: 'department', label: 'Department', options: departments },
      { name: 'is_officer', label: 'Officer', options: yesNo },
    ],
    fields: [
      { name: 'name', label: 'Name', type: 'text', required: true, maxLength: 80 },
      { name: 'short_code', label: 'Short code', type: 'text', required: true, maxLength: 10, upper: true, hint: 'e.g. CE, 2O' },
      { name: 'department', label: 'Department', type: 'select', required: true, options: departments, defaultValue: 'deck' },
      { name: 'is_officer', label: 'Officer rank', type: 'checkbox', defaultValue: false },
    ],
  },
  'vessel-types': {
    segment: 'vessel-types',
    title: 'Vessel Types',
    singular: 'Vessel type',
    description: 'AHTS, crew boats, tugs and other vessel categories.',
    columns: [col('name', 'Name')],
    filters: [],
    fields: [{ name: 'name', label: 'Name', type: 'text', required: true, maxLength: 80 }],
  },
  currencies: {
    segment: 'currencies',
    title: 'Currencies',
    singular: 'Currency',
    description: 'ISO 4217 currencies with their display decimals. Amounts are never converted.',
    columns: [col('code', 'Code'), col('name', 'Name'), col('symbol', 'Symbol', false), col('decimals', 'Decimals')],
    filters: [],
    fields: [
      { name: 'code', label: 'ISO code', type: 'text', required: true, maxLength: 3, upper: true, createOnly: true },
      { name: 'name', label: 'Name', type: 'text', maxLength: 80 },
      { name: 'symbol', label: 'Symbol', type: 'text', maxLength: 8 },
      { name: 'decimals', label: 'Decimal places', type: 'number', required: true, min: 0, max: 4, defaultValue: '2' },
    ],
  },
  'crew-statuses': {
    segment: 'crew-statuses',
    title: 'Crew Statuses',
    singular: 'Crew status',
    description: 'System statuses drive business rules and cannot be deleted.',
    readonlyOnSystem: true,
    columns: [
      col('name', 'Name', true, (r) => <Badge color={text(r, 'color')}>{text(r, 'name')}</Badge>),
      col('code', 'Code', false),
      col('is_system', 'System', true, yesNoCell('is_system')),
    ],
    filters: [{ name: 'is_system', label: 'System', options: yesNo }],
    fields: [
      { name: 'name', label: 'Name', type: 'text', required: true, maxLength: 60 },
      { name: 'color', label: 'Color', type: 'color', required: true, defaultValue: '#64748B' },
    ],
  },
  'allowance-types': {
    segment: 'allowance-types',
    title: 'Allowance Types',
    singular: 'Allowance type',
    description: 'Allowances added to contracts with their calculation basis.',
    columns: [col('name', 'Name'), col('basis', 'Basis', true, (r) => labelOf(allowanceBases, text(r, 'basis')))],
    filters: [{ name: 'basis', label: 'Basis', options: allowanceBases }],
    fields: [
      { name: 'name', label: 'Name', type: 'text', required: true, maxLength: 80 },
      { name: 'basis', label: 'Calculation basis', type: 'select', required: true, options: allowanceBases, defaultValue: 'fixed_month' },
    ],
  },
  'document-types': {
    segment: 'document-types',
    title: 'Document Types',
    singular: 'Document type',
    description: 'Travel documents such as passports, seaman books and medicals.',
    columns: [col('name', 'Name'), col('code', 'Code', false), col('has_expiry', 'Has expiry', true, yesNoCell('has_expiry')), col('number_hint', 'Number hint', false)],
    filters: [{ name: 'has_expiry', label: 'Has expiry', options: yesNo }],
    fields: [
      { name: 'name', label: 'Name', type: 'text', required: true, maxLength: 80 },
      { name: 'has_expiry', label: 'Has an expiry date', type: 'checkbox', defaultValue: true },
      { name: 'number_hint', label: 'Number format hint', type: 'text', maxLength: 80 },
    ],
  },
  'certificate-types': {
    segment: 'certificate-types',
    title: 'Certificate Types',
    singular: 'Certificate type',
    description: 'STCW certificates of competency and proficiency.',
    columns: [
      col('name', 'Name'),
      col('stcw_reference', 'STCW'),
      col('category', 'Category', true, (r) => labelOf(certificateCategories, text(r, 'category'))),
      col('has_expiry', 'Has expiry', true, yesNoCell('has_expiry')),
    ],
    filters: [
      { name: 'category', label: 'Category', options: certificateCategories },
      { name: 'has_expiry', label: 'Has expiry', options: yesNo },
    ],
    fields: [
      { name: 'name', label: 'Name', type: 'text', required: true, maxLength: 160 },
      { name: 'stcw_reference', label: 'STCW reference', type: 'text', maxLength: 20, hint: 'e.g. VI/1' },
      { name: 'category', label: 'Category', type: 'select', required: true, options: certificateCategories, defaultValue: 'cop' },
      { name: 'has_expiry', label: 'Has an expiry date', type: 'checkbox', defaultValue: true },
    ],
  },
}

export function isMasterSegment(value: string | undefined): value is MasterSegment {
  return value !== undefined && value in masterConfigs
}
