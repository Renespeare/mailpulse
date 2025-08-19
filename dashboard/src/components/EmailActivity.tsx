import { useState, useEffect } from 'react'
import { 
  MagnifyingGlassIcon,
  FunnelIcon,
  EyeIcon,
  ArrowPathIcon,
  ShieldCheckIcon
} from '@heroicons/react/24/outline'
import { getEmails, getProjects, resendEmail, type Email, type Project } from '../lib/api'
import EmailDetailModal from './EmailDetailModal'
import QuotaMonitor from './QuotaMonitor'

function EmailActivity() {
  const [emails, setEmails] = useState<Email[]>([])
  const [projects, setProjects] = useState<Project[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedEmail, setSelectedEmail] = useState<Email | null>(null)
  const [showEmailDetail, setShowEmailDetail] = useState(false)
  const [projectFilter, setProjectFilter] = useState<string>('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [resendingEmail, setResendingEmail] = useState<string | null>(null)
  const [showQuotaMonitor, setShowQuotaMonitor] = useState(false)
  const [quotaProjectId, setQuotaProjectId] = useState<string | null>(null)
  const [quotaRefreshTrigger, setQuotaRefreshTrigger] = useState(0)

  const fetchEmails = async () => {
    try {
      const filter = projectFilter === 'all' ? undefined : projectFilter
      const data = await getEmails(filter)
      setEmails(data || [])
    } catch (error) {
      console.error('Failed to fetch emails:', error)
      setEmails([])
    }
  }

  const handleRefresh = async () => {
    await fetchEmails()
    // Trigger quota refresh for the current project
    if (quotaProjectId) {
      setQuotaRefreshTrigger(prev => prev + 1)
    }
  }

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      try {
        const [emailsData, projectsData] = await Promise.all([
          getEmails(projectFilter === 'all' ? undefined : projectFilter),
          getProjects()
        ])
        // Batch state updates to prevent multiple renders
        setEmails(emailsData || [])
        setProjects(projectsData || [])
        setLoading(false) // Move setLoading(false) here to batch with other updates
      } catch (error) {
        console.error('Failed to fetch data:', error)
        setLoading(false)
      }
    }

    fetchData()
  }, [projectFilter])

  // Wait for EmailActivity to be stable before showing QuotaMonitor
  useEffect(() => {
    // Hide QuotaMonitor when project changes
    if (quotaProjectId !== projectFilter) {
      setShowQuotaMonitor(false)
      setQuotaProjectId(null)
    }
    
    if (projectFilter !== 'all' && !loading) {
      // Small delay to ensure all renders are complete
      const timer = setTimeout(() => {
        setQuotaProjectId(projectFilter)
        setShowQuotaMonitor(true)
      }, 150)
      
      return () => clearTimeout(timer)
    }
  }, [projectFilter, loading, quotaProjectId])

  const handleProjectChange = (projectId: string) => {
    setProjectFilter(projectId)
  }

  const getProjectName = (projectId: string) => {
    const project = projects.find(p => p.id === projectId)
    return project ? project.name : 'Unknown Project'
  }

  const handleViewEmail = (email: Email) => {
    setSelectedEmail(email)
    setShowEmailDetail(true)
  }

  const handleCloseEmailDetail = () => {
    setShowEmailDetail(false)
    setSelectedEmail(null)
  }

  const handleResend = async (emailId: string) => {
    setResendingEmail(emailId)
    try {
      const result = await resendEmail(emailId)
      if (result.success) {
        // Refresh emails to show updated status
        await fetchEmails()
      }
    } catch (error) {
      console.error('Failed to resend email:', error)
    } finally {
      setResendingEmail(null)
    }
  }

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'delivered':
        return 'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800'
      case 'processed':
        return 'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800'
      case 'failed':
        return 'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800'
      case 'queued':
        return 'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800'
      case 'bounced':
        return 'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-orange-100 text-orange-800'
      default:
        return 'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800'
    }
  }

  // Filter emails based on search query
  const filteredEmails = emails.filter(email => {
    if (!searchQuery) return true
    const query = searchQuery.toLowerCase()
    return (
      email.from.toLowerCase().includes(query) ||
      email.to.some(to => to.toLowerCase().includes(query)) ||
      email.subject.toLowerCase().includes(query)
    )
  })

  // Calculate stats
  const stats = {
    total: filteredEmails.length,
    sent: filteredEmails.filter(e => e.status === 'delivered' || e.status === 'processed').length,
    failed: filteredEmails.filter(e => e.status === 'failed').length,
    queued: filteredEmails.filter(e => e.status === 'queued').length,
    totalSize: filteredEmails.reduce((acc, e) => acc + e.size, 0)
  }

  if (loading) {
    return (
      <div className="p-8 animate-fade-in">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900 mb-2">Email Activity</h1>
          <p className="text-gray-600">Loading email activity...</p>
        </div>
        
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-100 bg-gray-50/50">
            <div className="h-6 bg-gray-200 rounded w-1/3 animate-pulse"></div>
          </div>
          <div className="p-6">
            <div className="space-y-4">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="flex space-x-4 animate-pulse">
                  <div className="h-4 bg-gray-200 rounded flex-1"></div>
                  <div className="h-4 bg-gray-200 rounded w-20"></div>
                  <div className="h-4 bg-gray-200 rounded w-16"></div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="p-8 animate-fade-in">
      <div className="mb-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 mb-2">Email Activity</h1>
            <p className="text-gray-600">
              {projectFilter !== 'all' 
                ? `Filtered by project: ${getProjectName(projectFilter)}`
                : 'Monitor all emails processed through your SMTP relay'
              }
            </p>
          </div>
          
          <div className="flex items-center space-x-4">
            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">{stats.total} emails</span>
            <button 
              onClick={handleRefresh}
              className="inline-flex items-center justify-center px-4 py-2 text-sm font-medium rounded-lg bg-gray-100 text-gray-700 hover:bg-gray-200 transition-colors"
              disabled={loading}
            >
              <ArrowPathIcon className="w-4 h-4 mr-2" />
              Refresh
            </button>
          </div>
        </div>
      </div>

      {/* Filters and Search */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden mb-6">
        <div className="p-6">
          <div className="flex flex-col sm:flex-row gap-4">
            {/* Search */}
            <div className="flex-1">
              <div className="relative">
                <MagnifyingGlassIcon className="absolute left-3 top-3 h-4 w-4 text-gray-400" />
                <input
                  type="text"
                  placeholder="Search emails by sender, recipient, or subject..."
                  className="block w-full rounded-lg border border-gray-300 pl-10 px-3 py-2 placeholder-gray-400 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 transition-colors"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
              </div>
            </div>
            
            {/* Project Filter */}
            {projects.length > 0 && (
              <div className="sm:w-64">
                <div className="relative">
                  <FunnelIcon className="absolute left-3 top-3 h-4 w-4 text-gray-400" />
                  <select
                    value={projectFilter}
                    onChange={(e) => handleProjectChange(e.target.value)}
                    className="block w-full rounded-lg border border-gray-300 pl-10 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 transition-colors bg-white"
                  >
                    <option value="all">All Projects</option>
                    {projects.map((project) => (
                      <option key={project.id} value={project.id}>
                        {project.name} {project.status === 'inactive' ? '(Inactive)' : ''}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Security Notice */}
      <div className="bg-green-50 border border-green-200 rounded-lg p-4 mb-6">
        <div className="flex">
          <ShieldCheckIcon className="h-5 w-5 text-green-400 mt-0.5 flex-shrink-0" />
          <div className="ml-3">
            <p className="text-sm text-green-700">
              <strong>Secure Relay Active:</strong> All displayed emails were authenticated and authorized before processing.
            </p>
          </div>
        </div>
      </div>

      {/* Stats Cards */}
      {stats.total > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 text-center">
            <div className="text-3xl font-bold text-green-600 mb-1">{stats.sent}</div>
            <div className="text-sm font-medium text-gray-500">Successful</div>
          </div>
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 text-center">
            <div className="text-3xl font-bold text-red-600 mb-1">{stats.failed}</div>
            <div className="text-sm font-medium text-gray-500">Failed</div>
          </div>
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 text-center">
            <div className="text-3xl font-bold text-yellow-600 mb-1">{stats.queued}</div>
            <div className="text-sm font-medium text-gray-500">Queued</div>
          </div>
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 text-center">
            <div className="text-3xl font-bold text-gray-600 mb-1">
              {(stats.totalSize / 1024).toFixed(1)}KB
            </div>
            <div className="text-sm font-medium text-gray-500">Total Size</div>
          </div>
        </div>
      )}

      {/* Quota Monitor - only show after EmailActivity is stable */}
      {showQuotaMonitor && quotaProjectId && (
        <div className="mb-6">
          <QuotaMonitor projectId={quotaProjectId} refreshTrigger={quotaRefreshTrigger} />
        </div>
      )}

      {/* Email Table */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-100 bg-gray-50/50">
          <h2 className="text-lg font-medium">Recent Emails</h2>
        </div>
        
        {filteredEmails.length === 0 ? (
          <div className="p-6 text-center py-12">
            <svg className="mx-auto h-12 w-12 text-gray-300 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
            <h3 className="text-sm font-medium text-gray-900 mb-2">
              {searchQuery ? 'No matching emails' : 'No emails yet'}
            </h3>
            <p className="text-sm text-gray-500 mb-4">
              {searchQuery 
                ? 'Try adjusting your search criteria or filters.'
                : 'Send your first email through the SMTP relay to see it here.'
              }
            </p>
            {!searchQuery && (
              <div className="text-xs text-gray-500 bg-gray-50 p-3 rounded mx-auto max-w-sm">
                <strong>SMTP Configuration:</strong><br />
                Host: localhost<br />
                Port: 2525<br />
                Auth: Required (API Key)
              </div>
            )}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">From</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">To</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Subject</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Project</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Sent At</th>
                  <th className="px-6 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {filteredEmails.map((email) => (
                  <tr key={email.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                      <div className="truncate max-w-[200px]">{email.from}</div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      <div className="truncate max-w-[200px]">
                        {Array.isArray(email.to) ? email.to.join(', ') : email.to}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      <div className="truncate max-w-[300px]">{email.subject}</div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={getStatusBadge(email.status)}>
                        {email.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {getProjectName(email.projectId)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {new Date(email.sentAt).toLocaleString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-center">
                      <div className="flex items-center justify-center space-x-2">
                        <button
                          onClick={() => handleViewEmail(email)}
                          className="text-blue-600 hover:text-blue-800 text-sm font-medium"
                          title="View email details"
                        >
                          <EyeIcon className="w-4 h-4" />
                        </button>
                        {(email.status === 'failed' || email.status === 'queued') && (
                          <button
                            onClick={() => handleResend(email.id)}
                            disabled={resendingEmail === email.id}
                            className="text-green-600 hover:text-green-800 text-sm font-medium disabled:opacity-50"
                            title="Resend email"
                          >
                            {resendingEmail === email.id ? (
                              <div className="loading-spinner"></div>
                            ) : (
                              <ArrowPathIcon className="w-4 h-4" />
                            )}
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Email Detail Modal */}
      {showEmailDetail && selectedEmail && (
        <EmailDetailModal
          email={selectedEmail}
          onClose={handleCloseEmailDetail}
          projectName={getProjectName(selectedEmail.projectId)}
        />
      )}
    </div>
  )
}

export default EmailActivity