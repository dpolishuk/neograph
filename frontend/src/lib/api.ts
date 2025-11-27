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

export interface AgentChatRequest {
  message: string
  repoId?: string
  agentType?: 'explorer' | 'analyzer' | 'doc_writer'
}

export interface AgentChatResponse {
  response: string
}

export const agentApi = {
  chat: async (
    message: string,
    repoId?: string,
    agentType: 'explorer' | 'analyzer' | 'doc_writer' = 'explorer'
  ): Promise<AgentChatResponse> => {
    const { data } = await api.post('/api/agents/chat', {
      message,
      repo_id: repoId,
      agent_type: agentType,
    })
    return data
  },
}

// Wiki types
export interface WikiNavItem {
  slug: string
  title: string
  order: number
  children?: WikiNavItem[]
}

export interface WikiNavigation {
  items: WikiNavItem[]
}

export interface TOCItem {
  id: string
  title: string
  level: number
}

export interface Diagram {
  id: string
  title: string
  code: string
}

export interface WikiPage {
  id: string
  repoId: string
  slug: string
  title: string
  content: string
  order: number
  parentSlug?: string
  diagrams?: Diagram[]
  tableOfContents?: TOCItem[]
  generatedAt?: string
}

export interface WikiStatus {
  status: 'none' | 'generating' | 'ready' | 'error'
  progress: number
  currentPage?: string
  totalPages?: number
  errorMessage?: string
}

export const wikiApi = {
  getNavigation: async (repoId: string): Promise<WikiNavigation> => {
    const { data } = await api.get(`/api/repositories/${repoId}/wiki`)
    return data
  },

  getPage: async (repoId: string, slug: string): Promise<WikiPage> => {
    const { data } = await api.get(`/api/repositories/${repoId}/wiki/${slug}`)
    return data
  },

  getStatus: async (repoId: string): Promise<WikiStatus> => {
    const { data } = await api.get(`/api/repositories/${repoId}/wiki/status`)
    return data
  },

  generate: async (repoId: string): Promise<void> => {
    await api.post(`/api/repositories/${repoId}/wiki/generate`)
  },
}
