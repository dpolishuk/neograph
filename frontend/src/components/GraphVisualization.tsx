import { useEffect, useRef } from 'react'
import { useQuery } from '@tanstack/react-query'
import { repositoryApi } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Network } from 'vis-network/standalone'
import { DataSet } from 'vis-data/standalone'

interface GraphVisualizationProps {
  repoId: string
  type: 'structure' | 'calls'
  onTypeChange: (type: 'structure' | 'calls') => void
  selectedNode: string | null
  onNodeClick: (nodeId: string) => void
}

interface GraphData {
  nodes: Array<{
    id: string
    label: string
    type: string
    props: Record<string, any>
  }>
  edges: Array<{
    id: string
    source: string
    target: string
    type: string
  }>
}

export function GraphVisualization({
  repoId,
  type,
  onTypeChange,
  selectedNode,
  onNodeClick,
}: GraphVisualizationProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const networkRef = useRef<Network | null>(null)

  const { data: graphData, isLoading } = useQuery<GraphData>({
    queryKey: ['repository-graph', repoId, type],
    queryFn: () => repositoryApi.getGraph(repoId, type),
  })

  useEffect(() => {
    if (!containerRef.current || !graphData) return

    // Prepare nodes with colors based on type
    const nodes = graphData.nodes.map((n) => ({
      id: n.id,
      label: n.label,
      color: n.type === 'File' ? '#3b82f6' : '#22c55e',
      shape: n.type === 'File' ? 'box' : 'ellipse',
      font: {
        color: '#333333',
        size: 14,
      },
    }))

    // Prepare edges
    const edges = graphData.edges.map((e) => ({
      id: e.id,
      from: e.source,
      to: e.target,
      arrows: 'to',
      label: e.type,
      font: {
        size: 10,
        align: 'middle',
      },
    }))

    // Create vis-network datasets
    const nodesDS = new DataSet(nodes)
    const edgesDS = new DataSet(edges)

    // Destroy existing network if it exists
    if (networkRef.current) {
      networkRef.current.destroy()
    }

    // Create new network
    networkRef.current = new Network(
      containerRef.current,
      { nodes: nodesDS, edges: edgesDS },
      {
        physics: {
          stabilization: {
            iterations: 100,
          },
          barnesHut: {
            gravitationalConstant: -2000,
            springLength: 100,
          },
        },
        interaction: {
          hover: true,
          navigationButtons: true,
          keyboard: true,
        },
        layout: {
          improvedLayout: true,
        },
      }
    )

    // Handle click events on nodes
    networkRef.current.on('click', (params) => {
      if (params.nodes.length > 0) {
        onNodeClick(params.nodes[0])
      }
    })

    return () => {
      if (networkRef.current) {
        networkRef.current.destroy()
        networkRef.current = null
      }
    }
  }, [graphData, onNodeClick])

  // Highlight selected node
  useEffect(() => {
    if (networkRef.current && selectedNode) {
      networkRef.current.selectNodes([selectedNode])
      networkRef.current.focus(selectedNode, {
        scale: 1.2,
        animation: {
          duration: 500,
          easingFunction: 'easeInOutQuad',
        },
      })
    }
  }, [selectedNode])

  return (
    <div className="bg-white rounded-lg border flex flex-col">
      <div className="p-3 border-b flex items-center justify-between">
        <span className="font-medium text-sm">Graph</span>
        <div className="flex gap-1">
          <Button
            variant={type === 'structure' ? 'default' : 'outline'}
            size="sm"
            onClick={() => onTypeChange('structure')}
          >
            Structure
          </Button>
          <Button
            variant={type === 'calls' ? 'default' : 'outline'}
            size="sm"
            onClick={() => onTypeChange('calls')}
          >
            Calls
          </Button>
        </div>
      </div>
      <div ref={containerRef} className="flex-1 min-h-[400px]">
        {isLoading && (
          <div className="flex items-center justify-center h-full p-4 text-gray-500">
            Loading graph...
          </div>
        )}
      </div>
    </div>
  )
}
