import { useState, useEffect, useRef } from 'react'
import { 
  MagnifyingGlassIcon,
  FunnelIcon,
  EyeIcon,
  ArrowPathIcon,
  ShieldCheckIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  XMarkIcon
} from '@heroicons/react/24/outline'
import { getEmails, getProjects, resendEmail, getEmailStats, getAllEmailStats, type Email, type Project, type EmailsResponse, type EmailStats } from '../lib/api'
import EmailDetailModal from './EmailDetailModal'
import QuotaMonitor from './QuotaMonitor'

function EmailActivity() {
  const [emails, setEmails] = useState<Email[]>([])
  const [projects, setProjects] = useState<Project[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedEmail, setSelectedEmail] = useState<Email | null>(null)
  const [showEmailDetail, setShowEmailDetail] = useState(false)
  const [projectFilter, setProjectFilter] = useState<string>('all')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [debouncedSearchQuery, setDebouncedSearchQuery] = useState('')
  const [resendingEmail, setResendingEmail] = useState<string | null>(null)
  const [showQuotaMonitor, setShowQuotaMonitor] = useState(false)
  const [quotaProjectId, setQuotaProjectId] = useState<string | null>(null)
  const [quotaRefreshTrigger, setQuotaRefreshTrigger] = useState(0)
  // Pagination state
  const [currentPage, setCurrentPage] = useState(1)
  const [totalCount, setTotalCount] = useState(0)
  const [hasMore, setHasMore] = useState(false)
  const [limit] = useState(20)
  const [refreshingEmails, setRefreshingEmails] = useState(false)
  const [emailStats, setEmailStats] = useState<EmailStats | null>(null)
  const [loadingStats, setLoadingStats] = useState(false)
  
  // Ref for search input to maintain focus
  const searchInputRef = useRef<HTMLInputElement>(null)

  const handleRefresh = async () => {
    // Store current focus state
    const wasSearchFocused = document.activeElement === searchInputRef.current
    
    setRefreshingEmails(true)
    
    try {
      // Refresh both emails and stats
      await Promise.all([
        fetchEmailsData(searchQuery, currentPage),
        fetchEmailStats()
      ])
      
      // Trigger quota refresh for the current project
      if (quotaProjectId) {
        setQuotaRefreshTrigger(prev => prev + 1)
      }
    } catch (error) {
      console.error('Failed to refresh emails:', error)
    } finally {
      setRefreshingEmails(false)
      
      // Restore focus if search was focused
      if (wasSearchFocused && searchInputRef.current) {
        setTimeout(() => {
          searchInputRef.current?.focus()
        }, 50)
      }
    }
  }

  // Debounce search query only
  useEffect(() => {
    const timer = setTimeout(async () => {
      setDebouncedSearchQuery(searchQuery)
      setCurrentPage(1) // Reset to first page when search changes
      
      // Only fetch if this is a search change, not filter change
      if (!loading && searchQuery !== debouncedSearchQuery) {
        await fetchEmailsData(searchQuery)
      }
    }, 500)

    return () => clearTimeout(timer)
  }, [searchQuery]) // Only depend on searchQuery

  // Helper function to fetch stats for current project filter
  const fetchEmailStats = async () => {
    setLoadingStats(true)
    
    try {
      if (projectFilter === 'all') {
        // Call all projects stats API
        const stats = await getAllEmailStats()
        setEmailStats(stats)
      } else {
        // Call single project stats API
        const stats = await getEmailStats(projectFilter)
        setEmailStats(stats)
      }
    } catch (error) {
      console.error('Failed to fetch email stats:', error)
      setEmailStats(null)
    } finally {
      setLoadingStats(false)
    }
  }

  // Helper function to fetch emails with current filters
  const fetchEmailsData = async (search: string = searchQuery, page: number = 1) => {
    try {
      const filter = projectFilter === 'all' ? undefined : projectFilter
      const status = statusFilter === 'all' ? undefined : statusFilter
      const offset = (page - 1) * limit
      const data: EmailsResponse = await getEmails(filter, search, limit, offset, status)
      
      setEmails(data.emails || [])
      setTotalCount(data.totalCount || 0)
      setHasMore(data.hasMore || false)
    } catch (error) {
      console.error('Failed to fetch emails:', error)
    }
  }

  // Initial data loading and filter changes (project/status filters)
  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      try {
        // Always fetch projects to ensure we have the latest
        const projectsData = await getProjects()
        setProjects(projectsData || [])
        
        // Reset pagination when filters change
        setCurrentPage(1)
        
        // Fetch emails and stats for the new filters
        const filter = projectFilter === 'all' ? undefined : projectFilter
        const status = statusFilter === 'all' ? undefined : statusFilter
        
        const [emailsData] = await Promise.all([
          getEmails(filter, searchQuery, limit, 0, status),
          fetchEmailStats() // Fetch stats separately
        ])
        
        setEmails(emailsData.emails || [])
        setTotalCount(emailsData.totalCount || 0)
        setHasMore(emailsData.hasMore || false)
        
        setLoading(false)
      } catch (error) {
        console.error('Failed to fetch data:', error)
        setLoading(false)
      }
    }

    fetchData()
  }, [projectFilter, statusFilter, limit]) // Remove searchQuery dependency to avoid double calls

  // Handle debounced search changes
  useEffect(() => {
    if (!loading && debouncedSearchQuery !== searchQuery) {
      // Search query changed, fetch with new search
      fetchEmailsData(debouncedSearchQuery, 1)
    }
  }, [debouncedSearchQuery, loading])

  // Fetch emails when page changes (without full page refresh)
  useEffect(() => {
    if (!loading && currentPage > 1) {
      const filter = projectFilter === 'all' ? undefined : projectFilter
      const status = statusFilter === 'all' ? undefined : statusFilter
      const offset = (currentPage - 1) * limit
      
      const fetchPageEmails = async () => {
        try {
          const data: EmailsResponse = await getEmails(filter, searchQuery, limit, offset, status)
          
          setEmails(data.emails || [])
          setTotalCount(data.totalCount || 0)
          setHasMore(data.hasMore || false)
        } catch (error) {
          console.error('Failed to fetch page emails:', error)
        }
      }
      
      fetchPageEmails()
    }
  }, [currentPage, loading, projectFilter, statusFilter, searchQuery, limit])

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
    setCurrentPage(1) // Reset to first page when project changes
  }

  const handleStatusChange = (status: string) => {
    setStatusFilter(status)
    setCurrentPage(1) // Reset to first page when status changes
  }

  const handlePageChange = (page: number) => {
    setCurrentPage(page)
  }

  const handlePrevPage = () => {
    if (currentPage > 1) {
      setCurrentPage(currentPage - 1)
    }
  }

  const handleNextPage = () => {
    if (hasMore) {
      setCurrentPage(currentPage + 1)
    }
  }

  const handleClearSearch = () => {
    setSearchQuery('')
    if (searchInputRef.current) {
      searchInputRef.current.focus()
    }
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
        await fetchEmailsData(searchQuery, currentPage)
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

  // Use API stats when available, fallback to calculated stats from current page
  const stats = emailStats ? {
    total: emailStats.totalEmails,
    sent: emailStats.sentEmails,
    failed: emailStats.failedEmails,
    queued: emailStats.queuedEmails,
    totalSize: emailStats.totalSize
  } : {
    total: totalCount, // Use server-provided total count
    sent: emails.filter(e => e.status === 'delivered' || e.status === 'processed').length,
    failed: emails.filter(e => e.status === 'failed').length,
    queued: emails.filter(e => e.status === 'queued').length,
    totalSize: emails.reduce((acc, e) => acc + e.size, 0)
  }

  // Calculate pagination info
  const totalPages = Math.ceil(totalCount / limit)
  const startItem = totalCount > 0 ? (currentPage - 1) * limit + 1 : 0
  const endItem = Math.min(currentPage * limit, totalCount)

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
              {projectFilter !== 'all' || statusFilter !== 'all'
                ? `Filtered by ${
                    [projectFilter !== 'all' ? `project: ${getProjectName(projectFilter)}` : null,
                     statusFilter !== 'all' ? `status: ${statusFilter}` : null
                    ].filter(Boolean).join(', ')
                  }`
                : 'Monitor all emails processed through your SMTP relay'
              }
            </p>
          </div>
          
          <div className="flex items-center space-x-4">
            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
              {stats.total} total emails
            </span>
            {totalCount > 0 && (
              <span className="text-sm text-gray-500">
                Showing {startItem}-{endItem} of {totalCount}
              </span>
            )}
            <button 
              onClick={handleRefresh}
              className="inline-flex items-center justify-center px-4 py-2 text-sm font-medium rounded-lg bg-gray-100 text-gray-700 hover:bg-gray-200 transition-colors"
              disabled={loading || refreshingEmails}
            >
              <ArrowPathIcon className={`w-4 h-4 mr-2 ${refreshingEmails ? 'animate-spin' : ''}`} />
              {refreshingEmails ? 'Refreshing...' : 'Refresh'}
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
                  ref={searchInputRef}
                  type="text"
                  placeholder="Search emails by sender, recipient, or subject..."
                  className="block w-full rounded-lg border border-gray-300 pl-10 pr-10 px-3 py-2 placeholder-gray-400 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 transition-colors"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
                {searchQuery && (
                  <button
                    onClick={handleClearSearch}
                    className="absolute right-3 top-3 h-4 w-4 text-gray-400 hover:text-gray-600 transition-colors"
                    title="Clear search"
                  >
                    <XMarkIcon className="h-4 w-4" />
                  </button>
                )}
              </div>
            </div>
            
            {/* Status Filter */}
            <div className="sm:w-48">
              <div className="relative">
                <FunnelIcon className="absolute left-3 top-3 h-4 w-4 text-gray-400" />
                <select
                  value={statusFilter}
                  onChange={(e) => handleStatusChange(e.target.value)}
                  className="block w-full rounded-lg border border-gray-300 pl-10 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 transition-colors bg-white"
                >
                  <option value="all">All Status</option>
                  <option value="delivered">Delivered</option>
                  <option value="processed">Processed</option>
                  <option value="failed">Failed</option>
                  <option value="queued">Queued</option>
                  <option value="bounced">Bounced</option>
                </select>
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
      {(stats.total > 0 || loadingStats) && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 text-center">
            <div className="text-3xl font-bold text-green-600 mb-1">
              {loadingStats ? (
                <div className="animate-pulse bg-gray-200 rounded h-8 w-12 mx-auto"></div>
              ) : (
                stats.sent
              )}
            </div>
            <div className="text-sm font-medium text-gray-500">Successful</div>
          </div>
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 text-center">
            <div className="text-3xl font-bold text-red-600 mb-1">
              {loadingStats ? (
                <div className="animate-pulse bg-gray-200 rounded h-8 w-12 mx-auto"></div>
              ) : (
                stats.failed
              )}
            </div>
            <div className="text-sm font-medium text-gray-500">Failed</div>
          </div>
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 text-center">
            <div className="text-3xl font-bold text-yellow-600 mb-1">
              {loadingStats ? (
                <div className="animate-pulse bg-gray-200 rounded h-8 w-12 mx-auto"></div>
              ) : (
                stats.queued
              )}
            </div>
            <div className="text-sm font-medium text-gray-500">Queued</div>
          </div>
          <div className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 text-center">
            <div className="text-3xl font-bold text-gray-600 mb-1">
              {loadingStats ? (
                <div className="animate-pulse bg-gray-200 rounded h-8 w-12 mx-auto"></div>
              ) : (
                (stats.totalSize / 1024).toFixed(1) + 'KB'
              )}
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
      <div className={`bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden transition-opacity duration-200 ${refreshingEmails ? 'opacity-75' : 'opacity-100'}`}>
        <div className="px-6 py-4 border-b border-gray-100 bg-gray-50/50">
          <h2 className="text-lg font-medium">Recent Emails</h2>
        </div>
        
        {emails.length === 0 ? (
          <div className="p-6 text-center py-12">
            <svg className="mx-auto h-12 w-12 text-gray-300 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 8l7.89 4.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
            <h3 className="text-sm font-medium text-gray-900 mb-2">
              {debouncedSearchQuery ? 'No matching emails' : 'No emails yet'}
            </h3>
            <p className="text-sm text-gray-500 mb-4">
              {debouncedSearchQuery 
                ? 'Try adjusting your search criteria or filters.'
                : 'Send your first email through the SMTP relay to see it here.'
              }
            </p>
            {!debouncedSearchQuery && (
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
                {emails.map((email) => (
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
        
        {/* Pagination */}
        {totalPages > 1 && (
          <div className="px-6 py-4 border-t border-gray-100 bg-gray-50/50">
            <div className="flex items-center justify-between">
              <div className="flex items-center space-x-2">
                <button
                  onClick={handlePrevPage}
                  disabled={currentPage === 1}
                  className="inline-flex items-center px-3 py-2 text-sm font-medium text-gray-500 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  <ChevronLeftIcon className="w-4 h-4 mr-1" />
                  Previous
                </button>
                <button
                  onClick={handleNextPage}
                  disabled={!hasMore}
                  className="inline-flex items-center px-3 py-2 text-sm font-medium text-gray-500 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Next
                  <ChevronRightIcon className="w-4 h-4 ml-1" />
                </button>
              </div>
              
              <div className="flex items-center space-x-2">
                <span className="text-sm text-gray-700">
                  Page {currentPage} of {totalPages}
                </span>
                
                {/* Page number buttons for smaller page counts */}
                {totalPages <= 10 && (
                  <div className="flex space-x-1">
                    {Array.from({ length: totalPages }, (_, i) => i + 1).map((page) => (
                      <button
                        key={page}
                        onClick={() => handlePageChange(page)}
                        className={`px-3 py-2 text-sm font-medium rounded-lg ${
                          page === currentPage
                            ? 'bg-blue-600 text-white'
                            : 'text-gray-500 bg-white border border-gray-300 hover:bg-gray-50'
                        }`}
                      >
                        {page}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            </div>
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