import { useState, useEffect } from 'react'
import { BrowserRouter as Router, Routes, Route, Link, useLocation, Navigate } from 'react-router-dom'
import { 
  HomeIcon, 
  EnvelopeIcon, 
  CogIcon,
  ServerIcon,
  Bars3Icon,
  ChevronLeftIcon
} from '@heroicons/react/24/outline'
import { 
  HomeIcon as HomeIconSolid, 
  EnvelopeIcon as EnvelopeIconSolid, 
  CogIcon as CogIconSolid
} from '@heroicons/react/24/solid'
import { getRelayHealth } from './lib/api'
import Dashboard from './components/Dashboard'
import EmailActivity from './components/EmailActivity'
import Projects from './components/Projects'

interface RelayHealth {
  status: string
  message: string
}

interface NavigationItem {
  path: string
  name: string
  icon: any
  iconSolid: any
  description: string
}

function App() {
  return (
    <Router>
      <AppLayout />
    </Router>
  )
}

function AppLayout() {
  const location = useLocation()
  const [relayHealth, setRelayHealth] = useState<RelayHealth | null>(null)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => {
    // Remember sidebar state, but default to collapsed on mobile
    const saved = localStorage.getItem('mailpulse-sidebar-collapsed')
    return saved !== null ? JSON.parse(saved) : window.innerWidth < 768
  })

  const navigation: NavigationItem[] = [
    { 
      path: '/dashboard', 
      name: 'Dashboard', 
      icon: HomeIcon,
      iconSolid: HomeIconSolid,
      description: 'Overview and metrics'
    },
    { 
      path: '/emails', 
      name: 'Email Activity', 
      icon: EnvelopeIcon,
      iconSolid: EnvelopeIconSolid,
      description: 'Monitor sent emails'
    },
    { 
      path: '/projects', 
      name: 'Projects', 
      icon: CogIcon,
      iconSolid: CogIconSolid,
      description: 'Manage SMTP projects'
    },
  ]

  useEffect(() => {
    const checkRelayHealth = async () => {
      const health = await getRelayHealth()
      setRelayHealth(health)
    }
    
    checkRelayHealth()
    const interval = setInterval(checkRelayHealth, 30000)
    
    return () => clearInterval(interval)
  }, [])

  const handleSidebarToggle = () => {
    const newState = !sidebarCollapsed
    setSidebarCollapsed(newState)
    localStorage.setItem('mailpulse-sidebar-collapsed', JSON.stringify(newState))
  }

  const getHealthStatusBadge = () => {
    if (!relayHealth) {
      return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">Checking...</span>
    }

    switch (relayHealth.status) {
      case 'healthy':
        return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">Online</span>
      case 'unhealthy':
        return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">Warning</span>
      default:
        return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">Offline</span>
    }
  }

  return (
    <div className="flex h-screen bg-gray-50">
      {/* Mobile Overlay */}
      {!sidebarCollapsed && (
        <div 
          className="md:hidden fixed inset-0 bg-black bg-opacity-50 z-40"
          onClick={() => {
            setSidebarCollapsed(true)
            localStorage.setItem('mailpulse-sidebar-collapsed', JSON.stringify(true))
          }}
        />
      )}

      {/* Sidebar */}
      <div className={`${
        sidebarCollapsed 
          ? 'w-16 md:w-16' 
          : 'w-64 md:w-64'
        } ${
        sidebarCollapsed 
          ? 'md:relative fixed md:translate-x-0 -translate-x-full' 
          : 'md:relative fixed md:translate-x-0 translate-x-0'
        } bg-white border-r border-gray-200 h-screen overflow-y-auto transition-all duration-300 ease-in-out flex flex-col z-50`}>
        {/* Header with Collapse Button */}
        <div className={`${sidebarCollapsed ? 'p-3' : 'p-6'} flex items-center justify-between border-b border-gray-100`}>
          {!sidebarCollapsed && (
            <div className="flex-1">
              <h1 className="text-xl font-bold text-gradient">MailPulse</h1>
              <p className="text-sm text-gray-500">SMTP Relay Dashboard</p>
            </div>
          )}
          <button
            onClick={handleSidebarToggle}
            className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
            title={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {sidebarCollapsed ? (
              <Bars3Icon className="w-5 h-5" />
            ) : (
              <ChevronLeftIcon className="w-5 h-5" />
            )}
          </button>
        </div>

        {/* Navigation */}
        <nav className={`${sidebarCollapsed ? 'px-2' : 'px-4'} py-4 space-y-2 flex-1`}>
          {navigation.map((item) => {
            const isActive = location.pathname === item.path
            const Icon = isActive ? item.iconSolid : item.icon
            return (
              <Link
                key={item.path}
                to={item.path}
                className={`w-full flex items-center ${sidebarCollapsed ? 'justify-center px-3 py-3' : 'px-4 py-3'} text-sm font-medium rounded-lg transition-colors ${
                  isActive 
                    ? 'bg-blue-50 text-blue-700' 
                    : 'text-gray-600 hover:bg-gray-50 hover:text-gray-900'
                }`}
                title={sidebarCollapsed ? item.name : undefined}
              >
                <Icon className="w-5 h-5 flex-shrink-0" />
                {!sidebarCollapsed && (
                  <div className="ml-3 text-left">
                    <div className="font-medium">{item.name}</div>
                    <div className="text-xs opacity-60">{item.description}</div>
                  </div>
                )}
              </Link>
            )
          })}
        </nav>

        {/* Relay Status */}
        <div className={`${sidebarCollapsed ? 'p-2' : 'p-4'} border-t border-gray-100`}>
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-3">
            <div className={`flex items-center ${sidebarCollapsed ? 'justify-center' : ''} group relative`}>
              <ServerIcon className={`w-4 h-4 flex-shrink-0 ${
                !relayHealth ? 'text-gray-400' :
                relayHealth.status === 'healthy' ? 'text-green-500' :
                relayHealth.status === 'unhealthy' ? 'text-yellow-500' :
                'text-red-500'
              }`} />
              {!sidebarCollapsed && (
                <div className="ml-2">
                  <div className="text-xs font-medium text-gray-600">Relay Status</div>
                  <div className="mt-1">{getHealthStatusBadge()}</div>
                </div>
              )}
              {sidebarCollapsed && (
                <div className="absolute left-16 bg-gray-800 text-white text-xs px-2 py-1 rounded opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap z-10">
                  {relayHealth ? relayHealth.status : 'Checking...'}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Main Content */}
      <div className="flex-1 overflow-hidden">
        {/* Mobile Header with Sidebar Toggle */}
        <div className="md:hidden bg-white border-b border-gray-200 p-4">
          <button
            onClick={handleSidebarToggle}
            className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
          >
            <Bars3Icon className="w-6 h-6" />
          </button>
        </div>
        
        <div className="h-full overflow-y-auto custom-scrollbar">
          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/emails" element={<EmailActivity />} />
            <Route path="/projects" element={<Projects />} />
          </Routes>
        </div>
      </div>
    </div>
  )
}

export default App