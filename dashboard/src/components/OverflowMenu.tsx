import { useEffect, useRef, useState } from 'react'
import { MoreHorizontal } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

/** The shared item style for an OverflowMenu entry. */
export const overflowItem =
  'flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-sm ' +
  'text-foreground transition-colors hover:bg-muted disabled:opacity-50'

/**
 * OverflowMenu is the mockup's ⋯ button: rarer verbs folded behind one
 * quiet control. Hand-rolled disclosure (OpenUrlMenu's reasoning: a menu
 * primitive is a dependency, and two poppers is not a system). The panel
 * does NOT close on item click - an item that arms a confirm needs to stay
 * visible - so items that finish something call the `close` they receive.
 */
export function OverflowMenu({
  label = 'More actions',
  children,
}: {
  label?: string
  children: (close: () => void) => React.ReactNode
}) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  return (
    <div className="relative inline-block" ref={ref}>
      <Button
        size="sm"
        variant="outline"
        aria-label={label}
        aria-expanded={open}
        className={cn('h-7 px-2', open ? 'bg-muted' : '')}
        onClick={() => setOpen((o) => !o)}
      >
        <MoreHorizontal size={14} />
      </Button>
      {open ? (
        <div className="absolute right-0 z-20 mt-1 w-52 rounded-lg border border-border bg-card p-1 text-left shadow-lg">
          {children(() => setOpen(false))}
        </div>
      ) : null}
    </div>
  )
}
