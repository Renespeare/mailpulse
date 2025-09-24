import { useState, useEffect } from 'react'
import { 
  PlusIcon,
  EyeIcon,
  EyeSlashIcon,
  TrashIcon,
  PencilIcon,
  KeyIcon,
  ClipboardDocumentIcon,
  CheckIcon
} from '@heroicons/react/24/outline'
import { getProjects, createProject, updateProject, deleteProject, getQuotaUsage, type Project, type QuotaUsage } from '../lib/api'
import CreateProjectModal from '../components/forms/CreateProjectModal'

function Projects() {
  const [projects, setProjects] = useState<Project[]>([])
  const [quotaData, setQuotaData] = useState<Record<string, QuotaUsage>>({})
  const [loading, setLoading] = useState(true)
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [editingProject, setEditingProject] = useState<Project | null>(null)
  const [showPasswords, setShowPasswords] = useState<Record<string, boolean>>({})
  const [copiedField, setCopiedField] = useState<string | null>(null)
  const [deletingProject, setDeletingProject] = useState<Project | null>(null)

  const fetchData = async () => {
    try {
      const projectsData = await getProjects()
      setProjects(projectsData)

      // Fetch quota data for each project
      const quotaPromises = projectsData.map(async (project) => {
        const quota = await getQuotaUsage(project.id)
        return { projectId: project.id, quota }
      })

      const results = await Promise.all(quotaPromises)
      const quotaMap: Record<string, QuotaUsage> = {}
      results.forEach(({ projectId, quota }) => {
        if (quota) quotaMap[projectId] = quota
      })

      setQuotaData(quotaMap)
    } catch (error) {
      console.error('Failed to fetch projects:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  const handleCreateProject = async (projectData: Omit<Project, 'id' | 'createdAt' | 'updatedAt'>) => {
    try {
      await createProject(projectData)
      await fetchData()
      setShowCreateModal(false)
    } catch (error) {
      console.error('Failed to create project:', error)
    }
  }

  const handleUpdateProject = async (projectData: Omit<Project, 'id' | 'createdAt' | 'updatedAt'>) => {
    if (!editingProject) return
    try {
      await updateProject(editingProject.id, projectData)
      await fetchData()
      setEditingProject(null)
    } catch (error) {
      console.error('Failed to update project:', error)
    }
  }

  const handleDeleteProject = (project: Project) => {
    setDeletingProject(project)
  }

  const confirmDeleteProject = async () => {
    if (!deletingProject) return
    
    try {
      await deleteProject(deletingProject.id)
      await fetchData()
      setDeletingProject(null)
    } catch (error) {
      console.error('Failed to delete project:', error)
    }
  }

  const handleToggleStatus = async (projectId: string, currentStatus: string) => {
    const newStatus = currentStatus === 'active' ? 'inactive' : 'active'
    
    try {
      const result = await updateProject(projectId, { status: newStatus })
      if (result.success) {
        await fetchData()
      } else {
        console.error('Failed to update project status:', result.error)
      }
    } catch (error) {
      console.error('Failed to update project status:', error)
    }
  }

  const togglePasswordVisibility = (projectId: string) => {
    setShowPasswords(prev => ({
      ...prev,
      [projectId]: !prev[projectId]
    }))
  }

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'active':
        return 'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800'
      case 'inactive':
        return 'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800'
      default:
        return 'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800'
    }
  }

  const copyToClipboard = async (text: string, fieldName: string) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopiedField(fieldName)
      setTimeout(() => setCopiedField(null), 2000)
    } catch (error) {
      console.error('Failed to copy to clipboard:', error)
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

  if (loading) {
    return (
      <div className="p-8 animate-fade-in">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900 mb-2">Projects</h1>
          <p className="text-gray-600">Loading your SMTP projects...</p>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
          {[...Array(3)].map((_, i) => (
            <div key={i} className="bg-white rounded-xl shadow-sm border border-gray-100 p-6 animate-pulse">
              <div className="h-6 bg-gray-200 rounded mb-2"></div>
              <div className="h-4 bg-gray-200 rounded w-2/3 mb-4"></div>
              <div className="space-y-2">
                {[...Array(3)].map((_, j) => (
                  <div key={j} className="h-3 bg-gray-200 rounded"></div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="p-8 animate-fade-in">
      <div className="mb-8">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 mb-2">Projects</h1>
            <p className="text-gray-600">
              Manage your SMTP relay projects and view their performance
            </p>
          </div>
          
          <button
            onClick={() => setShowCreateModal(true)}
            className="inline-flex items-center px-4 py-2 text-sm font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 transition-colors"
          >
            <PlusIcon className="w-4 h-4 mr-2" />
            New Project
          </button>
        </div>
      </div>

      {/* Security Notice */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6">
        <div className="flex">
          <svg className="h-5 w-5 text-blue-400 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
          </svg>
          <div className="ml-3">
            <p className="text-sm text-blue-700">
              <strong>API Key Security:</strong> Store API keys securely. Each project requires authentication for all SMTP connections.
            </p>
          </div>
        </div>
      </div>

      {projects.length === 0 ? (
        <div className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden">
          <div className="p-6 text-center py-12">
            <KeyIcon className="w-16 h-16 mx-auto text-gray-300 mb-4" />
            <h3 className="text-xl font-medium text-gray-900 mb-2">No projects yet</h3>
            <p className="text-gray-600 mb-6">
              Create your first SMTP project to start sending emails through the relay
            </p>
            <button
              onClick={() => setShowCreateModal(true)}
              className="inline-flex items-center px-6 py-3 text-base font-medium rounded-lg bg-blue-600 text-white hover:bg-blue-700 transition-colors"
            >
              <PlusIcon className="w-5 h-5 mr-2" />
              Create First Project
            </button>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
          {projects.map((project) => {
            const quota = quotaData[project.id]
            const passwordVisible = showPasswords[project.id]
            
            return (
              <div key={project.id} className="bg-white rounded-xl shadow-sm border border-gray-100 overflow-hidden hover:shadow-md transition-shadow">
                <div className="p-6">
                  <div className="flex items-center justify-between mb-4">
                    <h3 className="text-lg font-semibold text-gray-900">{project.name}</h3>
                    <span className={getStatusBadge(project.status)}>{project.status}</span>
                  </div>
                  
                  {project.description && (
                    <p className="text-sm text-gray-600 mb-4">{project.description}</p>
                  )}

                  {/* API Key */}
                  <div className="space-y-3">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">API Key</label>
                      <div className="flex items-center space-x-2">
                        <div className="flex-1 font-mono text-xs bg-gray-50 px-3 py-2 rounded border leading-tight">
                          <div className="break-all">
                            {passwordVisible ? project.apiKey : '••••••••••••••••••••••••••••••••'}
                          </div>
                        </div>
                        <div className="flex items-center space-x-1 flex-shrink-0">
                          <button
                            onClick={() => togglePasswordVisibility(project.id)}
                            className="p-2 text-gray-400 hover:text-gray-600 transition-colors"
                            title={passwordVisible ? 'Hide API key' : 'Show API key'}
                          >
                            {passwordVisible ? (
                              <EyeSlashIcon className="w-4 h-4" />
                            ) : (
                              <EyeIcon className="w-4 h-4" />
                            )}
                          </button>
                          <CopyButton text={project.apiKey} fieldName={`apikey-${project.id}`} />
                        </div>
                      </div>
                    </div>

                    {/* SMTP Configuration */}
                    {project.smtpHost && (
                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">SMTP Configuration</label>
                        <div className="text-sm text-gray-600 bg-gray-50 p-3 rounded">
                          <div>Host: {project.smtpHost}</div>
                          <div>Port: {project.smtpPort}</div>
                          {project.smtpUser && <div>User: {project.smtpUser}</div>}
                        </div>
                      </div>
                    )}
                  </div>

                  {/* Usage Stats */}
                  <div className="grid grid-cols-3 gap-3 mt-6">
                    <div className="text-center">
                      <div className="text-xl font-bold text-blue-600">{quota?.dailyUsed || 0}</div>
                      <div className="text-xs text-gray-500">Sent Today</div>
                    </div>
                    <div className="text-center">
                      <div className="text-xl font-bold text-green-600">{quota?.dailyLimit || project.quotaDaily}</div>
                      <div className="text-xs text-gray-500">Daily Limit</div>
                    </div>
                    <div className="text-center">
                      <div className="text-xl font-bold text-purple-600">{quota?.minuteLimit || project.quotaPerMinute}</div>
                      <div className="text-xs text-gray-500">Per Minute</div>
                    </div>
                  </div>

                  {/* Usage Progress Bar */}
                  {quota && (
                    <div className="mt-6">
                      <div className="flex items-center justify-between mb-1">
                        <span className="text-xs text-gray-500">Daily Usage</span>
                        <span className="text-xs text-gray-500">
                          {quota.dailyUsagePercent < 1 && quota.dailyUsagePercent > 0 
                            ? quota.dailyUsagePercent.toFixed(1) 
                            : Math.round(quota.dailyUsagePercent)
                          }%
                        </span>
                      </div>
                      <div className="w-full bg-gray-200 rounded-full h-2">
                        <div 
                          className={`h-2 rounded-full transition-all ${
                            quota.dailyUsagePercent > 80 ? 'bg-red-500' :
                            quota.dailyUsagePercent > 60 ? 'bg-yellow-500' :
                            'bg-green-500'
                          }`}
                          style={{ width: `${Math.min(quota.dailyUsagePercent, 100)}%` }}
                        ></div>
                      </div>
                    </div>
                  )}

                  {/* Action Buttons */}
                  <div className="space-y-3 pt-6 border-t border-gray-100">
                    <div className="flex items-center justify-between">
                      <button
                        onClick={() => handleToggleStatus(project.id, project.status)}
                        className={`px-3 py-1 text-sm font-medium rounded transition-colors ${
                          project.status === 'active'
                            ? 'bg-yellow-100 text-yellow-800 hover:bg-yellow-200'
                            : 'bg-green-100 text-green-800 hover:bg-green-200'
                        }`}
                      >
                        {project.status === 'active' ? 'Deactivate' : 'Activate'}
                      </button>
                      
                      <div className="flex items-center space-x-2">
                        <button
                          onClick={() => setEditingProject(project)}
                          className="p-2 text-gray-400 hover:text-blue-600 transition-colors"
                          title="Edit project"
                        >
                          <PencilIcon className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleDeleteProject(project)}
                          className="p-2 text-gray-400 hover:text-red-600 transition-colors"
                          title="Delete project"
                        >
                          <TrashIcon className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                    
                    <div className="text-xs text-gray-500">
                      Created {new Date(project.createdAt).toLocaleDateString()}
                    </div>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Create Project Modal */}
      {showCreateModal && (
        <CreateProjectModal
          onClose={() => setShowCreateModal(false)}
          onSubmit={handleCreateProject}
        />
      )}

      {/* Edit Project Modal */}
      {editingProject && (
        <CreateProjectModal
          onClose={() => setEditingProject(null)}
          onSubmit={handleUpdateProject}
          initialData={editingProject}
          isEditing={true}
        />
      )}

      {/* Delete Confirmation Dialog */}
      {deletingProject && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm">
          <div className="fixed left-[50%] top-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4 border bg-white p-6 shadow-lg duration-200 rounded-lg">
            <div className="flex flex-col space-y-1.5 text-center sm:text-left">
              <h2 className="text-lg font-semibold text-gray-900">Delete Confirmation</h2>
              <p className="text-sm text-gray-600">
                Are you sure you want to delete <strong>"{deletingProject.name}"</strong>? This action cannot be undone.
              </p>
            </div>
            <div className="flex flex-col-reverse sm:flex-row sm:justify-end sm:space-x-2">
              <button
                onClick={() => setDeletingProject(null)}
                className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 border border-gray-300 bg-white hover:bg-gray-50 text-gray-700 h-10 px-4 py-2 mt-2 sm:mt-0"
              >
                Cancel
              </button>
              <button
                onClick={confirmDeleteProject}
                className="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 bg-red-600 text-white hover:bg-red-700 h-10 px-4 py-2"
              >
                Yes, Delete
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default Projects