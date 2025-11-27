import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { repositoryApi, Repository } from '@/lib/api'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Trash2, RefreshCw, GitBranch, FileCode, Box } from 'lucide-react'

function StatusBadge({ status }: { status: Repository['status'] }) {
  const variants: Record<Repository['status'], 'default' | 'success' | 'warning' | 'destructive'> = {
    pending: 'default',
    indexing: 'warning',
    ready: 'success',
    error: 'destructive',
  }

  return <Badge variant={variants[status]}>{status}</Badge>
}

function RepositoryCard({ repo }: { repo: Repository }) {
  const queryClient = useQueryClient()

  const deleteMutation = useMutation({
    mutationFn: () => repositoryApi.delete(repo.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
    },
  })

  const reindexMutation = useMutation({
    mutationFn: () => repositoryApi.reindex(repo.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
    },
  })

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-base font-medium">{repo.name}</CardTitle>
        <StatusBadge status={repo.status} />
      </CardHeader>
      <CardContent>
        <p className="text-sm text-gray-500 mb-3 truncate">{repo.url}</p>

        <div className="flex gap-4 text-sm text-gray-600 mb-4">
          <span className="flex items-center gap-1">
            <GitBranch className="w-4 h-4" />
            {repo.defaultBranch}
          </span>
          <span className="flex items-center gap-1">
            <FileCode className="w-4 h-4" />
            {repo.filesCount} files
          </span>
          <span className="flex items-center gap-1">
            <Box className="w-4 h-4" />
            {repo.functionsCount} functions
          </span>
        </div>

        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => reindexMutation.mutate()}
            disabled={reindexMutation.isPending || repo.status === 'indexing'}
          >
            <RefreshCw className={`w-4 h-4 mr-1 ${reindexMutation.isPending ? 'animate-spin' : ''}`} />
            Reindex
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={() => {
              if (confirm('Delete this repository?')) {
                deleteMutation.mutate()
              }
            }}
            disabled={deleteMutation.isPending}
          >
            <Trash2 className="w-4 h-4 mr-1" />
            Delete
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

export function RepositoryList() {
  const { data: repositories, isLoading, error } = useQuery({
    queryKey: ['repositories'],
    queryFn: repositoryApi.list,
    refetchInterval: 5000, // Poll for status updates
  })

  if (isLoading) {
    return <div className="text-center py-8 text-gray-500">Loading repositories...</div>
  }

  if (error) {
    return (
      <div className="text-center py-8 text-red-500">
        Error loading repositories. Is the backend running?
      </div>
    )
  }

  if (!repositories?.length) {
    return (
      <div className="text-center py-8 text-gray-500">
        No repositories yet. Add one to get started!
      </div>
    )
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {repositories.map((repo) => (
        <RepositoryCard key={repo.id} repo={repo} />
      ))}
    </div>
  )
}
