import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useState } from 'react'
import { useBlocker } from 'react-router'
import { ApiError, errorMessage } from '@/api/client'
import { fetchAllPages, menusApi, MENUS_KEY, PAGE_OPTIONS_KEY, type MenuName } from '@/api/cms'
import { Button } from '@/components/ui/Button'
import { PageHeader } from '@/components/ui/Layout'
import { ConfirmDialog } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Tabs } from '@/components/ui/Tabs'
import { useToast } from '@/components/ui/Toast'
import { isValidLink } from '@/lib/links'
import { appendNode, fromServer, mapServerErrors, MENU_MAX_ITEMS, payloadSignature, toPayload, validateTree, type MenuErrors, type MenuNode } from '@/lib/menuTree'
import { AddMenuItem } from './AddMenuItem'
import { MenuTreeEditor } from './MenuTreeEditor'

type Drafts = Record<MenuName, MenuNode[] | null>
type ErrorState = Record<MenuName, MenuErrors>

const EMPTY_ERRORS: MenuErrors = { byKey: {}, general: [] }

export default function MenusPage() {
  const qc = useQueryClient()
  const toast = useToast()
  const menus = useQuery({ queryKey: MENUS_KEY, queryFn: ({ signal }) => menusApi.get(signal) })
  const pages = useQuery({ queryKey: PAGE_OPTIONS_KEY, queryFn: ({ signal }) => fetchAllPages(signal) })
  const [active, setActive] = useState<MenuName>('header')
  const [drafts, setDrafts] = useState<Drafts>({ header: null, footer: null })
  const [errors, setErrors] = useState<ErrorState>({ header: EMPTY_ERRORS, footer: EMPTY_ERRORS })

  const saved = useMemo(() => (menus.data ? { header: fromServer(menus.data.header), footer: fromServer(menus.data.footer) } : null), [menus.data])
  const current = (name: MenuName): MenuNode[] => drafts[name] ?? saved?.[name] ?? []
  const isDirty = (name: MenuName): boolean => {
    const draft = drafts[name]
    return draft !== null && saved !== null && payloadSignature(draft) !== payloadSignature(saved[name])
  }
  const anyDirty = isDirty('header') || isDirty('footer')

  const blocker = useBlocker(({ currentLocation, nextLocation }) => anyDirty && currentLocation.pathname !== nextLocation.pathname)

  useEffect(() => {
    if (!anyDirty) return
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault()
    }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [anyDirty])

  const setTree = (name: MenuName, tree: MenuNode[]) => {
    setDrafts((d) => ({ ...d, [name]: tree }))
    setErrors((e) => ({ ...e, [name]: EMPTY_ERRORS }))
  }

  const save = useMutation({
    mutationFn: ({ name, tree }: { name: MenuName; tree: MenuNode[] }) => menusApi.replace(name, toPayload(tree).items),
    onSuccess: (response, { name }) => {
      qc.setQueryData(MENUS_KEY, response)
      setDrafts((d) => ({ ...d, [name]: null }))
      setErrors((e) => ({ ...e, [name]: EMPTY_ERRORS }))
      toast.success(`${name === 'header' ? 'Header' : 'Footer'} menu saved.`)
    },
    onError: (err, { name, tree }) => {
      if (err instanceof ApiError && Object.keys(err.fields).length > 0) {
        const mapped = mapServerErrors(tree, err.fields)
        setErrors((e) => ({ ...e, [name]: { byKey: mapped.byKey, general: [err.message, ...mapped.general] } }))
        return
      }
      setErrors((e) => ({ ...e, [name]: { byKey: {}, general: [errorMessage(err)] } }))
    },
  })

  const submit = () => {
    const tree = current(active)
    const invalid = validateTree(tree, isValidLink)
    if (Object.keys(invalid).length > 0) {
      setErrors((e) => ({ ...e, [active]: { byKey: invalid, general: ['Fix the highlighted items before saving.'] } }))
      return
    }
    save.mutate({ name: active, tree })
  }

  const tabs = [
    { id: 'header', label: 'Header', dirty: isDirty('header') },
    { id: 'footer', label: 'Footer', dirty: isDirty('footer') },
  ]
  const tree = current(active)
  const activeErrors = errors[active]

  return (
    <div>
      <PageHeader
        title="Menus"
        description="Build the header and footer navigation. Menus have up to two levels."
        actions={
          <>
            <Button
              variant="secondary"
              disabled={!isDirty(active) || save.isPending}
              onClick={() => {
                setDrafts((d) => ({ ...d, [active]: null }))
                setErrors((e) => ({ ...e, [active]: EMPTY_ERRORS }))
              }}
            >
              Discard changes
            </Button>
            <Button onClick={submit} loading={save.isPending} disabled={!isDirty(active) || menus.isPending}>
              Save {active} menu
            </Button>
          </>
        }
      />
      <Tabs tabs={tabs} active={active} onChange={(id) => setActive(id === 'footer' ? 'footer' : 'header')} />
      <div className="mt-4 space-y-6">
        {menus.isPending ? (
          <div className="flex justify-center p-8">
            <Spinner />
          </div>
        ) : menus.isError ? (
          <p role="alert" className="text-sm text-red-700">
            {errorMessage(menus.error)}
          </p>
        ) : (
          <>
            {activeErrors.general.length > 0 ? (
              <div role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
                {activeErrors.general.map((m) => (
                  <p key={m}>{m}</p>
                ))}
              </div>
            ) : null}
            <MenuTreeEditor tree={tree} errors={activeErrors.byKey} onChange={(next) => setTree(active, next)} />
            <AddMenuItem
              pages={pages.data ?? []}
              pagesLoading={pages.isPending}
              disabled={tree.length >= MENU_MAX_ITEMS}
              onAdd={(node) => setTree(active, appendNode(tree, node))}
            />
          </>
        )}
      </div>
      <ConfirmDialog
        open={blocker.state === 'blocked'}
        title="Discard unsaved menu changes?"
        message="Menu changes have not been saved. Leaving this page will discard them."
        confirmLabel="Discard and leave"
        danger
        onConfirm={() => blocker.state === 'blocked' && blocker.proceed()}
        onClose={() => blocker.state === 'blocked' && blocker.reset()}
      />
    </div>
  )
}
