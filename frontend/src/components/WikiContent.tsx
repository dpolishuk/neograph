import { useQuery } from '@tanstack/react-query'
import { wikiApi, TOCItem } from '@/lib/api'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import rehypeHighlight from 'rehype-highlight'
import { useEffect, useRef } from 'react'
import mermaid from 'mermaid'
import { List, Clock } from 'lucide-react'

interface WikiContentProps {
  repoId: string
  slug: string
}

function TableOfContents({ items }: { items: TOCItem[] }) {
  if (!items || items.length === 0) return null

  return (
    <div className="bg-gray-50 rounded-lg p-4 mb-6">
      <h3 className="text-sm font-medium text-gray-700 mb-2 flex items-center gap-2">
        <List className="w-4 h-4" />
        Table of Contents
      </h3>
      <nav className="text-sm">
        {items.map((item) => (
          <a
            key={item.id}
            href={`#${item.id}`}
            className="block text-blue-600 hover:underline py-0.5"
            style={{ paddingLeft: `${(item.level - 1) * 12}px` }}
          >
            {item.title}
          </a>
        ))}
      </nav>
    </div>
  )
}

function MermaidDiagram({ code }: { code: string }) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (ref.current) {
      mermaid.initialize({
        startOnLoad: false,
        theme: 'default',
        securityLevel: 'loose',
      })

      const id = `mermaid-${Math.random().toString(36).substr(2, 9)}`

      mermaid.render(id, code).then(({ svg }) => {
        if (ref.current) {
          ref.current.innerHTML = svg
        }
      }).catch((error) => {
        if (ref.current) {
          ref.current.innerHTML = `<pre class="text-red-500 text-sm">Mermaid error: ${error.message}</pre>`
        }
      })
    }
  }, [code])

  return <div ref={ref} className="my-4 overflow-auto" />
}

function MarkdownContent({ content }: { content: string }) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      rehypePlugins={[rehypeHighlight]}
      components={{
        h1: ({ children, ...props }) => (
          <h1 id={createId(children)} className="text-3xl font-bold mt-8 mb-4 pb-2 border-b" {...props}>
            {children}
          </h1>
        ),
        h2: ({ children, ...props }) => (
          <h2 id={createId(children)} className="text-2xl font-semibold mt-6 mb-3" {...props}>
            {children}
          </h2>
        ),
        h3: ({ children, ...props }) => (
          <h3 id={createId(children)} className="text-xl font-semibold mt-4 mb-2" {...props}>
            {children}
          </h3>
        ),
        h4: ({ children, ...props }) => (
          <h4 id={createId(children)} className="text-lg font-medium mt-4 mb-2" {...props}>
            {children}
          </h4>
        ),
        p: ({ children }) => <p className="my-3 leading-relaxed">{children}</p>,
        ul: ({ children }) => <ul className="list-disc list-inside my-3 space-y-1">{children}</ul>,
        ol: ({ children }) => <ol className="list-decimal list-inside my-3 space-y-1">{children}</ol>,
        li: ({ children }) => <li className="ml-4">{children}</li>,
        a: ({ href, children }) => (
          <a href={href} className="text-blue-600 hover:underline" target="_blank" rel="noopener noreferrer">
            {children}
          </a>
        ),
        code: ({ className, children, ...props }) => {
          const match = /language-(\w+)/.exec(className || '')
          const isBlock = match || (typeof children === 'string' && children.includes('\n'))

          // Handle mermaid code blocks
          if (match && match[1] === 'mermaid') {
            return <MermaidDiagram code={String(children).trim()} />
          }

          if (isBlock) {
            return (
              <pre className="bg-gray-900 text-gray-100 rounded-lg p-4 my-4 overflow-auto">
                <code className={className} {...props}>
                  {children}
                </code>
              </pre>
            )
          }
          return (
            <code className="bg-gray-100 text-gray-800 px-1.5 py-0.5 rounded text-sm" {...props}>
              {children}
            </code>
          )
        },
        pre: ({ children }) => <>{children}</>,
        blockquote: ({ children }) => (
          <blockquote className="border-l-4 border-blue-500 pl-4 my-4 italic text-gray-700">
            {children}
          </blockquote>
        ),
        table: ({ children }) => (
          <div className="my-4 overflow-auto">
            <table className="min-w-full border-collapse border border-gray-300">{children}</table>
          </div>
        ),
        th: ({ children }) => (
          <th className="border border-gray-300 px-4 py-2 bg-gray-100 font-semibold text-left">
            {children}
          </th>
        ),
        td: ({ children }) => (
          <td className="border border-gray-300 px-4 py-2">{children}</td>
        ),
        hr: () => <hr className="my-8 border-gray-300" />,
      }}
    >
      {content}
    </ReactMarkdown>
  )
}

function createId(children: React.ReactNode): string {
  const text = String(children)
  return text
    .toLowerCase()
    .replace(/\s+/g, '-')
    .replace(/[^a-z0-9-]/g, '')
}

export function WikiContent({ repoId, slug }: WikiContentProps) {
  const { data: page, isLoading, error } = useQuery({
    queryKey: ['wiki-page', repoId, slug],
    queryFn: () => wikiApi.getPage(repoId, slug),
    enabled: !!slug,
  })

  if (!slug) {
    return (
      <div className="bg-white rounded-lg border h-full flex items-center justify-center">
        <div className="text-center text-gray-500">
          <p>Select a page from the sidebar to view content</p>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="bg-white rounded-lg border h-full p-8">
        <div className="animate-pulse space-y-4">
          <div className="h-8 bg-gray-200 rounded w-3/4"></div>
          <div className="h-4 bg-gray-200 rounded w-1/2"></div>
          <div className="h-4 bg-gray-200 rounded w-full"></div>
          <div className="h-4 bg-gray-200 rounded w-full"></div>
          <div className="h-4 bg-gray-200 rounded w-3/4"></div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-white rounded-lg border h-full p-8">
        <div className="text-red-500">Failed to load page content</div>
      </div>
    )
  }

  if (!page) {
    return (
      <div className="bg-white rounded-lg border h-full p-8">
        <div className="text-gray-500">Page not found</div>
      </div>
    )
  }

  return (
    <div className="bg-white rounded-lg border h-full overflow-auto">
      <div className="p-8 max-w-4xl mx-auto">
        {page.generatedAt && (
          <div className="flex items-center gap-1 text-xs text-gray-500 mb-4">
            <Clock className="w-3 h-3" />
            Generated: {new Date(page.generatedAt).toLocaleDateString()}
          </div>
        )}

        {page.tableOfContents && page.tableOfContents.length > 0 && (
          <TableOfContents items={page.tableOfContents} />
        )}

        <article className="prose prose-slate max-w-none">
          <MarkdownContent content={page.content} />
        </article>

        {page.diagrams && page.diagrams.length > 0 && (
          <div className="mt-8 border-t pt-8">
            <h2 className="text-xl font-semibold mb-4">Diagrams</h2>
            {page.diagrams.map((diagram) => (
              <div key={diagram.id} className="mb-6">
                <h3 className="text-lg font-medium mb-2">{diagram.title}</h3>
                <MermaidDiagram code={diagram.code} />
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
