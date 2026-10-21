import { lazy } from 'react'
import type { RouteObject } from 'react-router'
import { OptionsReady } from '@/components/forms/OptionsReady'

const VacancyListPage = lazy(() => import('./VacancyListPage'))
const VacancyFormPage = lazy(() => import('./VacancyFormPage'))

export const vacancyRoutes: RouteObject[] = [
  { path: 'vacancies', element: <OptionsReady><VacancyListPage /></OptionsReady> },
  { path: 'vacancies/new', element: <OptionsReady><VacancyFormPage /></OptionsReady> },
  { path: 'vacancies/:id', element: <OptionsReady><VacancyFormPage /></OptionsReady> },
]
