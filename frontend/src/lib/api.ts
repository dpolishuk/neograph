import axios from 'axios'

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:3001'

export const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

export interface Repository {
  id: string
  url: string
  name: string
  defaultBranch: string
  status: 'pending' | 'indexing' | 'ready' | 'error'
  filesCount: number
  functionsCount: number
  lastIndexed: string
}

export interface CreateRepositoryInput {
  url: string
  name?: string
  defaultBranch?: string
}

export interface FileNode {
  id: string
  path: string
  language: string
  functions: Array<{
    id: string
    name: string
    signature: string
    startLine: number
    endLine: number
  }>
}

export const repositoryApi = {
  list: async (): Promise<Repository[]> => {
    const { data } = await api.get('/api/repositories')
    return data
  },

  get: async (id: string): Promise<Repository> => {
    const { data } = await api.get(`/api/repositories/${id}`)
    return data
  },

  create: async (input: CreateRepositoryInput): Promise<Repository> => {
    const { data } = await api.post('/api/repositories', input)
    return data
  },

  delete: async (id: string): Promise<void> => {
    await api.delete(`/api/repositories/${id}`)
  },

  reindex: async (id: string): Promise<void> => {
    await api.post(`/api/repositories/${id}/reindex`)
  },

  getFiles: async (id: string): Promise<FileNode[]> => {
    const { data } = await api.get(`/api/repositories/${id}/files`)
    return data
  },

  getGraph: async (id: string, type: 'structure' | 'calls' = 'structure') => {
    const { data } = await api.get(`/api/repositories/${id}/graph?type=${type}`)
    return data
  },

  getNodeDetail: async (repoId: string, nodeId: string): Promise<NodeDetail> => {
    const { data } = await api.get(`/api/repositories/${repoId}/nodes/${nodeId}`)
    return data
  },
}

export interface NodeDetail {
  id: string
  name: string
  type: 'File' | 'Function' | 'Method'
  signature?: string
  filePath?: string
  startLine?: number
  endLine?: number
  calls?: string[]
  calledBy?: string[]
}

export interface SearchResult {
  id: string
  name: string
  signature: string
  filePath: string
  repoId: string
  repoName: string
  score: number
}

export const searchApi = {
  global: async (query: string): Promise<SearchResult[]> => {
    const { data } = await api.get(`/api/search?q=${encodeURIComponent(query)}`)
    return data
  },

  repo: async (repoId: string, query: string): Promise<SearchResult[]> => {
    const { data } = await api.get(
      `/api/repositories/${repoId}/search?q=${encodeURIComponent(query)}`
    )
    return data
  },
}
