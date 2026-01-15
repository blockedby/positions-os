import { useWebSocket } from '@/hooks/useWebSocket'
import { StatsCards, RecentJobs } from '@/components/dashboard'

export default function Dashboard() {
  // Enable real-time updates
  useWebSocket({ enabled: true })

  return (
    <div className="dashboard-page">
      <h1>Dashboard</h1>
      <p className="text-muted mb-6">
        Welcome to Positions OS. View your job search statistics and recent activity.
      </p>

      <StatsCards className="mb-6" />
      <RecentJobs limit={8} />
    </div>
  )
}
