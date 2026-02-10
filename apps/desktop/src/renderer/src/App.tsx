import { createHashRouter, RouterProvider } from 'react-router-dom'
import Layout from './pages/layout'
import HomePage from './pages/home'
import ChatPage from './pages/chat'
import ToolsPage from './pages/tools'
import SkillsPage from './pages/skills'
import ChannelsPage from './pages/channels'
import CronsPage from './pages/crons'

const router = createHashRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      { index: true, element: <HomePage /> },
      { path: 'chat', element: <ChatPage /> },
      { path: 'tools', element: <ToolsPage /> },
      { path: 'skills', element: <SkillsPage /> },
      { path: 'channels', element: <ChannelsPage /> },
      { path: 'crons', element: <CronsPage /> },
    ],
  },
])

export default function App() {
  return <RouterProvider router={router} />
}
