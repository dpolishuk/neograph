import { Routes, Route, Link } from 'react-router-dom'
import RepositoryListPage from './pages/RepositoryListPage'
import RepositoryDetailPage from './pages/RepositoryDetailPage'
import SearchPage from './pages/SearchPage'
import { CommandBar } from './components/CommandBar'
import { Search } from 'lucide-react'

function App() {
  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b">
        <div className="max-w-7xl mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div>
              <Link to="/">
                <h1 className="text-2xl font-bold text-gray-900 hover:text-blue-600 transition-colors">
                  NeoGraph
                </h1>
              </Link>
              <p className="text-gray-500">Code Intelligence with Neo4j</p>
            </div>
            <nav className="flex items-center gap-4">
              <Link
                to="/search"
                className="flex items-center gap-2 text-gray-700 hover:text-blue-600 transition-colors"
              >
                <Search className="w-4 h-4" />
                <span>Search</span>
              </Link>
            </nav>
          </div>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 py-8">
        <Routes>
          <Route path="/" element={<RepositoryListPage />} />
          <Route path="/repository/:id" element={<RepositoryDetailPage />} />
          <Route path="/search" element={<SearchPage />} />
        </Routes>
      </main>

      <CommandBar />
    </div>
  )
}

export default App
