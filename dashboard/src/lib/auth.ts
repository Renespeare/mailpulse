import type { AdminLoginRequest, AdminLoginResponse, AdminVerifyResponse } from '../types/auth'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'

class AuthService {
  private tokenKey = 'mailpulse_admin_token'

  // Login with admin credentials
  async login(credentials: AdminLoginRequest): Promise<AdminLoginResponse> {
    const response = await fetch(`${API_BASE_URL}/api/admin/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(credentials),
    })

    if (!response.ok) {
      const error = await response.text()
      throw new Error(error || 'Login failed')
    }

    const data: AdminLoginResponse = await response.json()
    
    // Store token in localStorage
    localStorage.setItem(this.tokenKey, data.token)
    
    return data
  }

  // Logout (remove token)
  async logout(): Promise<void> {
    // Remove token from localStorage
    localStorage.removeItem(this.tokenKey)
    
    // Call logout endpoint (optional, since JWT is stateless)
    try {
      await fetch(`${API_BASE_URL}/api/admin/logout`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${this.getToken()}`,
        },
      })
    } catch (error) {
      // Ignore errors on logout endpoint
      console.warn('Logout endpoint error:', error)
    }
  }

  // Verify current token
  async verifyToken(): Promise<AdminVerifyResponse | null> {
    const token = this.getToken()
    if (!token) return null

    try {
      const response = await fetch(`${API_BASE_URL}/api/admin/verify`, {
        headers: {
          'Authorization': `Bearer ${token}`,
        },
      })

      if (!response.ok) {
        // Token is invalid, remove it
        this.logout()
        return null
      }

      return await response.json()
    } catch (error) {
      console.error('Token verification error:', error)
      this.logout()
      return null
    }
  }

  // Get stored token
  getToken(): string | null {
    return localStorage.getItem(this.tokenKey)
  }

  // Check if user is authenticated
  isAuthenticated(): boolean {
    const token = this.getToken()
    if (!token) return false

    try {
      // Basic JWT expiration check (decode payload)
      const payload = JSON.parse(atob(token.split('.')[1]))
      const now = Date.now() / 1000
      
      return payload.exp > now
    } catch (error) {
      // If token is malformed, remove it
      this.logout()
      return false
    }
  }

  // Get authorization header
  getAuthHeader(): Record<string, string> {
    const token = this.getToken()
    if (!token) {
      console.warn('No auth token available for API request')
      return {}
    }
    
    return {
      'Authorization': `Bearer ${token}`,
    }
  }
}

export const authService = new AuthService()