import { AddRepositoryForm } from '@/components/AddRepositoryForm'
import { RepositoryList } from '@/components/RepositoryList'

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
        <div className="flex justify-between items-center mb-6">
          <h2 className="text-xl font-semibold">Repositories</h2>
          <AddRepositoryForm />
        </div>

        <RepositoryList />
      </main>
    </div>
  )
}

export default App
