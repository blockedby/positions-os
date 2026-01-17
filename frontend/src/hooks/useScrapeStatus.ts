import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/types'

export function useScrapeStatus() {
  return useQuery({
    queryKey: queryKeys.scrapeStatus(),
    queryFn: () => api.getScrapeStatus(),
    refetchInterval: 3000, // Poll every 3 seconds when scraping
  })
}
