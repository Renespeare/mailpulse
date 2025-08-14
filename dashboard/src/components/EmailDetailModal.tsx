import { useState } from 'react'
import { 
  XMarkIcon,
  ClipboardDocumentIcon,
  CheckIcon,
  ArrowPathIcon
} from '@heroicons/react/24/outline'
import { resendEmail, type Email } from '../lib/api'

interface EmailDetailModalProps {
  email: Email
  onClose: () => void
  projectName: string
}

function EmailDetailModal({ email, onClose, projectName }: EmailDetailModalProps) {
  const [activeTab, setActiveTab] = useState<'content' | 'headers' | 'raw'>('content')
  const [copiedField, setCopiedField] = useState<string | null>(null)
  const [resending, setResending] = useState(false)

  const copyToClipboard = async (text: string, fieldName: string) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopiedField(fieldName)
      setTimeout(() => setCopiedField(null), 2000)
    } catch (error) {
      console.error('Failed to copy to clipboard:', error)
    }
  }

  const handleResend = async () => {
    setResending(true)
    try {
      await resendEmail(email.id)
      // You might want to show a success message or refresh the parent list
    } catch (error) {
      console.error('Failed to resend email:', error)
    } finally {
      setResending(false)
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

  const CopyButton = ({ text, fieldName }: { text: string; fieldName: string }) => (
    <button
      onClick={() => copyToClipboard(text, fieldName)}
      className="ml-2 p-1 text-gray-400 hover:text-gray-600 transition-colors"
      title="Copy to clipboard"
    >
      {copiedField === fieldName ? (
        <CheckIcon className="w-4 h-4 text-green-500" />
      ) : (
        <ClipboardDocumentIcon className="w-4 h-4" />
      )}
    </button>
  )

  const parseEmailBody = (contentEnc?: string | Uint8Array) => {
    if (!contentEnc) return null
    
    // Step 1: Handle Uint8Array conversion
    let content = typeof contentEnc === 'string' 
      ? contentEnc 
      : new TextDecoder().decode(contentEnc)
    
    // Step 2: Base64 decoding detection and handling
    try {
      if (content.match(/^[A-Za-z0-9+/]+=*$/)) {
        const decoded = atob(content)
        content = decoded
      }
    } catch (e) {
      console.log('Debug - Not Base64 or decode failed, using as-is')
    }
    
    // Step 3: SMTP message parsing
    const lines = content.split('\n').map(line => line.replace(/\r$/, ''))
    
    // Step 4: Header detection and body extraction
    let lastHeaderIndex = -1
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i].trim()
      if (line.match(/^(Subject|From|To|Date|Content-Type|Content-Transfer-Encoding|MIME-Version):/i)) {
        lastHeaderIndex = i
      }
    }
    
    // Step 5: Clean body extraction with SMTP terminator removal
    if (lastHeaderIndex !== -1 && lastHeaderIndex + 1 < lines.length) {
      let bodyLines = lines.slice(lastHeaderIndex + 1)
      
      // Remove empty lines at start
      while (bodyLines.length > 0 && bodyLines[0].trim() === '') {
        bodyLines = bodyLines.slice(1)
      }
      
      // Remove SMTP terminators ("." and empty lines)
      while (bodyLines.length > 0) {
        const lastLine = bodyLines[bodyLines.length - 1].trim()
        if (lastLine === '.' || lastLine === '') {
          bodyLines = bodyLines.slice(0, -1)
        } else {
          break
        }
      }
      
      const body = bodyLines.join('\n').trim()
      return body || null
    }
    
    return content
  }

  return (
    <div className="fixed inset-0 bg-transparent flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-xl shadow-2xl w-full max-w-4xl h-[90vh] flex flex-col animate-slide-up">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200">
          <div className="flex-1">
            <h2 className="text-xl font-semibold text-gray-900 mb-1">Email Details</h2>
            <p className="text-sm text-gray-500">
              {projectName} • Sent {new Date(email.sentAt).toLocaleString()}
            </p>
          </div>
          <div className="flex items-center space-x-3">
            <span className={getStatusBadge(email.status)}>{email.status}</span>
            {(email.status === 'failed' || email.status === 'queued') && (
              <button
                onClick={handleResend}
                disabled={resending}
                className="inline-flex items-center px-3 py-1.5 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 transition-colors disabled:opacity-50"
              >
                {resending ? (
                  <div className="loading-spinner mr-2"></div>
                ) : (
                  <ArrowPathIcon className="w-4 h-4 mr-2" />
                )}
                Resend
              </button>
            )}
            <button
              onClick={onClose}
              className="p-2 text-gray-400 hover:text-gray-600 transition-colors"
            >
              <XMarkIcon className="w-6 h-6" />
            </button>
          </div>
        </div>

        {/* Email Metadata */}
        <div className="p-6 border-b border-gray-200 bg-gray-50/50">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
            <div>
              <label className="font-medium text-gray-700">From</label>
              <div className="flex items-center mt-1">
                <span className="text-gray-900">{email.from}</span>
                <CopyButton text={email.from} fieldName="from" />
              </div>
            </div>
            <div>
              <label className="font-medium text-gray-700">To</label>
              <div className="flex items-center mt-1">
                <span className="text-gray-900">
                  {Array.isArray(email.to) ? email.to.join(', ') : email.to}
                </span>
                <CopyButton 
                  text={Array.isArray(email.to) ? email.to.join(', ') : email.to} 
                  fieldName="to" 
                />
              </div>
            </div>
            <div>
              <label className="font-medium text-gray-700">Subject</label>
              <div className="flex items-center mt-1">
                <span className="text-gray-900">{email.subject}</span>
                <CopyButton text={email.subject} fieldName="subject" />
              </div>
            </div>
            <div>
              <label className="font-medium text-gray-700">Size</label>
              <div className="mt-1">
                <span className="text-gray-900">{Math.round(email.size / 1024)} KB</span>
              </div>
            </div>
          </div>

          {email.error && (
            <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg">
              <div className="text-sm font-medium text-red-800 mb-1">Error Message</div>
              <div className="text-sm text-red-700 font-mono">{email.error}</div>
            </div>
          )}
        </div>

        {/* Tabs */}
        <div className="border-b border-gray-200">
          <nav className="flex space-x-8 px-6">
            {[
              { key: 'content', label: 'Content' },
              { key: 'headers', label: 'Headers' },
              { key: 'raw', label: 'Raw Message' }
            ].map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key as any)}
                className={`py-4 px-1 border-b-2 font-medium text-sm transition-colors ${
                  activeTab === tab.key
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </nav>
        </div>

        {/* Tab Content */}
        <div className="flex-1 overflow-hidden">
          <div className="h-full overflow-y-auto custom-scrollbar">
            {activeTab === 'content' && (
              <div className="p-6">
                {email.contentEnc ? (
                  <div className="space-y-4">
                    {/* Parsed Email Body */}
                    {parseEmailBody(email.contentEnc) && (
                      <div>
                        <h3 className="font-medium text-gray-900 mb-2">Email Body</h3>
                        <div className="bg-white border border-gray-200 p-4 rounded-lg text-sm text-gray-800 leading-relaxed whitespace-pre-wrap">
                          {parseEmailBody(email.contentEnc)}
                        </div>
                      </div>
                    )}
                  </div>
                ) : (
                  <div className="text-center py-12 text-gray-500">
                    <p>No content available for this email.</p>
                  </div>
                )}
              </div>
            )}

            {activeTab === 'headers' && (
              <div className="p-6">
                {email.metadata && Object.keys(email.metadata).length > 0 ? (
                  <div className="space-y-3">
                    {Object.entries(email.metadata).map(([key, value]) => (
                      <div key={key} className="border-b border-gray-100 pb-3">
                        <div className="flex items-center justify-between">
                          <span className="font-medium text-gray-700 text-sm uppercase tracking-wide">
                            {key}
                          </span>
                          <CopyButton text={String(value)} fieldName={key} />
                        </div>
                        <div className="mt-1 text-sm text-gray-900 font-mono break-all">
                          {String(value)}
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-center py-12 text-gray-500">
                    <p>No metadata available for this email.</p>
                  </div>
                )}
              </div>
            )}

            {activeTab === 'raw' && (
              <div className="p-6">
                {email.contentEnc ? (
                  <div>
                    <div className="flex items-center justify-between mb-3">
                      <h3 className="font-medium text-gray-900">Raw Message</h3>
                      <CopyButton 
                        text={typeof email.contentEnc === 'string' 
                          ? email.contentEnc 
                          : new TextDecoder().decode(email.contentEnc)} 
                        fieldName="raw" 
                      />
                    </div>
                    <pre className="bg-gray-900 text-green-400 p-4 rounded-lg text-xs overflow-x-auto whitespace-pre-wrap font-mono">
                      {typeof email.contentEnc === 'string' 
                        ? email.contentEnc 
                        : new TextDecoder().decode(email.contentEnc)}
                    </pre>
                  </div>
                ) : (
                  <div className="text-center py-12 text-gray-500">
                    <p>Raw message not available for this email.</p>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

export default EmailDetailModal