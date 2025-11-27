import { useQuery } from '@tanstack/react-query'
import { repositoryApi, searchApi } from '@/lib/api'
import { ChevronRight, ChevronDown, FileCode, Box, Search } from 'lucide-react'
import { useState } from 'react'
import { Input } from '@/components/ui/input'

interface FileTreeProps {
  repoId: string
  onNodeSelect: (nodeId: string) => void
  onSearchResults?: (nodeIds: string[]) => void
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

export function FileTree({ repoId, onNodeSelect, onSearchResults }: FileTreeProps) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [searchQuery, setSearchQuery] = useState('')

  const { data: files, isLoading } = useQuery({
    queryKey: ['repository-files', repoId],
    queryFn: () => repositoryApi.getFiles(repoId),
  })

  const { data: searchResults, isLoading: isSearching } = useQuery({
    queryKey: ['repository-search', repoId, searchQuery],
    queryFn: () => searchApi.repo(repoId, searchQuery),
    enabled: searchQuery.length > 2,
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

  // Notify parent of search results
  if (searchResults && onSearchResults) {
    const nodeIds = searchResults.map(r => r.id)
    onSearchResults(nodeIds)
  }

  // Filter files/functions based on search
  const searchResultIds = new Set(searchResults?.map(r => r.id) || [])
  const hasActiveSearch = searchQuery.length > 2 && searchResults

  const filteredFiles = hasActiveSearch
    ? files?.map((file: FileNode) => ({
        ...file,
        functions: file.functions.filter(fn => searchResultIds.has(fn.id))
      })).filter((file: FileNode) => file.functions.length > 0 || searchResultIds.has(file.id))
    : files

  if (isLoading) return <div className="p-4">Loading files...</div>

  return (
    <div className="bg-white rounded-lg border overflow-auto flex flex-col">
      <div className="p-3 border-b font-medium text-sm">Files</div>
      <div className="p-2 border-b">
        <div className="relative">
          <Search className="absolute left-2 top-2.5 w-4 h-4 text-gray-400" />
          <Input
            type="text"
            placeholder="Search..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="pl-8 h-9"
          />
        </div>
        {isSearching && (
          <div className="text-xs text-gray-500 mt-1 px-2">Searching...</div>
        )}
      </div>
      <div className="p-2 flex-1 overflow-auto">
        {filteredFiles?.length === 0 && hasActiveSearch && (
          <div className="text-sm text-gray-500 text-center py-4">
            No matches found
          </div>
        )}
        {filteredFiles?.map((file: FileNode) => {
          const isFileMatched = searchResultIds.has(file.id)
          const hasMatchedFunctions = file.functions.some(fn => searchResultIds.has(fn.id))

          return (
            <div key={file.id}>
              <button
                className={`flex items-center gap-1 w-full p-1.5 rounded hover:bg-gray-100 text-left text-sm ${
                  isFileMatched && hasActiveSearch ? 'bg-orange-50 border border-orange-200' : ''
                }`}
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

              {(expanded.has(file.id) || (hasActiveSearch && hasMatchedFunctions)) && (
                <div className="ml-6">
                  {file.functions.map((fn) => {
                    const isFunctionMatched = searchResultIds.has(fn.id)
                    return (
                      <button
                        key={fn.id}
                        className={`flex items-center gap-1 w-full p-1.5 rounded hover:bg-gray-100 text-left text-sm ${
                          isFunctionMatched && hasActiveSearch ? 'bg-orange-50 border border-orange-200 font-medium' : ''
                        }`}
                        onClick={() => onNodeSelect(fn.id)}
                      >
                        <Box className="w-4 h-4 text-green-500" />
                        <span className="truncate">{fn.name}</span>
                      </button>
                    )
                  })}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
