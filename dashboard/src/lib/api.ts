// API client for MailPulse relay
const RELAY_API_URL = import.meta.env.VITE_RELAY_API_URL || 'http://localhost:8080'

export interface QuotaUsage {
  projectId: string
  emailsThisMinute: number
  emailsToday: number
  lastEmailSent?: string
  quotaPerMinute: number
  quotaDaily: number
  minuteUsagePercent: number
  dailyUsagePercent: number
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
    const response = await fetch(`${RELAY_API_URL}/api/quota/${projectId}`)
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
    const response = await fetch(`${RELAY_API_URL}/api/emails/stats/${projectId}`)
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

export async function getRelayHealth(): Promise<any> {
  try {
    const response = await fetch(`${RELAY_API_URL}/health`)
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
      headers: {
        'Content-Type': 'application/json',
      },
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
    const response = await fetch(`${RELAY_API_URL}/api/projects`)
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
      headers: {
        'Content-Type': 'application/json',
      },
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
      headers: {
        'Content-Type': 'application/json',
      },
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

export async function getEmails(projectFilter?: string): Promise<Email[]> {
  try {
    const url = projectFilter 
      ? `${RELAY_API_URL}/api/emails?project=${encodeURIComponent(projectFilter)}`
      : `${RELAY_API_URL}/api/emails`
    
    const response = await fetch(url)
    if (!response.ok) {
      console.error('Failed to fetch emails:', response.statusText)
      return []
    }
    const emails = await response.json()
    
    // Transform Go API field names to match dashboard expectations
    return emails.map((email: any) => ({
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
  } catch (error) {
    console.error('Error fetching emails:', error)
    return []
  }
}