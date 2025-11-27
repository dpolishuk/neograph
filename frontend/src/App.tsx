import { Routes, Route } from 'react-router-dom'
import RepositoryListPage from './pages/RepositoryListPage'
import RepositoryDetailPage from './pages/RepositoryDetailPage'

function App() {
  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b">
        <div className="max-w-7xl mx-auto px-4 py-6">
          <h1 className="text-2xl font-bold text-gray-900">NeoGraph</h1>
          <p className="text-gray-500">Code Intelligence with Neo4j</p>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 py-8">
        <Routes>
          <Route path="/" element={<RepositoryListPage />} />
          <Route path="/repository/:id" element={<RepositoryDetailPage />} />
        </Routes>
      </main>
    </div>
  )
}

export default App
