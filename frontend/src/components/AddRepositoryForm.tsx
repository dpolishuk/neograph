import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { repositoryApi } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Plus } from 'lucide-react'

export function AddRepositoryForm() {
  const [url, setUrl] = useState('')
  const [isOpen, setIsOpen] = useState(false)
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: repositoryApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['repositories'] })
      setUrl('')
      setIsOpen(false)
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (url.trim()) {
      mutation.mutate({ url: url.trim() })
    }
  }

  if (!isOpen) {
    return (
      <Button onClick={() => setIsOpen(true)}>
        <Plus className="w-4 h-4 mr-2" />
        Add Repository
      </Button>
    )
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-2">
      <div className="flex gap-2">
        <Input
          type="url"
          placeholder="https://github.com/user/repo"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          className="flex-1"
          autoFocus
        />
        <Button type="submit" disabled={mutation.isPending}>
          {mutation.isPending ? 'Adding...' : 'Add'}
        </Button>
        <Button
          type="button"
          variant="outline"
          onClick={() => {
            setIsOpen(false)
            setUrl('')
            mutation.reset()
          }}
        >
          Cancel
        </Button>
      </div>
      {mutation.error && (
        <p className="text-sm text-red-600">
          {mutation.error instanceof Error ? mutation.error.message : 'Failed to add repository'}
        </p>
      )}
    </form>
  )
}
