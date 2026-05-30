import { useState, useCallback, createContext, useContext, type ReactNode } from 'react'
import { CheckCircle2, XCircle, X } from 'lucide-react'

type ToastType = 'success' | 'error'

interface Toast {
  id: number
  type: ToastType
  title: string
  message?: string
}

interface ToastCtx {
  toast: (t: Omit<Toast, 'id'>) => void
}

const Ctx = createContext<ToastCtx>({ toast: () => {} })

export function useToast() {
  return useContext(Ctx)
}

let nextId = 0

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const addToast = useCallback((t: Omit<Toast, 'id'>) => {
    const id = nextId++
    setToasts(prev => [...prev, { ...t, id }])
    setTimeout(() => {
      setToasts(prev => prev.filter(x => x.id !== id))
    }, 4000)
  }, [])

  const remove = (id: number) => setToasts(prev => prev.filter(x => x.id !== id))

  return (
    <Ctx.Provider value={{ toast: addToast }}>
      {children}
      {/* Toast container */}
      <div className="fixed bottom-6 right-6 z-50 flex flex-col gap-2 max-w-sm">
        {toasts.map(t => (
          <div
            key={t.id}
            className={`flex items-start gap-3 px-4 py-3 rounded-xl border shadow-dialog bg-bg-card transition-all animate-in slide-in-from-right-2
              ${t.type === 'success' ? 'border-green-200/60' : 'border-red-200/60'}`}
          >
            {t.type === 'success'
              ? <CheckCircle2 className="w-5 h-5 text-green-600 shrink-0 mt-0.5" />
              : <XCircle className="w-5 h-5 text-red-500 shrink-0 mt-0.5" />
            }
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium text-fg-base">{t.title}</p>
              {t.message && <p className="text-xs text-fg-muted mt-0.5">{t.message}</p>}
            </div>
            <button onClick={() => remove(t.id)} className="p-0.5 rounded hover:bg-sidebar-hover text-fg-subtle hover:text-fg-base transition-colors shrink-0">
              <X className="w-3.5 h-3.5" />
            </button>
          </div>
        ))}
      </div>
    </Ctx.Provider>
  )
}
