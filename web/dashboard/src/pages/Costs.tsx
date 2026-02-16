import { useQuery } from '@tanstack/react-query';
import {
  BarChart, Bar, AreaChart, Area, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer
} from 'recharts';
import { fetchCostBreakdown } from '../api/dashboard';

const COLORS = ['#3b82f6', '#8b5cf6', '#22c55e', '#f59e0b', '#ef4444'];

export default function Costs() {
  const { data: costs, isLoading } = useQuery(['costBreakdown'], fetchCostBreakdown);

  const pipelineCosts = costs?.pipeline_costs ? Object.values(costs.pipeline_costs) as any[] : [];
  const trendData = (costs?.trend_data || []).map((t: any) => ({
    time: new Date(t.timestamp * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
    total: t.total_cost,
    cpu: t.cpu_cost,
    memory: t.memory_cost,
  }));

  const resourceData = [
    { name: 'CPU',     cost: costs?.cpu_cost ?? 0 },
    { name: 'Memory',  cost: costs?.memory_cost ?? 0 },
    { name: 'Storage', cost: costs?.storage_cost ?? 0 },
  ];

  const hasCostData = (costs?.total_cost ?? 0) > 0;

  return (
    <div className="p-8 space-y-6 max-w-[1400px]">
      <div>
        <h1 className="text-2xl font-bold text-white">Cost Analysis</h1>
        <p className="text-sm text-gray-400 mt-1">Track and optimize pipeline resource costs</p>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {[
          { label: 'Total Cost (24h)', value: `$${(costs?.total_cost ?? 0).toFixed(2)}`, icon: '💰', accent: 'text-purple-400' },
          { label: 'CPU Cost',         value: `$${(costs?.cpu_cost ?? 0).toFixed(2)}`,    icon: '🖥️', accent: 'text-blue-400' },
          { label: 'Memory Cost',      value: `$${(costs?.memory_cost ?? 0).toFixed(2)}`, icon: '🧠', accent: 'text-green-400' },
          { label: 'Storage Cost',     value: `$${(costs?.storage_cost ?? 0).toFixed(2)}`,icon: '💾', accent: 'text-yellow-400' },
        ].map((card, i) => (
          <div key={i} className="bg-gray-900 rounded-xl border border-gray-800 p-5">
            <div className="text-xs font-medium text-gray-400 uppercase tracking-wide">{card.label}</div>
            <div className={`text-2xl font-bold mt-1 ${card.accent}`}>{card.value}</div>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Cost Trend */}
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-5">
          <h3 className="text-sm font-semibold text-gray-300 mb-4">Cost Trend</h3>
          {trendData.length > 1 ? (
            <ResponsiveContainer width="100%" height={240}>
              <AreaChart data={trendData}>
                <defs>
                  <linearGradient id="costGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.3}/>
                    <stop offset="95%" stopColor="#8b5cf6" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#1f2937" />
                <XAxis dataKey="time" tick={{ fill: '#6b7280', fontSize: 11 }} />
                <YAxis tick={{ fill: '#6b7280', fontSize: 11 }} tickFormatter={(v) => `$${v.toFixed(2)}`} />
                <Tooltip contentStyle={{ background: '#111827', border: '1px solid #374151', borderRadius: 8, color: '#fff' }}
                  formatter={(v: any) => `$${Number(v).toFixed(4)}`} />
                <Area type="monotone" dataKey="total" stroke="#8b5cf6" fill="url(#costGrad)" name="Total Cost" />
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-60 flex items-center justify-center text-gray-600">
              <div className="text-center">
                <div className="text-4xl mb-2">📈</div>
                <div className="text-sm">Cost trend data will appear after multiple collection cycles</div>
              </div>
            </div>
          )}
        </div>

        {/* Resource Breakdown */}
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-5">
          <h3 className="text-sm font-semibold text-gray-300 mb-4">Resource Cost Breakdown</h3>
          {hasCostData ? (
            <ResponsiveContainer width="100%" height={240}>
              <BarChart data={resourceData} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke="#1f2937" />
                <XAxis type="number" tick={{ fill: '#6b7280', fontSize: 11 }} tickFormatter={(v) => `$${v.toFixed(2)}`} />
                <YAxis type="category" dataKey="name" tick={{ fill: '#9ca3af', fontSize: 12 }} width={60} />
                <Tooltip contentStyle={{ background: '#111827', border: '1px solid #374151', borderRadius: 8, color: '#fff' }}
                  formatter={(v: any) => `$${Number(v).toFixed(4)}`} />
                <Bar dataKey="cost" radius={[0, 6, 6, 0]}>
                  {resourceData.map((_, i) => <Cell key={i} fill={COLORS[i]} />)}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="h-60 flex items-center justify-center text-gray-600">
              <div className="text-center">
                <div className="text-4xl mb-2">💰</div>
                <div className="text-sm">No cost data available yet</div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Pipeline Cost Table */}
      <div className="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
        <div className="px-5 py-4 border-b border-gray-800">
          <h3 className="text-sm font-semibold text-gray-300">Cost by Pipeline</h3>
        </div>
        {isLoading ? (
          <div className="p-8 text-center text-gray-600 animate-pulse">Loading cost data...</div>
        ) : pipelineCosts.length === 0 ? (
          <div className="p-12 text-center">
            <div className="text-4xl mb-3">📊</div>
            <div className="text-gray-400 font-medium">No pipeline cost data yet</div>
            <div className="text-sm text-gray-600 mt-1">Cost data appears after pipeline runs complete</div>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-xs text-gray-500 uppercase tracking-wider border-b border-gray-800">
                  <th className="px-5 py-3 text-left">Pipeline</th>
                  <th className="px-5 py-3 text-left">Namespace</th>
                  <th className="px-5 py-3 text-right">Runs</th>
                  <th className="px-5 py-3 text-right">Total Cost</th>
                  <th className="px-5 py-3 text-right">Avg/Run</th>
                  <th className="px-5 py-3 text-right">CPU</th>
                  <th className="px-5 py-3 text-right">Memory</th>
                  <th className="px-5 py-3 text-right">Storage</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-800">
                {pipelineCosts.sort((a: any, b: any) => b.total_cost - a.total_cost).map((pc: any, i: number) => (
                  <tr key={i} className="hover:bg-gray-800/50">
                    <td className="px-5 py-3 font-medium text-white">{pc.pipeline_name || '—'}</td>
                    <td className="px-5 py-3 text-gray-400">{pc.namespace || '—'}</td>
                    <td className="px-5 py-3 text-right text-gray-300">{pc.run_count}</td>
                    <td className="px-5 py-3 text-right font-medium text-purple-400">${pc.total_cost?.toFixed(4)}</td>
                    <td className="px-5 py-3 text-right text-gray-300">${pc.average_cost_per_run?.toFixed(4)}</td>
                    <td className="px-5 py-3 text-right text-blue-400">${pc.cpu_cost?.toFixed(4)}</td>
                    <td className="px-5 py-3 text-right text-green-400">${pc.memory_cost?.toFixed(4)}</td>
                    <td className="px-5 py-3 text-right text-yellow-400">${pc.storage_cost?.toFixed(4)}</td>
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
