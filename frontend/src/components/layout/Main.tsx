import { Outlet } from 'react-router-dom'

export function Main({ children }: { children: React.ReactNode }) {
  return <main className="main-content">{children}</main>
}

export function MainWithOutlet() {
  return (
    <main className="main-content">
      <Outlet />
    </main>
  )
}
