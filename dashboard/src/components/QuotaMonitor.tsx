import { useState, useEffect } from 'react'
import { ExclamationTriangleIcon, CheckCircleIcon } from '@heroicons/react/24/outline'

interface QuotaData {
  projectId: string
  dailyUsed: number
  dailyLimit: number
  dailyRemaining: number
  minuteUsed: number
  minuteLimit: number
  minuteRemaining: number
  dailyUsagePercent: number
  minuteUsagePercent: number
}

interface QuotaMonitorProps {
  projectId: string
  refreshTrigger?: number
}

function QuotaMonitor({ projectId, refreshTrigger }: QuotaMonitorProps) {
  const [quotaData, setQuotaData] = useState<QuotaData | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchQuotaData = async () => {
    if (!projectId) return
    
    try {
      const response = await fetch(`http://localhost:8080/api/quota/${projectId}`)
      if (!response.ok) {
        throw new Error('Failed to fetch quota data')
      }
      const data = await response.json()
      setQuotaData(data)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setIsLoading(false)
    }
  }

  // Initial fetch on mount/project change and when refresh is triggered
  useEffect(() => {
    fetchQuotaData()
  }, [projectId, refreshTrigger])

  if (isLoading) {
    return (
      <div className="bg-white rounded-lg shadow p-6 animate-pulse">
        <div className="h-4 bg-gray-200 rounded w-1/4 mb-4"></div>
        <div className="space-y-4">
          <div className="h-2 bg-gray-200 rounded"></div>
          <div className="h-2 bg-gray-200 rounded"></div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-white rounded-lg shadow p-6 border-l-4 border-red-400">
        <div className="flex">
          <ExclamationTriangleIcon className="h-5 w-5 text-red-400" />
          <div className="ml-3">
            <h3 className="text-sm font-medium text-red-800">Quota Monitoring Error</h3>
            <div className="mt-2 text-sm text-red-700">
              {error}
            </div>
            <button
              onClick={() => window.location.reload()}
              className="mt-3 text-sm bg-red-100 text-red-800 px-3 py-1 rounded hover:bg-red-200"
            >
              Retry
            </button>
          </div>
        </div>
      </div>
    )
  }

  if (!quotaData) {
    return null
  }

  const getProgressBarColor = (percentage: number) => {
    if (percentage >= 90) return 'bg-red-500'
    if (percentage >= 75) return 'bg-yellow-500'
    return 'bg-green-500'
  }

  const getStatusColor = (percentage: number) => {
    if (percentage >= 90) return 'text-red-600'
    if (percentage >= 75) return 'text-yellow-600'
    return 'text-green-600'
  }

  return (
    <div className="bg-white rounded-lg shadow">
      <div className="p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-medium text-gray-900">Quota Usage</h3>
          <div className="flex items-center">
            {quotaData.dailyUsagePercent < 90 && quotaData.minuteUsagePercent < 90 ? (
              <CheckCircleIcon className="h-5 w-5 text-green-500 mr-2" />
            ) : (
              <ExclamationTriangleIcon className="h-5 w-5 text-red-500 mr-2" />
            )}
            <span className={`text-sm font-medium ${getStatusColor(Math.max(quotaData.dailyUsagePercent, quotaData.minuteUsagePercent))}`}>
              {quotaData.dailyUsagePercent < 90 && quotaData.minuteUsagePercent < 90 ? 'Normal' : 'Warning'}
            </span>
          </div>
        </div>

        <div className="space-y-6">
          {/* Daily Quota */}
          <div>
            <div className="flex justify-between items-center mb-2">
              <span className="text-sm font-medium text-gray-700">Daily Quota</span>
              <span className="text-sm text-gray-500">
                {quotaData.dailyUsed} / {quotaData.dailyLimit} emails
              </span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-2">
              <div
                className={`h-2 rounded-full transition-all duration-300 ${getProgressBarColor(quotaData.dailyUsagePercent || 0)}`}
                style={{ width: `${Math.min(quotaData.dailyUsagePercent || 0, 100)}%` }}
              ></div>
            </div>
            <div className="flex justify-between mt-1">
              <span className="text-xs text-gray-500">
                {quotaData.dailyRemaining} remaining
              </span>
              <span className={`text-xs font-medium ${getStatusColor(quotaData.dailyUsagePercent || 0)}`}>
                {(quotaData.dailyUsagePercent || 0).toFixed(1)}%
              </span>
            </div>
          </div>

          {/* Per-Minute Quota */}
          <div>
            <div className="flex justify-between items-center mb-2">
              <span className="text-sm font-medium text-gray-700">Per-Minute Rate</span>
              <span className="text-sm text-gray-500">
                {quotaData.minuteUsed} / {quotaData.minuteLimit} emails
              </span>
            </div>
            <div className="w-full bg-gray-200 rounded-full h-2">
              <div
                className={`h-2 rounded-full transition-all duration-300 ${getProgressBarColor(quotaData.minuteUsagePercent || 0)}`}
                style={{ width: `${Math.min(quotaData.minuteUsagePercent || 0, 100)}%` }}
              ></div>
            </div>
            <div className="flex justify-between mt-1">
              <span className="text-xs text-gray-500">
                {quotaData.minuteRemaining} remaining
              </span>
              <span className={`text-xs font-medium ${getStatusColor(quotaData.minuteUsagePercent || 0)}`}>
                {(quotaData.minuteUsagePercent || 0).toFixed(1)}%
              </span>
            </div>
          </div>

          {/* Status Messages */}
          {(quotaData.dailyUsagePercent || 0) >= 90 && (
            <div className="bg-red-50 border border-red-200 rounded-md p-3">
              <p className="text-sm text-red-700">
                <strong>Daily quota warning:</strong> You've used {(quotaData.dailyUsagePercent || 0).toFixed(1)}% of your daily email limit.
              </p>
            </div>
          )}
          
          {(quotaData.minuteUsagePercent || 0) >= 90 && (
            <div className="bg-red-50 border border-red-200 rounded-md p-3">
              <p className="text-sm text-red-700">
                <strong>Rate limit warning:</strong> You're approaching the per-minute email limit.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default QuotaMonitor