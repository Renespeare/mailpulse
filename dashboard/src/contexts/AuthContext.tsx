import React, { createContext, useContext, useEffect, useState } from 'react'
import { authService } from '../lib/auth'
import type { AdminLoginRequest, AdminVerifyResponse } from '../types/auth'

interface AuthContextType {
  isAuthenticated: boolean
  isLoading: boolean
  user: AdminVerifyResponse | null
  login: (credentials: AdminLoginRequest) => Promise<void>
  logout: (showToast?: boolean) => void
  checkAuth: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export function useAuth() {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

interface AuthProviderProps {
  children: React.ReactNode
  onSessionExpired?: () => void
}

export function AuthProvider({ children, onSessionExpired }: AuthProviderProps) {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [isLoading, setIsLoading] = useState(true)
  const [user, setUser] = useState<AdminVerifyResponse | null>(null)

  const checkAuth = async () => {
    try {
      setIsLoading(true)
      
      // Quick check if token exists and isn't expired
      if (!authService.isAuthenticated()) {
        setIsAuthenticated(false)
        setUser(null)
        return
      }

      // Verify token with server
      const userData = await authService.verifyToken()
      if (userData) {
        setIsAuthenticated(true)
        setUser(userData)
      } else {
        // Token is invalid - check if user was previously authenticated (session expired)
        const wasAuthenticated = isAuthenticated
        setIsAuthenticated(false)
        setUser(null)
        
        if (wasAuthenticated && onSessionExpired) {
          onSessionExpired() // Show session expired toast
        }
      }
    } catch (error) {
      console.error('Auth check failed:', error)
      setIsAuthenticated(false)
      setUser(null)
    } finally {
      setIsLoading(false)
    }
  }

  const login = async (credentials: AdminLoginRequest) => {
    try {
      setIsLoading(true)
      await authService.login(credentials)
      
      // Verify the new token
      await checkAuth()
    } catch (error) {
      // Don't reset auth state on failed login - user is already not authenticated
      // This prevents LoginForm from re-mounting and losing form data
      throw error // Re-throw so login component can handle it
    } finally {
      setIsLoading(false)
    }
  }

  const logout = (showToast = false) => {
    authService.logout()
    setIsAuthenticated(false)
    setUser(null)
    
    if (showToast && onSessionExpired) {
      onSessionExpired()
    }
  }

  // Check authentication on mount and when token changes
  useEffect(() => {
    checkAuth()
  }, [])

  // Periodically check if token is still valid
  useEffect(() => {
    if (isAuthenticated) {
      const interval = setInterval(() => {
        if (!authService.isAuthenticated()) {
          logout(true) // Show toast when auto-logout due to expired token
        }
      }, 60000) // Check every minute

      return () => clearInterval(interval)
    }
  }, [isAuthenticated])

  const value: AuthContextType = {
    isAuthenticated,
    isLoading,
    user,
    login,
    logout,
    checkAuth,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}