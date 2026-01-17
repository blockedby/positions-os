import { useInfiniteQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { JobsQuery } from '@/lib/types'

const PAGE_SIZE = 20

export function useInfiniteJobs(filters?: Omit<JobsQuery, 'page' | 'limit'>) {
  return useInfiniteQuery({
    queryKey: ['jobs', 'infinite', filters],
    queryFn: async ({ pageParam = 1 }) => {
      return api.getJobs({
        ...filters,
        page: pageParam,
        limit: PAGE_SIZE,
      })
    },
    initialPageParam: 1,
    getNextPageParam: (lastPage) => {
      if (lastPage.page < lastPage.pages) {
        return lastPage.page + 1
      }
      return undefined
    },
  })
}
