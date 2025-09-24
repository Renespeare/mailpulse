import React, { createContext, useState, useCallback } from 'react'
import { ToastContainer } from '../components/ui/Toast'
import type { ToastMessage, ToastType } from '../types/toast'

interface ToastContextType {
  addToast: (type: ToastType, title: string, message?: string, duration?: number) => void
  removeToast: (id: string) => void
  success: (title: string, message?: string) => void
  error: (title: string, message?: string) => void
  warning: (title: string, message?: string) => void
  info: (title: string, message?: string) => void
}

const ToastContext = createContext<ToastContextType | undefined>(undefined)

// Export context for the hook to use
export { ToastContext }

interface ToastProviderProps {
  children: React.ReactNode
}

export function ToastProvider({ children }: ToastProviderProps) {
  const [toasts, setToasts] = useState<ToastMessage[]>([])

  const addToast = useCallback((type: ToastType, title: string, message?: string, duration?: number) => {
    const id = Math.random().toString(36).substring(2, 11)
    const newToast: ToastMessage = {
      id,
      type,
      title,
      message,
      duration
    }

    setToasts(prev => [...prev, newToast])
  }, [])

  const removeToast = useCallback((id: string) => {
    setToasts(prev => prev.filter(toast => toast.id !== id))
  }, [])

  // Convenience methods
  const success = useCallback((title: string, message?: string) => {
    addToast('success', title, message)
  }, [addToast])

  const error = useCallback((title: string, message?: string) => {
    addToast('error', title, message)
  }, [addToast])

  const warning = useCallback((title: string, message?: string) => {
    addToast('warning', title, message)
  }, [addToast])

  const info = useCallback((title: string, message?: string) => {
    addToast('info', title, message)
  }, [addToast])

  const value: ToastContextType = {
    addToast,
    removeToast,
    success,
    error,
    warning,
    info,
  }

  return (
    <ToastContext.Provider value={value}>
      {children}
      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </ToastContext.Provider>
  )
}