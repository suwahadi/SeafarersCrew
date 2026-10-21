export interface NavItem {
  to: string
  label: string
}

export interface NavGroup {
  label: string
  items: NavItem[]
}

export const navigation: NavGroup[] = [
  {
    label: 'Crew Operations',
    items: [
      { to: '/', label: 'Dashboard' },
      { to: '/crews', label: 'Crew' },
      { to: '/contracts', label: 'Contracts' },
      { to: '/rotation', label: 'Rotation Planning' },
      { to: '/quotations', label: 'Quotations' },
      { to: '/documents', label: 'Generated Documents' },
      { to: '/email-logs', label: 'Email Log' },
    ],
  },
  {
    label: 'Master Data',
    items: [
      { to: '/masters/companies', label: 'Companies & Principals' },
      { to: '/masters/vessels', label: 'Vessels' },
      { to: '/masters/ranks', label: 'Ranks' },
      { to: '/masters/vessel-types', label: 'Vessel Types' },
      { to: '/masters/currencies', label: 'Currencies' },
      { to: '/masters/crew-statuses', label: 'Crew Statuses' },
      { to: '/masters/allowance-types', label: 'Allowance Types' },
      { to: '/masters/document-types', label: 'Document Types' },
      { to: '/masters/certificate-types', label: 'Certificate Types' },
      { to: '/compliance-matrix', label: 'Compliance Matrix' },
    ],
  },
  {
    label: 'Settings',
    items: [
      { to: '/settings', label: 'Settings' },
      { to: '/document-templates', label: 'Document Templates' },
      { to: '/email-templates', label: 'Email Templates' },
      { to: '/account', label: 'Account' },
    ],
  },
]
