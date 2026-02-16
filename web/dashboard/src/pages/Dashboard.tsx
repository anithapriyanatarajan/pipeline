import { useQuery } from '@tanstack/react-query';
import {
  BarChart, Bar, PieChart, Pie, Cell, AreaChart, Area,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer
} from 'recharts';
import { fetchOverviewMetrics, fetchMetricsHistory, fetchCostBreakdown, fetchInsights } from '../api/dashboard';

/* ─── Stat Card ─── */
function StatCard({ label, value, sub, icon, accent }: {
  label: string; value: string | number; sub?: string; icon: string; accent: string;
}) {
  return (
    <div className="bg-gray-900 rounded-xl border border-gray-800 p-5 flex items-start gap-4">
      <div className={`h-11 w-11 rounded-lg flex items-center justify-center text-xl ${accent}`}>{icon}</div>
      <div>
        <div className="text-xs font-medium text-gray-400 uppercase tracking-wide">{label}</div>
        <div className="text-2xl font-bold text-white mt-0.5">{value}</div>
        {sub && <div className="text-xs text-gray-500 mt-1">{sub}</div>}
      </div>
    </div>
  );
}

/* ─── Loading Skeleton ─── */
function Skeleton() {
  return (
    <div className="animate-pulse space-y-6 p-8">
      <div className="h-8 bg-gray-800 rounded w-64"></div>
      <div className="grid grid-cols-4 gap-4">
        {[1,2,3,4].map(i => <div key={i} className="h-24 bg-gray-800 rounded-xl"></div>)}
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="h-64 bg-gray-800 rounded-xl"></div>
        <div className="h-64 bg-gray-800 rounded-xl"></div>
      </div>
    </div>
  );
}

const CHART_COLORS = ['#3b82f6', '#22c55e', '#f59e0b', '#ef4444', '#8b5cf6'];

export default function Dashboard() {
  const { data: overview, isLoading: loadingOverview } = useQuery(['overview'], fetchOverviewMetrics);
  const { data: history, isLoading: loadingHistory } = useQuery(['history'], () => fetchMetricsHistory('1h'));
  const { data: costs } = useQuery(['costs'], fetchCostBreakdown);
  const { data: insights } = useQuery(['insights'], fetchInsights);

  if (loadingOverview && loadingHistory) return <Skeleton />;

  const successRate = overview?.success_rate ?? 0;
  const totalRuns = (overview?.successful_pipelines ?? 0) + (overview?.failed_pipelines ?? 0);

  // Build pie data only if we have runs
  const pieData = [
    { name: 'Succeeded', value: overview?.successful_pipelines ?? 0 },
    { name: 'Failed',    value: overview?.failed_pipelines ?? 0 },
    { name: 'Running',   value: overview?.running_pipelines ?? 0 },
  ].filter(d => d.value > 0);

  // Cost bar data
  const costData = [
    { name: 'CPU',     cost: costs?.cpu_cost ?? 0 },
    { name: 'Memory',  cost: costs?.memory_cost ?? 0 },
    { name: 'Storage', cost: costs?.storage_cost ?? 0 },
  ];

  // History for area chart
  const historyData = (history || []).map((h: any) => ({
    time: new Date(h.timestamp * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    running: h.running_pipelines ?? 0,
    total: h.total_pipelines ?? 0,
  }));

  return (
    <div className="p-8 space-y-6 max-w-[1400px]">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-white">Dashboard</h1>
        <p className="text-sm text-gray-400 mt-1">Real-time observability for Tekton Pipelines</p>
      </div>

      {/* Stat Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard icon="🔄" label="Running Pipelines" value={overview?.running_pipelines ?? 0}
          sub={`${totalRuns} total runs`} accent="bg-blue-600/20 text-blue-400" />
        <StatCard icon="✅" label="Success Rate" value={`${successRate.toFixed(1)}%`}
          sub={`${overview?.successful_pipelines ?? 0} succeeded`}
          accent={successRate >= 80 ? 'bg-green-600/20 text-green-400' : 'bg-yellow-600/20 text-yellow-400'} />
        <StatCard icon="💰" label="Cost (24h)" value={`$${(costs?.total_cost ?? 0).toFixed(2)}`}
          sub={`${Object.keys(costs?.pipeline_costs ?? {}).length} pipelines tracked`} accent="bg-purple-600/20 text-purple-400" />
        <StatCard icon="⚠️" label="Anomalies" value={overview?.active_anomalies ?? 0}
          sub={`${overview?.open_recommendations ?? 0} recommendations`} accent="bg-red-600/20 text-red-400" />
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Pipeline Activity */}
        <div className="lg:col-span-2 bg-gray-900 rounded-xl border border-gray-800 p-5">
          <h3 className="text-sm font-semibold text-gray-300 mb-4">Pipeline Activity</h3>
          {historyData.length > 0 ? (
            <ResponsiveContainer width="100%" height={240}>
              <AreaChart data={historyData}>
                <defs>
                  <linearGradient id="colorRunning" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/>
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#1f2937" />
                <XAxis dataKey="time" tick={{ fill: '#6b7280', fontSize: 11 }} />
                <YAxis tick={{ fill: '#6b7280', fontSize: 11 }} />
                <Tooltip contentStyle={{ background: '#111827', border: '1px solid #374151', borderRadius: 8, color: '#fff' }} />
                <Area type="monotone" dataKey="running" stroke="#3b82f6" fill="url(#colorRunning)" name="Running" />
                <Area type="monotone" dataKey="total" stroke="#6b7280" fill="none" strokeDasharray="4 4" name="Total" />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-60 flex items-center justify-center text-gray-600">
              <div className="text-center">
                <div className="text-4xl mb-2">📈</div>
                <div className="text-sm">Collecting metrics... data will appear shortly</div>
              </div>
            </div>
          )}
        </div>

        {/* Status Distribution */}
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-5">
          <h3 className="text-sm font-semibold text-gray-300 mb-4">Status Distribution</h3>
          {pieData.length > 0 ? (
            <ResponsiveContainer width="100%" height={240}>
              <PieChart>
                <Pie data={pieData} cx="50%" cy="50%" innerRadius={50} outerRadius={80}
                  dataKey="value" paddingAngle={3} label={({ name, value }) => `${name}: ${value}`}
                >
                  {pieData.map((_, i) => (
                    <Cell key={i} fill={CHART_COLORS[i % CHART_COLORS.length]} />
                  ))}
                </Pie>
                <Tooltip contentStyle={{ background: '#111827', border: '1px solid #374151', borderRadius: 8, color: '#fff' }} />
              </PieChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-60 flex items-center justify-center text-gray-600">
              <div className="text-center">
                <div className="text-4xl mb-2">🥧</div>
                <div className="text-sm">No pipeline runs yet</div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Cost + Insights Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Cost Breakdown */}
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-5">
          <h3 className="text-sm font-semibold text-gray-300 mb-4">Cost Breakdown</h3>
          {(costs?.total_cost ?? 0) > 0 ? (
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={costData} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke="#1f2937" />
                <XAxis type="number" tick={{ fill: '#6b7280', fontSize: 11 }} tickFormatter={(v) => `$${v.toFixed(2)}`} />
                <YAxis type="category" dataKey="name" tick={{ fill: '#9ca3af', fontSize: 12 }} width={60} />
                <Tooltip contentStyle={{ background: '#111827', border: '1px solid #374151', borderRadius: 8, color: '#fff' }}
                  formatter={(v: any) => `$${Number(v).toFixed(4)}`} />
                <Bar dataKey="cost" radius={[0, 6, 6, 0]}>
                  {costData.map((_, i) => <Cell key={i} fill={CHART_COLORS[i]} />)}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-48 flex items-center justify-center text-gray-600">
              <div className="text-center">
                <div className="text-4xl mb-2">💰</div>
                <div className="text-sm">Cost data will appear after pipeline runs complete</div>
              </div>
            </div>
          )}
        </div>

        {/* Recent Insights */}
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-5">
          <h3 className="text-sm font-semibold text-gray-300 mb-4">Recent Insights</h3>
          <div className="space-y-3 max-h-48 overflow-y-auto">
            {insights?.anomalies?.slice(0, 3).map((a: any, i: number) => (
              <div key={i} className="flex items-start gap-3 p-3 rounded-lg bg-red-950/40 border border-red-900/50">
                <span className="text-lg">⚠️</span>
                <div className="min-w-0">
                  <div className="text-xs font-semibold text-red-400">{a.type?.toUpperCase()} — {a.severity}</div>
                  <div className="text-sm text-gray-300 mt-0.5 truncate">{a.description}</div>
                </div>
              </div>
            ))}
            {insights?.recommendations?.slice(0, 2).map((r: any, i: number) => (
              <div key={i} className="flex items-start gap-3 p-3 rounded-lg bg-blue-950/40 border border-blue-900/50">
                <span className="text-lg">💡</span>
                <div className="min-w-0">
                  <div className="text-xs font-semibold text-blue-400">{r.title}</div>
                  <div className="text-sm text-gray-300 mt-0.5 truncate">{r.description}</div>
                </div>
              </div>
            ))}
            {(!insights?.anomalies?.length && !insights?.recommendations?.length) && (
              <div className="text-center py-6 text-gray-600">
                <div className="text-3xl mb-2">✨</div>
                <div className="text-sm">No anomalies detected — everything looks healthy!</div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
