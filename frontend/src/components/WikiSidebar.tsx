import { useQuery } from '@tanstack/react-query'
import { wikiApi, WikiNavItem } from '@/lib/api'
import { ChevronRight, ChevronDown, FileText, Book } from 'lucide-react'
import { useState } from 'react'

interface WikiSidebarProps {
  repoId: string
  currentSlug?: string
  onPageSelect: (slug: string) => void
}

function NavItem({
  item,
  currentSlug,
  onPageSelect,
  expanded,
  toggleExpand,
  level = 0,
}: {
  item: WikiNavItem
  currentSlug?: string
  onPageSelect: (slug: string) => void
  expanded: Set<string>
  toggleExpand: (slug: string) => void
  level?: number
}) {
  const hasChildren = item.children && item.children.length > 0
  const isExpanded = expanded.has(item.slug)
  const isActive = item.slug === currentSlug

  return (
    <div>
      <button
        className={`flex items-center gap-1 w-full p-2 rounded hover:bg-gray-100 text-left text-sm ${
          isActive ? 'bg-blue-50 border border-blue-200 font-medium' : ''
        }`}
        style={{ paddingLeft: `${level * 12 + 8}px` }}
        onClick={() => {
          if (hasChildren) {
            toggleExpand(item.slug)
          }
          onPageSelect(item.slug)
        }}
      >
        {hasChildren ? (
          isExpanded ? (
            <ChevronDown className="w-4 h-4 flex-shrink-0" />
          ) : (
            <ChevronRight className="w-4 h-4 flex-shrink-0" />
          )
        ) : (
          <span className="w-4" />
        )}
        <FileText className="w-4 h-4 text-blue-500 flex-shrink-0" />
        <span className="truncate">{item.title}</span>
      </button>

      {hasChildren && isExpanded && (
        <div>
          {item.children!.map((child) => (
            <NavItem
              key={child.slug}
              item={child}
              currentSlug={currentSlug}
              onPageSelect={onPageSelect}
              expanded={expanded}
              toggleExpand={toggleExpand}
              level={level + 1}
            />
          ))}
        </div>
      )}
    </div>
  )
}

export function WikiSidebar({ repoId, currentSlug, onPageSelect }: WikiSidebarProps) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set())

  const { data: navigation, isLoading, error } = useQuery({
    queryKey: ['wiki-navigation', repoId],
    queryFn: () => wikiApi.getNavigation(repoId),
  })

  const toggleExpand = (slug: string) => {
    const next = new Set(expanded)
    if (next.has(slug)) {
      next.delete(slug)
    } else {
      next.add(slug)
    }
    setExpanded(next)
  }

  if (isLoading) {
    return (
      <div className="bg-white rounded-lg border h-full">
        <div className="p-3 border-b font-medium text-sm flex items-center gap-2">
          <Book className="w-4 h-4" />
          Wiki
        </div>
        <div className="p-4 text-sm text-gray-500">Loading...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-white rounded-lg border h-full">
        <div className="p-3 border-b font-medium text-sm flex items-center gap-2">
          <Book className="w-4 h-4" />
          Wiki
        </div>
        <div className="p-4 text-sm text-red-500">Failed to load navigation</div>
      </div>
    )
  }

  const isEmpty = !navigation?.items || navigation.items.length === 0

  return (
    <div className="bg-white rounded-lg border h-full flex flex-col">
      <div className="p-3 border-b font-medium text-sm flex items-center gap-2">
        <Book className="w-4 h-4" />
        Wiki
      </div>
      <div className="p-2 flex-1 overflow-auto">
        {isEmpty ? (
          <div className="text-sm text-gray-500 text-center py-4">
            No pages yet. Generate wiki to get started.
          </div>
        ) : (
          navigation.items.map((item) => (
            <NavItem
              key={item.slug}
              item={item}
              currentSlug={currentSlug}
              onPageSelect={onPageSelect}
              expanded={expanded}
              toggleExpand={toggleExpand}
            />
          ))
        )}
      </div>
    </div>
  )
}
