import { RepositoryList } from '@/components/RepositoryList'
import { AddRepositoryForm } from '@/components/AddRepositoryForm'

export default function RepositoryListPage() {
  return (
    <>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-xl font-semibold">Repositories</h2>
        <AddRepositoryForm />
      </div>
      <RepositoryList />
    </>
  )
}
