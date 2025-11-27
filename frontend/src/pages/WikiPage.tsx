import { useParams, Link, useNavigate } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { repositoryApi, wikiApi } from '@/lib/api'
import { ArrowLeft, RefreshCw, Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { WikiSidebar } from '@/components/WikiSidebar'
import { WikiContent } from '@/components/WikiContent'
import { useEffect, useState } from 'react'

export default function WikiPage() {
  const { id, slug } = useParams<{ id: string; slug?: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [isGenerating, setIsGenerating] = useState(false)

  const { data: repo, isLoading: repoLoading } = useQuery({
    queryKey: ['repository', id],
    queryFn: () => repositoryApi.get(id!),
    enabled: !!id,
  })

  const { data: status, isLoading: statusLoading } = useQuery({
    queryKey: ['wiki-status', id],
    queryFn: () => wikiApi.getStatus(id!),
    enabled: !!id,
    refetchInterval: isGenerating ? 2000 : false,
  })

  useEffect(() => {
    if (status?.status === 'ready' || status?.status === 'error') {
      setIsGenerating(false)
      queryClient.invalidateQueries({ queryKey: ['wiki-navigation', id] })
    }
  }, [status, id, queryClient])

  const handlePageSelect = (selectedSlug: string) => {
    navigate(`/repository/${id}/wiki/${selectedSlug}`)
  }

  const handleGenerate = async () => {
    if (!id) return
    setIsGenerating(true)
    try {
      await wikiApi.generate(id)
      queryClient.invalidateQueries({ queryKey: ['wiki-status', id] })
    } catch (error) {
      setIsGenerating(false)
      console.error('Failed to generate wiki:', error)
    }
  }

  if (repoLoading || statusLoading) {
    return <div className="text-center py-8">Loading...</div>
  }

  if (!repo) {
    return <div className="text-center py-8">Repository not found</div>
  }

  const showGenerateButton = status?.status === 'none' || status?.status === 'error'
  const showGenerating = status?.status === 'generating' || isGenerating

  return (
    <div className="h-screen flex flex-col">
      <div className="flex items-center gap-4 p-4 border-b bg-white">
        <Link to={`/repository/${id}`}>
          <Button variant="ghost" size="sm">
            <ArrowLeft className="w-4 h-4 mr-1" /> Back to Repository
          </Button>
        </Link>
        <h2 className="text-xl font-semibold">{repo.name} - Wiki</h2>
        <div className="ml-auto flex items-center gap-2">
          {showGenerating && (
            <div className="flex items-center gap-2 text-sm text-gray-500">
              <Loader2 className="w-4 h-4 animate-spin" />
              Generating... {status?.progress}%
              {status?.currentPage && ` (${status.currentPage})`}
            </div>
          )}
          {status?.status === 'error' && (
            <div className="text-sm text-red-500">
              Error: {status.errorMessage}
            </div>
          )}
          <Button
            onClick={handleGenerate}
            disabled={showGenerating}
            size="sm"
            variant={showGenerateButton ? 'default' : 'outline'}
          >
            <RefreshCw className={`w-4 h-4 mr-1 ${showGenerating ? 'animate-spin' : ''}`} />
            {showGenerateButton ? 'Generate Wiki' : 'Regenerate'}
          </Button>
        </div>
      </div>

      <div className="flex-1 grid grid-cols-[280px_1fr] gap-4 p-4 overflow-hidden">
        <WikiSidebar
          repoId={id!}
          currentSlug={slug}
          onPageSelect={handlePageSelect}
        />
        <WikiContent repoId={id!} slug={slug || ''} />
      </div>
    </div>
  )
}
