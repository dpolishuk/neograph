import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { searchApi } from '@/lib/api'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Search, FileCode } from 'lucide-react'

interface SearchResult {
  id: string
  name: string
  signature: string
  filePath: string
  repoId: string
  repoName: string
  score: number
}

export default function SearchPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const query = searchParams.get('q') || ''
  const [inputValue, setInputValue] = useState(query)

  const { data: results, isLoading } = useQuery({
    queryKey: ['search', query],
    queryFn: () => searchApi.global(query),
    enabled: query.length > 2,
  })

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (inputValue.trim()) {
      setSearchParams({ q: inputValue })
    }
  }

  return (
    <div className="max-w-4xl mx-auto">
      <div className="mb-8">
        <h2 className="text-2xl font-bold mb-4">Search Code</h2>
        <form onSubmit={handleSearch} className="flex gap-2">
          <Input
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            placeholder="Search code across all repositories..."
            className="flex-1"
          />
          <Button type="submit">
            <Search className="w-4 h-4 mr-2" /> Search
          </Button>
        </form>
      </div>

      {isLoading && (
        <div className="text-center text-gray-500 py-8">
          Searching...
        </div>
      )}

      {query.length > 0 && query.length <= 2 && (
        <div className="text-center text-gray-500 py-8">
          Please enter at least 3 characters to search
        </div>
      )}

      {results && results.length === 0 && query.length > 2 && (
        <div className="text-center text-gray-500 py-8">
          No results found for "{query}"
        </div>
      )}

      {results && results.length > 0 && (
        <div className="space-y-4">
          <div className="text-sm text-gray-500 mb-2">
            Found {results.length} result{results.length !== 1 ? 's' : ''}
          </div>
          {results.map((result: SearchResult) => (
            <Link
              key={result.id}
              to={`/repository/${result.repoId}?node=${result.id}`}
            >
              <Card className="hover:shadow-md transition-shadow cursor-pointer">
                <CardHeader className="pb-3">
                  <CardTitle className="flex items-center gap-2 text-base">
                    <FileCode className="w-4 h-4 text-blue-500" />
                    <span className="font-medium">{result.name}</span>
                    <span className="text-gray-400 text-sm font-normal">
                      in {result.repoName}
                    </span>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <code className="text-sm text-gray-600 block mb-2 bg-gray-50 p-2 rounded">
                    {result.signature}
                  </code>
                  <p className="text-sm text-gray-500">{result.filePath}</p>
                  <p className="text-xs text-gray-400 mt-1">
                    Score: {result.score.toFixed(3)}
                  </p>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
