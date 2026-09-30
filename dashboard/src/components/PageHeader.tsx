export interface PageHeaderProps {
  title: React.ReactNode
  /** subtitle sits under the title: the page's one-line summary, sans-serif,
   * muted, with inline status dots where a count needs a tone. */
  subtitle?: React.ReactNode | undefined
  /** back renders above the title: the "← Services" chip on detail pages. */
  back?: React.ReactNode | undefined
  /** meta is the right edge's quiet facts ("up 8m · refreshed 5s"): mono,
   * muted, never a control. */
  meta?: React.ReactNode | undefined
  actions?: React.ReactNode | undefined
}

/** PageHeader is every page's first block: title over subtitle, with the
 * page's actions (and quiet meta) on the right edge. */
export function PageHeader({ title, subtitle, back, meta, actions }: PageHeaderProps) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div className="min-w-0">
        {back}
        <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
        {subtitle !== undefined ? (
          <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground">
            {subtitle}
          </div>
        ) : null}
      </div>
      {meta !== undefined || actions !== undefined ? (
        <div className="flex items-center gap-3 pt-1.5">
          {meta !== undefined ? (
            <span className="font-mono text-xs text-muted-foreground">{meta}</span>
          ) : null}
          {actions !== undefined ? <div className="flex items-center gap-2">{actions}</div> : null}
        </div>
      ) : null}
    </div>
  )
}
