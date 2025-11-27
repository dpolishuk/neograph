import { useQuery } from '@tanstack/react-query'
import { repositoryApi } from '@/lib/api'
import { FileCode, Box, ArrowRight, ArrowLeft } from 'lucide-react'

interface NodeDetailPanelProps {
  nodeId: string | null
  repoId: string
}

export function NodeDetailPanel({ nodeId, repoId }: NodeDetailPanelProps) {
  const { data: nodeDetail, isLoading } = useQuery({
    queryKey: ['node-detail', repoId, nodeId],
    queryFn: () => repositoryApi.getNodeDetail(repoId, nodeId!),
    enabled: !!nodeId,
  })

  if (!nodeId) {
    return (
      <div className="bg-white rounded-lg border p-4 text-gray-500 text-sm">
        Select a node to view details
      </div>
    )
  }

  if (isLoading) return <div className="bg-white rounded-lg border p-4">Loading...</div>

  return (
    <div className="bg-white rounded-lg border overflow-auto">
      <div className="p-3 border-b font-medium text-sm">Details</div>
      <div className="p-3 space-y-4">
        <div>
          <h3 className="text-lg font-medium flex items-center gap-2">
            {nodeDetail?.type === 'File' ? (
              <FileCode className="w-5 h-5 text-blue-500" />
            ) : (
              <Box className="w-5 h-5 text-green-500" />
            )}
            {nodeDetail?.name}
          </h3>
          {nodeDetail?.signature && (
            <code className="text-sm text-gray-600 block mt-1">
              {nodeDetail.signature}
            </code>
          )}
        </div>

        {nodeDetail?.filePath && (
          <div>
            <h4 className="text-sm font-medium text-gray-500">Location</h4>
            <p className="text-sm">
              {nodeDetail.filePath}
              {nodeDetail.startLine && nodeDetail.endLine && (
                <>:{nodeDetail.startLine}-{nodeDetail.endLine}</>
              )}
            </p>
          </div>
        )}

        {nodeDetail?.calls && nodeDetail.calls.length > 0 && (
          <div>
            <h4 className="text-sm font-medium text-gray-500 flex items-center gap-1">
              <ArrowRight className="w-4 h-4" /> Calls
            </h4>
            <ul className="text-sm mt-1 space-y-1">
              {nodeDetail.calls.map((name: string) => (
                <li key={name} className="text-blue-600 hover:underline cursor-pointer">
                  {name}
                </li>
              ))}
            </ul>
          </div>
        )}

        {nodeDetail?.calledBy && nodeDetail.calledBy.length > 0 && (
          <div>
            <h4 className="text-sm font-medium text-gray-500 flex items-center gap-1">
              <ArrowLeft className="w-4 h-4" /> Called By
            </h4>
            <ul className="text-sm mt-1 space-y-1">
              {nodeDetail.calledBy.map((name: string) => (
                <li key={name} className="text-blue-600 hover:underline cursor-pointer">
                  {name}
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </div>
  )
}
