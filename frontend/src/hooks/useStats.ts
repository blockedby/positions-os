import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/types'

export function useStats() {
  return useQuery({
    queryKey: queryKeys.stats(),
    queryFn: () => api.getStats(),
    refetchInterval: 1000 * 60, // Refetch every minute
  })
}

export function useStatsCards() {
  const { data, ...rest } = useStats()

  const cards = data
    ? [
        {
          label: 'Total Jobs',
          value: data.total_jobs,
          description: 'All scraped jobs',
        },
        {
          label: 'Analyzed',
          value: data.analyzed_jobs,
          description: 'Jobs with structured data',
        },
        {
          label: 'Interested',
          value: data.interested_jobs,
          description: 'Jobs you want to apply',
        },
        {
          label: 'Sent',
          value: data.sent_jobs,
          description: 'Applications sent',
        },
        {
          label: 'Targets',
          value: data.total_targets,
          description: `${data.active_targets} active`,
        },
        {
          label: 'Today',
          value: data.today_new_jobs,
          description: 'New jobs today',
        },
      ]
    : []

  return {
    ...rest,
    data: cards,
  }
}
