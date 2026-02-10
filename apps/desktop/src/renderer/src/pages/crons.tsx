import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@multica/ui/components/ui/card'
import { useCronJobs } from '../hooks/use-cron-jobs'
import { CronJobList } from '../components/cron-job-list'

export default function CronsPage() {
  const {
    jobs,
    loading,
    error,
    toggleJob,
    removeJob,
    refresh,
  } = useCronJobs()

  return (
    <div className="max-w-4xl mx-auto">
      <Card>
        <CardHeader>
          <CardTitle>Cron Jobs</CardTitle>
          <CardDescription>
            View and manage scheduled tasks. Create new jobs by asking the Agent in Chat.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <CronJobList
            jobs={jobs}
            loading={loading}
            error={error}
            onToggleJob={toggleJob}
            onRemoveJob={removeJob}
            onRefresh={refresh}
          />
        </CardContent>
      </Card>
    </div>
  )
}
