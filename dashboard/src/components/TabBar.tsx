import { cn } from '@/lib/utils'

export interface Tab<T extends string> {
  value: T
  label: string
  count?: number | undefined
}

/**
 * TabBar is the mockup's section switcher: quiet labels on a hairline, the
 * active one underlined in the accent. Counts ride as small pills so a tab
 * answers "is there anything in there" before it is opened.
 */
export function TabBar<T extends string>({
  tabs,
  value,
  onChange,
}: {
  tabs: readonly Tab<T>[]
  value: T
  onChange: (value: T) => void
}) {
  return (
    <div role="tablist" className="flex items-center gap-6 border-b border-border">
      {tabs.map((tab) => {
        const active = tab.value === value
        return (
          <button
            key={tab.value}
            role="tab"
            type="button"
            aria-selected={active}
            onClick={() => onChange(tab.value)}
            className={cn(
              '-mb-px flex items-center gap-1.5 border-b-2 px-0.5 pb-2 text-sm transition-colors',
              active
                ? 'border-primary font-medium text-foreground'
                : 'border-transparent text-muted-foreground hover:text-foreground',
            )}
          >
            {tab.label}
            {tab.count !== undefined ? (
              <span className="rounded-full bg-muted px-1.5 font-mono text-[11px] tabular-nums text-muted-foreground">
                {tab.count}
              </span>
            ) : null}
          </button>
        )
      })}
    </div>
  )
}
