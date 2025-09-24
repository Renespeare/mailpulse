import { useState, useEffect } from 'react'
import { 
  EnvelopeIcon, 
  CheckCircleIcon, 
  XCircleIcon,
  ClockIcon,
  ChartBarIcon,
  ServerIcon,
  CpuChipIcon
} from '@heroicons/react/24/outline'
import { getProjects, getQuotaUsage, getEmailStats, getAllEmailStats, type Project, type QuotaUsage, type EmailStats } from '../lib/api'

function Dashboard() {
  const [projects, setProjects] = useState<Project[]>([])
  const [loading, setLoading] = useState(true)
  const [quotaData, setQuotaData] = useState<Record<string, QuotaUsage>>({})
  const [emailStats, setEmailStats] = useState<Record<string, EmailStats>>({})
  const [allEmailStats, setAllEmailStats] = useState<EmailStats | null>(null)

  useEffect(() => {
    const fetchDashboardData = async () => {
      try {
        // Fetch projects and all email stats in parallel
        const [projectsData, allStats] = await Promise.all([
          getProjects(),
          getAllEmailStats()
        ])
        
        setProjects(projectsData)
        setAllEmailStats(allStats)

        // Fetch quota and individual stats for each project (for project cards)
        const projectPromises = projectsData.map(async (project) => {
          const quota = await getQuotaUsage(project.id)
          const stats = await getEmailStats(project.id)
          return { projectId: project.id, quota, stats }
        })

        const results = await Promise.all(projectPromises)
        
        const quotaMap: Record<string, QuotaUsage> = {}
        const statsMap: Record<string, EmailStats> = {}
        
        results.forEach(({ projectId, quota, stats }) => {
          if (quota) quotaMap[projectId] = quota
          if (stats) statsMap[projectId] = stats
        })

        setQuotaData(quotaMap)
        setEmailStats(statsMap)
      } catch (error) {
        console.error('Failed to fetch dashboard data:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchDashboardData()
    const interval = setInterval(fetchDashboardData, 60000) // Refresh every minute

    return () => clearInterval(interval)
  }, [])

  // Use API stats for totals instead of calculating client-side
  const totalStats = allEmailStats ? {
    totalEmails: allEmailStats.totalEmails,
    sentEmails: allEmailStats.sentEmails,
    failedEmails: allEmailStats.failedEmails,
    queuedEmails: allEmailStats.queuedEmails
  } : {
    totalEmails: 0,
    sentEmails: 0,
    failedEmails: 0,
    queuedEmails: 0
  }

  const successRate = allEmailStats?.successRate ? Math.round(allEmailStats.successRate) : 0

  if (loading) {
    return (
      <div className="p-8 animate-fade-in">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900 mb-2">Dashboard</h1>
          <p className="text-gray-600">Loading your SMTP relay overview...</p>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 text-center animate-pulse">
              <div className="h-8 bg-gray-200 rounded mb-2"></div>
              <div className="h-4 bg-gray-200 rounded w-2/3 mx-auto"></div>
            </div>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="p-8 animate-fade-in">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900 mb-2">Dashboard</h1>
        <p className="text-gray-600">
          Monitor your SMTP relay performance and project statistics
        </p>
      </div>

      {/* Overview Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 text-center bg-gradient-to-br from-blue-50 to-indigo-50 border-l-4 border-blue-500">
          <div className="flex items-center justify-between mb-2">
            <EnvelopeIcon className="w-8 h-8 text-blue-600" />
            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">{projects.length} Projects</span>
          </div>
          <div className="text-3xl font-bold text-blue-700 mb-1">{totalStats.totalEmails.toLocaleString()}</div>
          <div className="text-sm font-medium text-gray-500">Total Emails</div>
        </div>

        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 text-center bg-gradient-to-br from-green-50 to-emerald-50 border-l-4 border-green-500">
          <div className="flex items-center justify-between mb-2">
            <CheckCircleIcon className="w-8 h-8 text-green-600" />
            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">{successRate}% Success</span>
          </div>
          <div className="text-3xl font-bold text-green-700 mb-1">{totalStats.sentEmails.toLocaleString()}</div>
          <div className="text-sm font-medium text-gray-500">Delivered</div>
        </div>

        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 text-center bg-gradient-to-br from-red-50 to-rose-50 border-l-4 border-red-500">
          <div className="flex items-center justify-between mb-2">
            <XCircleIcon className="w-8 h-8 text-red-600" />
            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">{totalStats.failedEmails > 0 ? 'Issues' : 'Good'}</span>
          </div>
          <div className="text-3xl font-bold text-red-700 mb-1">{totalStats.failedEmails.toLocaleString()}</div>
          <div className="text-sm font-medium text-gray-500">Failed</div>
        </div>

        <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 text-center bg-gradient-to-br from-yellow-50 to-amber-50 border-l-4 border-yellow-500">
          <div className="flex items-center justify-between mb-2">
            <ClockIcon className="w-8 h-8 text-yellow-600" />
            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">{totalStats.queuedEmails > 0 ? 'Processing' : 'Empty'}</span>
          </div>
          <div className="text-3xl font-bold text-yellow-700 mb-1">{totalStats.queuedEmails.toLocaleString()}</div>
          <div className="text-sm font-medium text-gray-500">Queued</div>
        </div>
      </div>

      {/* Quick Actions */}
      <div className="mb-8">
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-100 bg-gray-50/50">
            <h2 className="text-lg font-semibold">Quick Setup</h2>
          </div>
          <div className="p-6">
            <div className="bg-gray-50 rounded-lg p-4 mb-4">
              <h3 className="font-medium mb-2 flex items-center">
                <CpuChipIcon className="w-5 h-5 mr-2 text-gray-600" />
                SMTP Configuration
              </h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
                <div>
                  <div className="font-medium text-gray-700">Host</div>
                  <code className="bg-white px-2 py-1 rounded border text-gray-600">localhost</code>
                </div>
                <div>
                  <div className="font-medium text-gray-700">Port</div>
                  <code className="bg-white px-2 py-1 rounded border text-gray-600">2525</code>
                </div>
                <div>
                  <div className="font-medium text-gray-700">Authentication</div>
                  <code className="bg-white px-2 py-1 rounded border text-gray-600">Required</code>
                </div>
              </div>
            </div>
            
            {/* <div className="flex items-center justify-between">
              <div className="text-sm text-gray-600">
                Need help getting started? Check the documentation for examples.
              </div>
              <button className="inline-flex items-center justify-center px-6 py-3 text-base font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 transition-colors">
                View Docs
              </button>
            </div> */}
          </div>
        </div>
      </div>

      {/* Projects Overview */}
      {projects.length > 0 ? (
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-100 bg-gray-50/50">
            <h2 className="text-lg font-semibold">Active Projects</h2>
          </div>
          <div className="p-6">
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {projects.filter(p => p.status === 'active').slice(0, 4).map((project) => {
                const quota = quotaData[project.id]
                const stats = emailStats[project.id]
                
                return (
                  <div key={project.id} className="border rounded-lg p-4 hover:shadow-md transition-shadow">
                    <div className="flex items-center justify-between mb-3">
                      <h3 className="font-medium text-gray-900">{project.name}</h3>
                      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">Active</span>
                    </div>
                    
                    <div className="grid grid-cols-2 gap-4 text-sm">
                      <div>
                        <div className="text-gray-500">Today's Emails</div>
                        <div className="font-medium">{quota?.dailyUsed || 0}</div>
                      </div>
                      <div>
                        <div className="text-gray-500">Overall Success Rate</div>
                        <div className="font-medium">
                          {stats && stats.totalEmails > 0 
                            ? Math.round((stats.sentEmails / stats.totalEmails) * 100) + '%'
                            : 'N/A'
                          }
                        </div>
                      </div>
                      <div>
                        <div className="text-gray-500">Daily Quota</div>
                        <div className="font-medium">
                          {quota?.dailyUsed || 0} / {quota?.dailyLimit || project.quotaDaily}
                        </div>
                      </div>
                      <div>
                        <div className="text-gray-500">Daily Usage</div>
                        <div className="font-medium">
                         {quota.dailyUsagePercent < 1 && quota.dailyUsagePercent > 0 
                            ? quota.dailyUsagePercent.toFixed(1) 
                            : Math.round(quota.dailyUsagePercent)
                          }%
                        </div>
                      </div>
                    </div>

                    {quota && (
                      <div className="mt-3">
                        <div className="w-full bg-gray-200 rounded-full h-2">
                          <div 
                            className={`h-2 rounded-full ${
                              quota.dailyUsagePercent > 80 ? 'bg-red-500' :
                              quota.dailyUsagePercent > 60 ? 'bg-yellow-500' :
                              'bg-green-500'
                            }`}
                            style={{ width: `${Math.min(quota.dailyUsagePercent, 100)}%` }}
                          ></div>
                        </div>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
            
            {projects.filter(p => p.status === 'active').length === 0 && (
              <div className="text-center py-8 text-gray-500">
                <ServerIcon className="w-12 h-12 mx-auto mb-3 text-gray-300" />
                <p>No active projects yet. Create your first project to get started!</p>
              </div>
            )}
          </div>
        </div>
      ) : (
        /* Empty State */
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
          <div className="p-6 text-center py-12">
            <ChartBarIcon className="w-16 h-16 mx-auto text-gray-300 mb-4" />
            <h3 className="text-xl font-medium text-gray-900 mb-2">Welcome to MailPulse</h3>
            <p className="text-gray-600 mb-6">
              Create your first SMTP project to start monitoring email delivery
            </p>
            <button className="inline-flex items-center justify-center px-6 py-3 text-base font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 transition-colors">
              Create First Project
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

export default Dashboard