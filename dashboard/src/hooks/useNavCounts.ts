import { useQuery } from '@tanstack/react-query'
import {
  allocsResponseSchema,
  fetchEvents,
  fetchRuns,
  servicesResponseSchema,
  Topic,
} from '@/lib/api'
import { useLiveTopic } from '@/hooks/useLiveTopic'
import { groupAllocs, serviceHealth } from '@/lib/state'

export interface NavAttention {
  /** services: some service is not settled - degraded, scaling, failing. */
  services?: boolean | undefined
  /** pipelines: a build is running or queued right now. */
  pipelines?: boolean | undefined
  /** events: warnings or errors in the last 24 h. */
  events?: boolean | undefined
}

/**
 * useNavCounts feeds the sidebar's attention dots. A dot means "this page
 * has something that needs looking at", never "this page has N things": the
 * mockup dropped the counts because a number that is always there is
 * furniture, while a dot that appears is a signal.
 *
 * Every source is shared with the page that owns it: the services/allocs WS
 * keys and the ['runs'] / ['events', ''] query keys are the same ones those
 * pages use, so the sidebar costs no extra subscription or request while
 * they are open.
 */
export function useNavCounts(): NavAttention {
  const services = useLiveTopic({ topic: Topic.Services }, servicesResponseSchema)
  const allocs = useLiveTopic({ topic: Topic.Allocs }, allocsResponseSchema)

  const runs = useQuery({
    queryKey: ['runs'],
    queryFn: ({ signal }) => fetchRuns({ limit: 200 }, signal),
    refetchInterval: 15_000,
  })

  const alerts = useQuery({
    queryKey: ['events', ''],
    queryFn: ({ signal }) => fetchEvents({ limit: 200 }, signal),
    refetchInterval: 15_000,
    // Counted in select rather than in render: the clock read is impure, and
    // here it runs when data (or a refetch) arrives instead of on every render.
    select: (data) => {
      const dayAgo = Date.now() - 24 * 60 * 60 * 1000
      return data.events.filter((e) => e.severity !== 'info' && Date.parse(e.at) >= dayAgo).length
    },
  })

  const list = services.data?.services ?? undefined
  const byService = groupAllocs(allocs.data?.allocs ?? [])
  const unsettled =
    list === undefined
      ? undefined
      : list.some(
          (svc) =>
            !serviceHealth(svc, byService.get(`${svc.Project}/${svc.Service}`) ?? []).settled,
        )

  return {
    services: unsettled,
    pipelines: runs.data?.some((r) => r.state === 'running' || r.state === 'queued'),
    events: alerts.data === undefined ? undefined : alerts.data > 0,
  }
}
