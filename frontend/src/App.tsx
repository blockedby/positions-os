import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import { queryClient } from '@/lib/query-client'
import { Sidebar } from '@/components/layout/Sidebar'
import { Main } from '@/components/layout/Main'
import Dashboard from '@/pages/Dashboard'
import Jobs from '@/pages/Jobs'
import Settings from '@/pages/Settings'

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <div className="app-layout">
          <Sidebar />
          <Routes>
            <Route
              path="/"
              element={
                <Main>
                  <Dashboard />
                </Main>
              }
            />
            <Route
              path="/jobs"
              element={
                <Main>
                  <Jobs />
                </Main>
              }
            />
            <Route
              path="/settings"
              element={
                <Main>
                  <Settings />
                </Main>
              }
            />
          </Routes>
        </div>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export default App
