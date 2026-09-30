import { useQuery } from '@tanstack/react-query'
import {
  Activity,
  Boxes,
  DatabaseBackup,
  FolderTree,
  FunctionSquare,
  GitBranch,
  HardDrive,
  LayoutDashboard,
  LogOut,
  Moon,
  RefreshCw,
  Settings2,
  Sun,
  type LucideIcon,
} from 'lucide-react'
import { Avatar } from '@/components/Avatar'
import { Mark } from '@/components/Mark'
import { fetchHealth } from '@/lib/api'
import { isActive } from '@/lib/paths'
import { Link } from '@/lib/router'
import { cn } from '@/lib/utils'
import { useNavCounts } from '@/hooks/useNavCounts'
import { useRouter } from '@/hooks/useRouter'
import { useSession } from '@/hooks/useSession'
import { useSocketStatus } from '@/hooks/useSocketStatus'
import { useTheme } from '@/hooks/useTheme'
import { useUpdateAttention } from '@/hooks/useUpdates'
import { DisplaySettings } from '@/components/layout/DisplaySettings'

/** Sidebar is the shell's left rail: brand, nav, the node's updates, user. */
export function Sidebar({ className }: { className?: string | undefined }) {
  const attention = useNavCounts()
  const updates = useUpdateAttention()
  const health = useQuery({
    queryKey: ['health'],
    queryFn: ({ signal }) => fetchHealth(signal),
    refetchInterval: 10_000,
  })

  const nav: { to: string; label: string; icon: LucideIcon; exact: boolean; dot?: boolean | undefined }[] = [
    { to: '/', label: 'Dashboard', icon: LayoutDashboard, exact: true },
    { to: '/projects', label: 'Projects', icon: FolderTree, exact: false },
    { to: '/services', label: 'Services', icon: Boxes, exact: false, dot: attention.services },
    { to: '/pipelines', label: 'Pipelines', icon: GitBranch, exact: false, dot: attention.pipelines },
    { to: '/functions', label: 'Functions', icon: FunctionSquare, exact: false },
    // Storage carries no dot on purpose: it would cost the sidebar a volume
    // poll of its own on every page, and a breached budget already surfaces
    // through Events, which does have one.
    { to: '/storage', label: 'Storage', icon: HardDrive, exact: false },
    { to: '/events', label: 'Events', icon: Activity, exact: false, dot: attention.events },
    { to: '/backups', label: 'Backups', icon: DatabaseBackup, exact: false },
    { to: '/settings', label: 'Settings', icon: Settings2, exact: false },
  ]

  return (
    <aside
      className={cn(
        'flex w-[230px] shrink-0 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground',
        className,
      )}
    >
      <div className="flex items-center gap-2 px-4 pb-4 pt-5">
        <Mark size={22} />
        <span className="text-base font-semibold tracking-tight">kanea</span>
        {health.data?.version ? (
          <span className="ml-auto font-mono text-[11px] text-muted-foreground">
            {`v${health.data.version.replace(/^v/, '')}`}
          </span>
        ) : null}
        <ThemeToggle />
      </div>

      <nav className="flex flex-col gap-0.5 px-2">
        {nav.map((item) => (
          <NavItem key={item.to} {...item} />
        ))}
      </nav>

      {/* The node's own updates sit apart from the pages above (PRD v1.108):
          those are the workloads, this is the machine. Rendered only while
          something is actually waiting - steady state says nothing. */}
      <div className="mt-auto">
        {updates > 0 ? (
          <Link
            to="/updates"
            className="flex items-center gap-2.5 px-4 py-2.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            <RefreshCw size={14} aria-hidden />
            <span>
              {updates} update{updates === 1 ? '' : 's'} available
            </span>
          </Link>
        ) : null}
        <UserRow />
      </div>
    </aside>
  )
}

function NavItem({
  to,
  label,
  icon: Icon,
  exact,
  dot,
}: {
  to: string
  label: string
  icon: LucideIcon
  exact: boolean
  /** dot marks the page as needing attention. A presence, never a count:
   * a number that is always there is furniture, a dot that appears is a
   * signal. */
  dot?: boolean | undefined
}) {
  const { path } = useRouter()
  const active = isActive(path, to, exact)
  return (
    <Link
      to={to}
      aria-current={active ? 'page' : undefined}
      className={cn(
        'flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm transition-colors',
        active
          ? 'bg-sidebar-accent font-medium text-primary'
          : 'text-muted-foreground hover:bg-sidebar-accent/60 hover:text-foreground',
      )}
    >
      <Icon size={16} aria-hidden />
      <span>{label}</span>
      {dot ? (
        <span
          aria-hidden
          title={`${label} needs attention`}
          className="ml-auto size-1.5 rounded-full bg-status-warn"
        />
      ) : null}
    </Link>
  )
}

/**
 * ThemeToggle sits beside the version, out of the cog since v1.108: the one
 * icon whose meaning is legible without a label, and the brand row is where
 * the theme's evidence is.
 */
function ThemeToggle() {
  const [theme, setTheme] = useTheme()
  const dark = theme === 'dark'
  return (
    <button
      type="button"
      aria-label="Toggle theme"
      title="Toggle theme"
      className="rounded-md border border-sidebar-border p-1 text-muted-foreground hover:bg-sidebar-accent hover:text-foreground"
      onClick={() => setTheme(dark ? 'light' : 'dark')}
    >
      {dark ? <Sun size={13} aria-hidden /> : <Moon size={13} aria-hidden />}
    </button>
  )
}

/**
 * UserRow carries who you are and whether this tab is live, in one card:
 * the presence dot on the avatar and the word beside the role are both the
 * socket, so "reconnecting…" reads as a fact about this session rather
 * than a loose indicator floating above it.
 */
function UserRow() {
  const { session, signOut } = useSession()
  const up = useSocketStatus()
  if (!session) return null

  return (
    <div className="flex items-center gap-2.5 border-t border-sidebar-border px-4 py-3">
      <span className="relative shrink-0">
        <Avatar name={session.subject} />
        <span
          aria-hidden
          className={cn(
            'absolute -bottom-px -right-px size-2 rounded-full border-2 border-sidebar',
            up ? 'bg-status-ok' : 'bg-status-warn',
          )}
        />
      </span>
      <div className="min-w-0">
        {/* Who you are and what you may do, always visible: a viewer who does
            not know they are one reads every missing button as broken. */}
        <div className="truncate text-sm font-medium">{session.subject}</div>
        <div
          className={cn(
            'truncate font-mono text-[11px]',
            up ? 'text-muted-foreground' : 'text-status-warn',
          )}
        >
          {up ? `${session.role} · live` : 'reconnecting…'}
        </div>
      </div>
      <div className="ml-auto flex items-center">
        {/* One cog rather than an icon per setting. Both live in this browser
            rather than on the node, so neither belongs on the admin-only
            Settings page; and the date format needs a label rather than an
            icon, because no icon says which of three orders is in force. */}
        <DisplaySettings />
        <button
          type="button"
          aria-label="Sign out"
          title="Sign out"
          className="rounded-md p-1.5 text-muted-foreground hover:bg-sidebar-accent hover:text-foreground"
          onClick={() => void signOut()}
        >
          <LogOut size={15} />
        </button>
      </div>
    </div>
  )
}
