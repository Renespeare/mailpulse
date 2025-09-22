// API client for MailPulse relay
import { authService } from './auth'

const RELAY_API_URL = import.meta.env.VITE_RELAY_API_URL || 'http://localhost:8080'

// Helper function to get headers with authentication
function getHeaders(includeAuth = true): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  
  if (includeAuth) {
    Object.assign(headers, authService.getAuthHeader())
  }
  
  return headers
}

export interface QuotaUsage {
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

export interface EmailStats {
  projectId: string
  totalEmails: number
  sentEmails: number
  failedEmails: number
  queuedEmails: number
  totalSize: number
  successRate: number
}

export interface Project {
  id: string
  name: string
  description: string
  apiKey: string
  username: string
  password: string
  smtpHost?: string
  smtpPort?: number
  smtpUser?: string
  quotaPerMinute: number
  quotaDaily: number
  createdAt: Date
  updatedAt: Date
  status: string
}

export interface Email {
  id: string
  messageId: string
  projectId: string
  from: string
  to: string[]
  subject: string
  contentEnc?: string | Uint8Array
  size: number
  status: string
  error: string | null
  attempts: number
  sentAt: Date
  openedAt?: Date | null
  clickedAt?: Date | null
  metadata?: Record<string, any>
}

export async function getQuotaUsage(projectId: string): Promise<QuotaUsage | null> {
  try {
    const response = await fetch(`${RELAY_API_URL}/api/quota/${projectId}`, {
      headers: getHeaders()
    })
    if (!response.ok) {
      console.error('Failed to fetch quota usage:', response.statusText)
      return null
    }
    return await response.json()
  } catch (error) {
    console.error('Error fetching quota usage:', error)
    return null
  }
}

export async function getEmailStats(projectId: string): Promise<EmailStats | null> {
  try {
    const response = await fetch(`${RELAY_API_URL}/api/emails/stats/${projectId}`, {
      headers: getHeaders()
    })
    if (!response.ok) {
      console.error('Failed to fetch email stats:', response.statusText)
      return null
    }
    return await response.json()
  } catch (error) {
    console.error('Error fetching email stats:', error)
    return null
  }
}

export async function getAllEmailStats(): Promise<EmailStats | null> {
  try {
    const response = await fetch(`${RELAY_API_URL}/api/emails/stats`, {
      headers: getHeaders()
    })
    if (!response.ok) {
      console.error('Failed to fetch all email stats:', response.statusText)
      return null
    }
    return await response.json()
  } catch (error) {
    console.error('Error fetching all email stats:', error)
    return null
  }
}

export async function getRelayHealth(): Promise<any> {
  try {
    const response = await fetch(`${RELAY_API_URL}/health`, {
      headers: getHeaders(false) // Health check doesn't need auth
    })
    if (!response.ok) {
      return { status: 'unhealthy', message: 'Relay not responding' }
    }
    return await response.json()
  } catch (error) {
    return { status: 'offline', message: 'Relay offline' }
  }
}

export async function resendEmail(emailId: string): Promise<{ success: boolean; message: string }> {
  try {
    const response = await fetch(`${RELAY_API_URL}/api/emails/${emailId}/resend`, {
      method: 'POST',
      headers: getHeaders(),
    })
    
    if (!response.ok) {
      const errorText = await response.text()
      return { success: false, message: errorText || 'Failed to resend email' }
    }
    
    const result = await response.json()
    return { success: true, message: result.message || 'Email queued for resend' }
  } catch (error) {
    console.error('Error resending email:', error)
    return { success: false, message: 'Network error' }
  }
}

export async function getProjects(): Promise<Project[]> {
  try {
    const response = await fetch(`${RELAY_API_URL}/api/projects`, {
      headers: getHeaders()
    })
    if (!response.ok) {
      console.error('Failed to fetch projects:', response.statusText)
      return []
    }
    const projects = await response.json()
    
    // Handle null or undefined response
    if (!projects || !Array.isArray(projects)) {
      return []
    }
    
    // Transform Go API field names to match dashboard expectations
    return projects.map((project: any) => ({
      id: project.ID,
      name: project.Name,
      description: project.Description || '',
      apiKey: project.APIKey,
      username: project.Username || project.APIKey, // Use APIKey as username fallback
      password: project.Password || project.APIKey, // Use APIKey as password fallback
      smtpHost: project.SMTPHost,
      smtpPort: project.SMTPPort,
      smtpUser: project.SMTPUser,
      quotaPerMinute: project.QuotaPerMinute,
      quotaDaily: project.QuotaDaily,
      createdAt: new Date(project.CreatedAt),
      updatedAt: new Date(project.UpdatedAt || project.CreatedAt),
      status: project.Status
    }))
  } catch (error) {
    console.error('Error fetching projects:', error)
    return []
  }
}

export async function createProject(projectData: Partial<Project>): Promise<{ success: boolean; data?: Project; error?: string }> {
  try {
    const response = await fetch(`${RELAY_API_URL}/api/projects`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify(projectData),
    })
    
    if (!response.ok) {
      const errorText = await response.text()
      return { success: false, error: errorText }
    }
    
    const result = await response.json()
    return { success: true, data: result }
  } catch (error) {
    console.error('Error creating project:', error)
    return { success: false, error: 'Network error' }
  }
}

export async function deleteProject(projectId: string): Promise<{ success: boolean; error?: string }> {
  try {
    const response = await fetch(`${RELAY_API_URL}/api/projects/${projectId}`, {
      method: 'DELETE',
      headers: getHeaders(),
    })
    
    if (!response.ok) {
      const errorText = await response.text()
      return { success: false, error: errorText }
    }
    
    return { success: true }
  } catch (error) {
    console.error('Error deleting project:', error)
    return { success: false, error: 'Network error' }
  }
}

export async function updateProject(projectId: string, updates: Partial<Project>): Promise<{ success: boolean; data?: Project; error?: string }> {
  try {
    const response = await fetch(`${RELAY_API_URL}/api/projects/${projectId}`, {
      method: 'PATCH',
      headers: getHeaders(),
      body: JSON.stringify(updates),
    })
    
    if (!response.ok) {
      const errorText = await response.text()
      return { success: false, error: errorText }
    }
    
    const result = await response.json()
    return { success: true, data: result }
  } catch (error) {
    console.error('Error updating project:', error)
    return { success: false, error: 'Network error' }
  }
}

export interface EmailsResponse {
  emails: Email[]
  totalCount: number
  limit: number
  offset: number
  hasMore: boolean
}

export async function getEmails(
  projectFilter?: string, 
  searchQuery?: string, 
  limit: number = 20, 
  offset: number = 0,
  statusFilter?: string
): Promise<EmailsResponse> {
  try {
    const params = new URLSearchParams()
    if (projectFilter) params.set('project', projectFilter)
    if (searchQuery) params.set('search', searchQuery)
    if (statusFilter && statusFilter !== 'all') params.set('status', statusFilter)
    params.set('limit', limit.toString())
    params.set('offset', offset.toString())
    
    const url = `${RELAY_API_URL}/api/emails?${params.toString()}`
    
    const response = await fetch(url, {
      headers: getHeaders()
    })
    if (!response.ok) {
      console.error('Failed to fetch emails:', response.statusText)
      return { emails: [], totalCount: 0, limit, offset, hasMore: false }
    }
    
    const data = await response.json()
    
    // Handle null/invalid response from API
    if (!data || !Array.isArray(data.emails)) {
      return { emails: [], totalCount: 0, limit, offset, hasMore: false }
    }
    
    // Transform Go API field names to match dashboard expectations
    const transformedEmails = data.emails.map((email: any) => ({
      id: email.ID,
      messageId: email.MessageID,
      projectId: email.ProjectID,
      from: email.From,
      to: email.To,
      subject: email.Subject,
      contentEnc: email.ContentEnc,
      size: email.Size,
      status: email.Status,
      error: email.Error,
      attempts: email.Attempts,
      sentAt: new Date(email.SentAt),
      openedAt: email.OpenedAt ? new Date(email.OpenedAt) : null,
      clickedAt: email.ClickedAt ? new Date(email.ClickedAt) : null,
      metadata: email.Metadata
    }))
    
    return {
      emails: transformedEmails,
      totalCount: data.totalCount || 0,
      limit: data.limit || limit,
      offset: data.offset || offset,
      hasMore: data.hasMore || false
    }
  } catch (error) {
    console.error('Error fetching emails:', error)
    return { emails: [], totalCount: 0, limit, offset, hasMore: false }
  }
}