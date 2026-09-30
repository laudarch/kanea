import { cn } from '@/lib/utils'

/**
 * FilterChips is the row of status buttons every list page filters by,
 * grouped in one bordered pill the mockup's way: the active chip solid,
 * the rest quiet, each carrying its count so the filter answers "how many
 * would I see" before it is pressed.
 */
export function FilterChips<F extends string>({
  options,
  value,
  onChange,
}: {
  options: readonly { value: F; label: string; count?: number | undefined }[]
  value: F
  onChange: (value: F) => void
}) {
  return (
    <div className="inline-flex flex-wrap items-center gap-0.5 rounded-lg border border-border bg-card p-1">
      {options.map((option) => {
        const active = value === option.value
        return (
          <button
            key={option.value}
            type="button"
            onClick={() => onChange(option.value)}
            className={cn(
              'flex items-center gap-1.5 rounded-md px-3 py-1 text-xs transition-colors',
              active
                ? 'bg-muted font-medium text-foreground'
                : 'text-muted-foreground hover:text-foreground',
            )}
          >
            {option.label}
            {option.count !== undefined ? (
              <span
                className={cn(
                  'font-mono text-[11px] tabular-nums',
                  active ? 'text-muted-foreground' : 'text-muted-foreground/60',
                )}
              >
                {option.count}
              </span>
            ) : null}
          </button>
        )
      })}
    </div>
  )
}
