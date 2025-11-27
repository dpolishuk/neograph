import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { repositoryApi } from '@/lib/api'
import { ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'

export default function RepositoryDetailPage() {
  const { id } = useParams<{ id: string }>()

  const { data: repo, isLoading } = useQuery({
    queryKey: ['repository', id],
    queryFn: () => repositoryApi.get(id!),
    enabled: !!id,
  })

  if (isLoading) {
    return <div className="text-center py-8">Loading repository...</div>
  }

  if (!repo) {
    return <div className="text-center py-8">Repository not found</div>
  }

  return (
    <div>
      <div className="flex items-center gap-4 mb-6">
        <Link to="/">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="w-4 h-4 mr-1" /> Back
          </Button>
        </Link>
        <h2 className="text-xl font-semibold">{repo.name}</h2>
      </div>

      <div className="bg-white rounded-lg border p-6">
        <p className="text-gray-600 mb-4">Repository detail view - Coming soon!</p>
        <dl className="space-y-2 text-sm">
          <div>
            <dt className="font-medium text-gray-500">URL:</dt>
            <dd className="text-gray-900">{repo.url}</dd>
          </div>
          <div>
            <dt className="font-medium text-gray-500">Branch:</dt>
            <dd className="text-gray-900">{repo.defaultBranch}</dd>
          </div>
          <div>
            <dt className="font-medium text-gray-500">Status:</dt>
            <dd className="text-gray-900">{repo.status}</dd>
          </div>
        </dl>
      </div>
    </div>
  )
}
