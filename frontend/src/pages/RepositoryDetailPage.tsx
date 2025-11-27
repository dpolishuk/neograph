import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { repositoryApi } from '@/lib/api'
import { ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useState } from 'react'
import { FileTree } from '@/components/FileTree'
import { GraphVisualization } from '@/components/GraphVisualization'
import { NodeDetailPanel } from '@/components/NodeDetailPanel'

export default function RepositoryDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [graphType, setGraphType] = useState<'structure' | 'calls'>('structure')
  const [selectedNode, setSelectedNode] = useState<string | null>(null)

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
    <div className="h-screen flex flex-col">
      <div className="flex items-center gap-4 p-4 border-b bg-white">
        <Link to="/">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="w-4 h-4 mr-1" /> Back
          </Button>
        </Link>
        <h2 className="text-xl font-semibold">{repo.name}</h2>
      </div>

      <div className="flex-1 grid grid-cols-[280px_1fr_300px] gap-4 p-4 overflow-hidden">
        <FileTree repoId={id!} onNodeSelect={setSelectedNode} />
        <GraphVisualization
          repoId={id!}
          type={graphType}
          onTypeChange={setGraphType}
          selectedNode={selectedNode}
          onNodeClick={setSelectedNode}
        />
        <NodeDetailPanel nodeId={selectedNode} repoId={id!} />
      </div>
    </div>
  )
}
