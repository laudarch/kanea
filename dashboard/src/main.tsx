import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { App } from '@/App'
// The design system's two faces, self-hosted (nothing from a CDN): Space
// Grotesk for UI, JetBrains Mono for what a machine wrote.
import '@fontsource-variable/space-grotesk'
import '@fontsource-variable/jetbrains-mono'
import '@/index.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Live data arrives over the socket, so REST queries exist for things
      // that change rarely. Refetching them on every window focus would be
      // noise against a daemon that is already pushing what matters.
      refetchOnWindowFocus: false,
      staleTime: 30_000,
    },
  },
})

const root = document.getElementById('root')
if (!root) throw new Error('no #root element')

createRoot(root).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>,
)
