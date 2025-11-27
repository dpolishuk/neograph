import { Command } from 'cmdk'
import { useEffect, useState } from 'react'
import { Search, Loader2, MessageSquare } from 'lucide-react'
import { agentApi } from '@/lib/api'
import { Button } from '@/components/ui/button'

interface CommandBarProps {
  onOpenChat?: (initialMessage: string) => void
}

export function CommandBar({ onOpenChat }: CommandBarProps) {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [result, setResult] = useState<string | null>(null)

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        setOpen((open) => !open)
      }
    }
    document.addEventListener('keydown', down)
    return () => document.removeEventListener('keydown', down)
  }, [])

  const handleSubmit = async (value: string) => {
    if (!value.trim() || isLoading) return

    setIsLoading(true)
    setResult(null)

    try {
      const response = await agentApi.chat(value)
      setResult(response.response)
    } catch (error) {
      setResult('Error: Failed to get response from agent')
    } finally {
      setIsLoading(false)
    }
  }

  const handleOpenChat = () => {
    if (onOpenChat && query) {
      onOpenChat(query)
      setOpen(false)
      setQuery('')
      setResult(null)
    }
  }

  const handleSuggestedAction = (action: string) => {
    setQuery(action)
    handleSubmit(action)
  }

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50">
      <div className="fixed inset-0 bg-black/50" onClick={() => setOpen(false)} />
      <div className="fixed top-1/4 left-1/2 -translate-x-1/2 w-full max-w-2xl">
        <Command className="bg-white rounded-lg shadow-xl border">
          <div className="flex items-center border-b px-3">
            <Search className="w-4 h-4 text-gray-400 mr-2" />
            <Command.Input
              value={query}
              onValueChange={setQuery}
              placeholder="Ask about code..."
              className="flex-1 py-3 outline-none text-sm"
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  handleSubmit(query)
                }
              }}
            />
            {isLoading && <Loader2 className="w-4 h-4 animate-spin text-gray-400" />}
          </div>

          {!result && !isLoading && (
            <Command.List className="max-h-80 overflow-auto p-2">
              <Command.Empty className="py-6 text-center text-sm text-gray-500">
                No results found.
              </Command.Empty>
              <Command.Group heading="Suggested actions">
                <Command.Item
                  onSelect={() => handleSuggestedAction('Find authentication code')}
                  className="px-3 py-2 rounded cursor-pointer hover:bg-gray-100 text-sm"
                >
                  Find authentication code
                </Command.Item>
                <Command.Item
                  onSelect={() => handleSuggestedAction('Analyze dependencies')}
                  className="px-3 py-2 rounded cursor-pointer hover:bg-gray-100 text-sm"
                >
                  Analyze dependencies
                </Command.Item>
                <Command.Item
                  onSelect={() => handleSuggestedAction('Generate documentation')}
                  className="px-3 py-2 rounded cursor-pointer hover:bg-gray-100 text-sm"
                >
                  Generate documentation
                </Command.Item>
              </Command.Group>
            </Command.List>
          )}

          {result && (
            <div className="p-4 max-h-80 overflow-auto">
              <div className="bg-gray-50 rounded p-3 text-sm">
                <div className="prose prose-sm max-w-none">
                  {result}
                </div>
              </div>
              <div className="mt-3 flex justify-end">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleOpenChat}
                  className="flex items-center gap-1"
                >
                  <MessageSquare className="w-3 h-3" />
                  Open in chat
                </Button>
              </div>
            </div>
          )}
        </Command>
      </div>
    </div>
  )
}
