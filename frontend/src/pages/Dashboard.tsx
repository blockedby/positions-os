export default function Dashboard() {
  return (
    <div>
      <h1>Dashboard</h1>
      <p className="text-muted mb-6">
        Welcome to Positions OS. View your job search statistics and recent activity.
      </p>

      <div className="grid">
        <div className="card">
          <h3>Total Jobs</h3>
          <p className="text-xs text-muted">All scraped jobs</p>
          <h2 className="mt-4">-</h2>
        </div>
        <div className="card">
          <h3>Analyzed</h3>
          <p className="text-xs text-muted">Jobs with structured data</p>
          <h2 className="mt-4">-</h2>
        </div>
        <div className="card">
          <h3>Interested</h3>
          <p className="text-xs text-muted">Jobs you want to apply</p>
          <h2 className="mt-4">-</h2>
        </div>
        <div className="card">
          <h3>Sent</h3>
          <p className="text-xs text-muted">Applications sent</p>
          <h2 className="mt-4">-</h2>
        </div>
      </div>
    </div>
  )
}
