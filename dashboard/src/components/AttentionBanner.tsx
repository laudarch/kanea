import { ChevronRight } from 'lucide-react'
import { Link } from '@/lib/router'
import { cn } from '@/lib/utils'

const tones = {
  warn: {
    row: 'border-status-warn/40 bg-status-warn/10 text-status-warn',
    pill: 'border-status-warn/50 bg-status-warn/15',
  },
  error: {
    row: 'border-status-error/40 bg-status-error/10 text-status-error',
    pill: 'border-status-error/50 bg-status-error/15',
  },
} as const

/**
 * AttentionBanner is the dashboard's needs-looking-at row: a labelled pill,
 * the subject as a link, one line of why, and a chevron. The whole row is
 * the link, because the only thing to do with a warning is go look at it.
 */
export function AttentionBanner({
  tone,
  label,
  to,
  subject,
  children,
}: {
  tone: keyof typeof tones
  label: string
  to: string
  subject: string
  children: React.ReactNode
}) {
  return (
    <Link
      to={to}
      className={cn(
        'flex items-center gap-3 rounded-lg border px-3 py-2 text-sm transition-colors hover:brightness-110',
        tones[tone].row,
      )}
    >
      <span
        className={cn(
          'shrink-0 rounded-full border px-2 py-0.5 text-xs font-medium',
          tones[tone].pill,
        )}
      >
        {label}
      </span>
      <span className="min-w-0 truncate">
        <span className="font-mono font-medium underline underline-offset-2">{subject}</span>
        <span> — {children}</span>
      </span>
      <ChevronRight size={15} aria-hidden className="ml-auto shrink-0" />
    </Link>
  )
}
