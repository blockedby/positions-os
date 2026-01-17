import { useState } from 'react'
import { Input, Select, Button } from '@/components/ui'
import type { JobStatus, JobsQuery } from '@/lib/types'

export interface FilterBarProps {
  onFilter: (filters: JobsQuery) => void
  isScraping?: boolean
}

const statusOptions = [
  { value: '', label: 'All Statuses' },
  { value: 'RAW', label: 'Raw' },
  { value: 'ANALYZED', label: 'Analyzed' },
  { value: 'INTERESTED', label: 'Interested' },
  { value: 'REJECTED', label: 'Rejected' },
  { value: 'TAILORED', label: 'Tailored' },
  { value: 'SENT', label: 'Sent' },
  { value: 'RESPONDED', label: 'Responded' },
]

const sortOptions = [
  { value: 'created_at', label: 'Date Created' },
  { value: 'updated_at', label: 'Date Updated' },
  { value: 'salary_max', label: 'Salary' },
]

export const FilterBar = ({ onFilter, isScraping }: FilterBarProps) => {
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState<JobStatus | ''>('')
  const [technologies, setTechnologies] = useState('')
  const [salaryMin, setSalaryMin] = useState('')
  const [salaryMax, setSalaryMax] = useState('')
  const [sortBy, setSortBy] = useState<'created_at' | 'updated_at' | 'salary_max'>('created_at')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')

  const handleApplyFilters = () => {
    onFilter({
      search: search || undefined,
      status: status || undefined,
      technologies: technologies ? technologies.split(',').map(t => t.trim()).filter(Boolean) : undefined,
      salary_min: salaryMin ? parseInt(salaryMin, 10) : undefined,
      salary_max: salaryMax ? parseInt(salaryMax, 10) : undefined,
      sort_by: sortBy,
      sort_order: sortOrder,
    })
  }

  const handleClearFilters = () => {
    setSearch('')
    setStatus('')
    setTechnologies('')
    setSalaryMin('')
    setSalaryMax('')
    setSortBy('created_at')
    setSortOrder('desc')
    onFilter({})
  }

  return (
    <div className="filter-bar">
      <div className="filter-bar-inputs">
        <Input
          variant="search"
          placeholder="Search jobs..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') handleApplyFilters()
          }}
        />
        <Select
          options={statusOptions}
          value={status}
          onChange={(e) => setStatus(e.target.value as JobStatus | '')}
          aria-label="Filter by status"
        />
        <Input
          placeholder="Technologies (e.g., go, react)"
          value={technologies}
          onChange={(e) => setTechnologies(e.target.value)}
        />
        <Input
          type="number"
          placeholder="Min salary"
          value={salaryMin}
          onChange={(e) => setSalaryMin(e.target.value)}
        />
        <Input
          type="number"
          placeholder="Max salary"
          value={salaryMax}
          onChange={(e) => setSalaryMax(e.target.value)}
        />
        <Select
          options={sortOptions}
          value={sortBy}
          onChange={(e) => setSortBy(e.target.value as 'created_at' | 'updated_at' | 'salary_max')}
          aria-label="Sort by"
        />
        <Button
          variant="secondary"
          size="sm"
          onClick={() => setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')}
          aria-label={`Sort ${sortOrder === 'asc' ? 'ascending' : 'descending'}`}
        >
          {sortOrder === 'asc' ? '\u2191' : '\u2193'}
        </Button>
      </div>
      <div className="filter-bar-actions">
        <Button variant="secondary" size="sm" onClick={handleClearFilters}>
          Clear
        </Button>
        <Button variant="primary" size="sm" onClick={handleApplyFilters} loading={isScraping}>
          Apply
        </Button>
      </div>
    </div>
  )
}
