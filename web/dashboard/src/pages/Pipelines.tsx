import { useQuery } from '@tanstack/react-query';
import { fetchPipelineMetrics, fetchTaskMetrics } from '../api/dashboard';

function StatusBadge({ rate }: { rate: number }) {
  const color = rate >= 90 ? 'bg-green-900/50 text-green-400 border-green-800'
    : rate >= 70 ? 'bg-yellow-900/50 text-yellow-400 border-yellow-800'
    : 'bg-red-900/50 text-red-400 border-red-800';
  return (
    <span className={`px-2 py-0.5 rounded-full text-xs font-medium border ${color}`}>
      {rate.toFixed(1)}%
    </span>
  );
}

export default function Pipelines() {
  const { data: pipelines, isLoading: loadingP } = useQuery(['pipelines'], fetchPipelineMetrics);
  const { data: tasks, isLoading: loadingT } = useQuery(['tasks'], fetchTaskMetrics);

  const pipelineList = pipelines ? Object.values(pipelines) as any[] : [];
  const taskList = tasks ? Object.values(tasks) as any[] : [];

  return (
    <div className="p-8 space-y-6 max-w-[1400px]">
      <div>
        <h1 className="text-2xl font-bold text-white">Pipelines</h1>
        <p className="text-sm text-gray-400 mt-1">Monitor pipeline and task execution metrics</p>
      </div>

      {/* Pipeline Table */}
      <div className="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
        <div className="px-5 py-4 border-b border-gray-800 flex items-center justify-between">
          <h3 className="text-sm font-semibold text-gray-300">Pipeline Metrics</h3>
          <span className="text-xs text-gray-500">{pipelineList.length} pipelines</span>
        </div>
        {loadingP ? (
          <div className="p-8 text-center text-gray-600 animate-pulse">Loading pipeline data...</div>
        ) : pipelineList.length === 0 ? (
          <div className="p-12 text-center">
            <div className="text-4xl mb-3">🔄</div>
            <div className="text-gray-400 font-medium">No pipeline data yet</div>
            <div className="text-sm text-gray-600 mt-1">Run some pipelines and metrics will appear here</div>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-xs text-gray-500 uppercase tracking-wider border-b border-gray-800">
                  <th className="px-5 py-3 text-left">Pipeline</th>
                  <th className="px-5 py-3 text-left">Namespace</th>
                  <th className="px-5 py-3 text-right">Total Runs</th>
                  <th className="px-5 py-3 text-right">Succeeded</th>
                  <th className="px-5 py-3 text-right">Failed</th>
                  <th className="px-5 py-3 text-right">Success Rate</th>
                  <th className="px-5 py-3 text-right">Avg Duration</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-800">
                {pipelineList.map((p: any, i: number) => (
                  <tr key={i} className="hover:bg-gray-800/50">
                    <td className="px-5 py-3 font-medium text-white">{p.name || '—'}</td>
                    <td className="px-5 py-3 text-gray-400">{p.namespace || '—'}</td>
                    <td className="px-5 py-3 text-right text-gray-300">{p.total_runs ?? 0}</td>
                    <td className="px-5 py-3 text-right text-green-400">{p.successful_runs ?? 0}</td>
                    <td className="px-5 py-3 text-right text-red-400">{p.failed_runs ?? 0}</td>
                    <td className="px-5 py-3 text-right"><StatusBadge rate={p.success_rate ?? 0} /></td>
                    <td className="px-5 py-3 text-right text-gray-300">
                      {p.average_duration ? `${p.average_duration.toFixed(1)}s` : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Task Table */}
      <div className="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
        <div className="px-5 py-4 border-b border-gray-800 flex items-center justify-between">
          <h3 className="text-sm font-semibold text-gray-300">Task Metrics</h3>
          <span className="text-xs text-gray-500">{taskList.length} tasks</span>
        </div>
        {loadingT ? (
          <div className="p-8 text-center text-gray-600 animate-pulse">Loading task data...</div>
        ) : taskList.length === 0 ? (
          <div className="p-12 text-center">
            <div className="text-4xl mb-3">📋</div>
            <div className="text-gray-400 font-medium">No task data yet</div>
            <div className="text-sm text-gray-600 mt-1">Task metrics will appear after pipeline runs</div>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-xs text-gray-500 uppercase tracking-wider border-b border-gray-800">
                  <th className="px-5 py-3 text-left">Task</th>
                  <th className="px-5 py-3 text-left">Namespace</th>
                  <th className="px-5 py-3 text-right">Total Runs</th>
                  <th className="px-5 py-3 text-right">Succeeded</th>
                  <th className="px-5 py-3 text-right">Failed</th>
                  <th className="px-5 py-3 text-right">Success Rate</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-800">
                {taskList.map((t: any, i: number) => (
                  <tr key={i} className="hover:bg-gray-800/50">
                    <td className="px-5 py-3 font-medium text-white">{t.name || '—'}</td>
                    <td className="px-5 py-3 text-gray-400">{t.namespace || '—'}</td>
                    <td className="px-5 py-3 text-right text-gray-300">{t.total_runs ?? 0}</td>
                    <td className="px-5 py-3 text-right text-green-400">{t.successful_runs ?? 0}</td>
                    <td className="px-5 py-3 text-right text-red-400">{t.failed_runs ?? 0}</td>
                    <td className="px-5 py-3 text-right"><StatusBadge rate={t.success_rate ?? 0} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
