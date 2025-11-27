import { useQuery } from '@tanstack/react-query'
import { repositoryApi } from '@/lib/api'
import { ChevronRight, ChevronDown, FileCode, Box } from 'lucide-react'
import { useState } from 'react'

interface FileTreeProps {
  repoId: string
  onNodeSelect: (nodeId: string) => void
}

interface FileNode {
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

export function FileTree({ repoId, onNodeSelect }: FileTreeProps) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set())

  const { data: files, isLoading } = useQuery({
    queryKey: ['repository-files', repoId],
    queryFn: () => repositoryApi.getFiles(repoId),
  })

  const toggleExpand = (fileId: string) => {
    const next = new Set(expanded)
    if (next.has(fileId)) {
      next.delete(fileId)
    } else {
      next.add(fileId)
    }
    setExpanded(next)
  }

  if (isLoading) return <div className="p-4">Loading files...</div>

  return (
    <div className="bg-white rounded-lg border overflow-auto">
      <div className="p-3 border-b font-medium text-sm">Files</div>
      <div className="p-2">
        {files?.map((file: FileNode) => (
          <div key={file.id}>
            <button
              className="flex items-center gap-1 w-full p-1.5 rounded hover:bg-gray-100 text-left text-sm"
              onClick={() => {
                toggleExpand(file.id)
                onNodeSelect(file.id)
              }}
            >
              {file.functions.length > 0 ? (
                expanded.has(file.id) ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />
              ) : <span className="w-4" />}
              <FileCode className="w-4 h-4 text-blue-500" />
              <span className="truncate">{file.path.split('/').pop()}</span>
            </button>

            {expanded.has(file.id) && (
              <div className="ml-6">
                {file.functions.map((fn) => (
                  <button
                    key={fn.id}
                    className="flex items-center gap-1 w-full p-1.5 rounded hover:bg-gray-100 text-left text-sm"
                    onClick={() => onNodeSelect(fn.id)}
                  >
                    <Box className="w-4 h-4 text-green-500" />
                    <span className="truncate">{fn.name}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
