import { Card, Skeleton } from '@/components/ui'
import { useStatsCards } from '@/hooks/useStats'

export interface StatsCardsProps {
  className?: string
}

export const StatsCards = ({ className = '' }: StatsCardsProps) => {
  const { data: cards, isLoading, error } = useStatsCards()

  if (isLoading) {
    return <StatsCardsSkeleton className={className} />
  }

  if (error) {
    return (
      <div className={`stats-cards ${className}`}>
        <Card className="stats-error">
          <p className="text-muted">Failed to load statistics</p>
        </Card>
      </div>
    )
  }

  return (
    <div className={`stats-cards ${className}`}>
      {cards.map((card) => (
        <Card key={card.label} className="stat-card">
          <h3 className="stat-label">{card.label}</h3>
          <p className="stat-description text-xs text-muted">{card.description}</p>
          <h2 className="stat-value">{card.value.toLocaleString()}</h2>
        </Card>
      ))}
    </div>
  )
}

const StatsCardsSkeleton = ({ className = '' }: { className?: string }) => (
  <div className={`stats-cards ${className}`}>
    {Array.from({ length: 6 }).map((_, i) => (
      <Card key={i} className="stat-card">
        <Skeleton variant="text" className="w-20 h-4" />
        <Skeleton variant="text" className="w-32 h-3 mt-2" />
        <Skeleton variant="text" className="w-12 h-8 mt-4" />
      </Card>
    ))}
  </div>
)
