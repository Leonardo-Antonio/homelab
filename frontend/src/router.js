// Maps app pages to URL paths. The page id matches the nav item ids in AppShell.
const routes = [
  { path: '/clipboard', page: 'clipboard' },
  { path: '/photos', page: 'photos' },
  { path: '/camera', page: 'camera' },
  { path: '/terminal', page: 'terminal' },
  { path: '/notes', page: 'notes' },
]

export const DEFAULT_PAGE = 'clipboard'

export function pageToPath(page) {
  const match = routes.find((route) => route.page === page)
  return match ? match.path : pageToPath(DEFAULT_PAGE)
}

export function pathToPage(pathname) {
  const match = routes.find((route) => route.path === pathname)
  return match ? match.page : DEFAULT_PAGE
}
