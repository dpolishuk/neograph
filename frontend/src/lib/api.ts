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
}
