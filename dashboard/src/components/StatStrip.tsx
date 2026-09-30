import { Card } from '@/components/ui/card'
import { cn } from '@/lib/utils'

export interface StatCell {
  label: React.ReactNode
  /** meta is the cell's top-right whisper ("8 cores", "of 16 GiB", "live"):
   * mono, muted, the value's denominator or provenance. */
  meta?: React.ReactNode | undefined
  value: React.ReactNode
  /** tone colours the big number; never the only signal - the label and sub
   * say why. */
  tone?: 'default' | 'primary' | 'ok' | 'error' | undefined
  /** sub sits beside the value: "all healthy", "8 running". */
  sub?: React.ReactNode | undefined
  /** chart renders full-width under the value: a sparkline, usually. */
  chart?: React.ReactNode | undefined
}

const valueTone = {
  default: 'text-foreground',
  primary: 'text-primary',
  ok: 'text-status-ok',
  error: 'text-status-error',
} as const

/**
 * StatStrip is the segmented number bar the mockup leads every page with:
 * one card, N cells split by hairlines, each cell a label, a big mono value
 * and optionally a sparkline. One card rather than N so the row reads as a
 * single instrument, not a shelf of boxes.
 */
export function StatStrip({ cells, className }: { cells: StatCell[]; className?: string }) {
  return (
    <Card className={cn('grid grid-cols-2 overflow-hidden lg:grid-cols-4', className)}>
      {cells.map((cell, i) => (
        <div
          key={i}
          className={cn(
            'min-w-0 px-4 py-3.5',
            // Hairlines between cells, drawn as left/top borders so the grid
            // wraps cleanly at two columns without doubled edges.
            i % 2 === 1 ? 'border-l border-border' : '',
            i >= 2 ? 'border-t border-border lg:border-t-0' : '',
            i > 0 ? 'lg:border-l lg:border-border' : '',
          )}
        >
          <div className="flex items-baseline justify-between gap-2">
            <span className="text-xs text-muted-foreground">{cell.label}</span>
            {cell.meta !== undefined ? (
              <span className="truncate font-mono text-[11px] text-muted-foreground/80">
                {cell.meta}
              </span>
            ) : null}
          </div>
          <div className="mt-1.5 flex items-baseline gap-2">
            <span
              className={cn(
                'font-mono text-[22px] font-medium tabular-nums',
                valueTone[cell.tone ?? 'default'],
              )}
            >
              {cell.value}
            </span>
            {cell.sub !== undefined ? (
              <span className="truncate text-xs text-muted-foreground">{cell.sub}</span>
            ) : null}
          </div>
          {cell.chart !== undefined ? <div className="mt-2.5">{cell.chart}</div> : null}
        </div>
      ))}
    </Card>
  )
}
