import { useQuery } from '@tanstack/react-query';
import { fetchInsights } from '../api/dashboard';

const SEVERITY_COLORS: Record<string, string> = {
  critical: 'border-red-500 bg-red-950/60',
  high:     'border-red-700 bg-red-950/40',
  medium:   'border-yellow-700 bg-yellow-950/30',
  low:      'border-blue-700 bg-blue-950/30',
};
const SEVERITY_TEXT: Record<string, string> = {
  critical: 'text-red-400',
  high:     'text-red-400',
  medium:   'text-yellow-400',
  low:      'text-blue-400',
};
const PRIORITY_BADGE: Record<string, string> = {
  high:   'bg-red-900/50 text-red-400 border-red-800',
  medium: 'bg-yellow-900/50 text-yellow-400 border-yellow-800',
  low:    'bg-blue-900/50 text-blue-400 border-blue-800',
};

export default function Insights() {
  const { data: insights, isLoading } = useQuery(['insights'], fetchInsights);

  const anomalies = insights?.anomalies || [];
  const recommendations = insights?.recommendations || [];
  const predictions = insights?.predictions || [];

  if (isLoading) {
    return (
      <div className="p-8 space-y-6">
        <div className="h-8 bg-gray-800 rounded w-48 animate-pulse"></div>
        {[1,2,3].map(i => <div key={i} className="h-32 bg-gray-800 rounded-xl animate-pulse"></div>)}
      </div>
    );
  }

  const hasData = anomalies.length > 0 || recommendations.length > 0 || predictions.length > 0;

  return (
    <div className="p-8 space-y-6 max-w-[1400px]">
      <div>
        <h1 className="text-2xl font-bold text-white">AI Insights</h1>
        <p className="text-sm text-gray-400 mt-1">Anomaly detection, recommendations, and predictive analytics</p>
      </div>

      {/* Summary Row */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-5 flex items-center gap-4">
          <div className="h-11 w-11 rounded-lg bg-red-600/20 flex items-center justify-center text-xl">⚠️</div>
          <div>
            <div className="text-xs font-medium text-gray-400 uppercase">Anomalies</div>
            <div className="text-2xl font-bold text-red-400">{anomalies.length}</div>
          </div>
        </div>
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-5 flex items-center gap-4">
          <div className="h-11 w-11 rounded-lg bg-blue-600/20 flex items-center justify-center text-xl">💡</div>
          <div>
            <div className="text-xs font-medium text-gray-400 uppercase">Recommendations</div>
            <div className="text-2xl font-bold text-blue-400">{recommendations.length}</div>
          </div>
        </div>
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-5 flex items-center gap-4">
          <div className="h-11 w-11 rounded-lg bg-purple-600/20 flex items-center justify-center text-xl">🔮</div>
          <div>
            <div className="text-xs font-medium text-gray-400 uppercase">Predictions</div>
            <div className="text-2xl font-bold text-purple-400">{predictions.length}</div>
          </div>
        </div>
      </div>

      {!hasData ? (
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-16 text-center">
          <div className="text-5xl mb-4">✨</div>
          <div className="text-lg text-gray-400 font-medium">All systems healthy</div>
          <div className="text-sm text-gray-600 mt-2 max-w-md mx-auto">
            The AI engine analyzes pipeline metrics to detect anomalies and generate optimization recommendations.
            Insights will appear as more data is collected.
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Anomalies */}
          <div className="space-y-4">
            <h2 className="text-sm font-semibold text-gray-300 flex items-center gap-2">
              <span className="text-lg">⚠️</span> Anomalies ({anomalies.length})
            </h2>
            {anomalies.length === 0 ? (
              <div className="bg-gray-900 rounded-xl border border-gray-800 p-8 text-center text-gray-600 text-sm">
                No anomalies detected
              </div>
            ) : (
              anomalies.map((a: any, i: number) => (
                <div key={i}
                  className={`rounded-xl border-l-4 p-4 ${SEVERITY_COLORS[a.severity] || SEVERITY_COLORS.low}`}
                >
                  <div className="flex items-center justify-between mb-2">
                    <span className={`text-xs font-bold uppercase tracking-wide ${SEVERITY_TEXT[a.severity] || 'text-gray-400'}`}>
                      {a.severity} — {a.type}
                    </span>
                    <span className="text-[10px] text-gray-500">
                      Score: {a.score?.toFixed(1)}
                    </span>
                  </div>
                  <p className="text-sm text-gray-300">{a.description}</p>
                  <div className="flex items-center gap-3 mt-2 text-xs text-gray-500">
                    <span>Pipeline: <span className="text-gray-400">{a.pipeline}</span></span>
                    <span>Namespace: <span className="text-gray-400">{a.namespace}</span></span>
                  </div>
                </div>
              ))
            )}
          </div>

          {/* Recommendations + Predictions */}
          <div className="space-y-6">
            {/* Recommendations */}
            <div className="space-y-4">
              <h2 className="text-sm font-semibold text-gray-300 flex items-center gap-2">
                <span className="text-lg">💡</span> Recommendations ({recommendations.length})
              </h2>
              {recommendations.length === 0 ? (
                <div className="bg-gray-900 rounded-xl border border-gray-800 p-8 text-center text-gray-600 text-sm">
                  No recommendations at this time
                </div>
              ) : (
                recommendations.map((r: any, i: number) => (
                  <div key={i} className="bg-gray-900 rounded-xl border border-gray-800 p-4">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-sm font-semibold text-white">{r.title}</span>
                      <span className={`px-2 py-0.5 rounded-full text-[10px] font-medium border ${PRIORITY_BADGE[r.priority] || PRIORITY_BADGE.low}`}>
                        {r.priority}
                      </span>
                    </div>
                    <p className="text-sm text-gray-400">{r.description}</p>
                    <div className="flex items-center gap-4 mt-3 text-xs">
                      {r.impact && <span className="text-green-400">Impact: {r.impact}</span>}
                      {r.savings > 0 && <span className="text-purple-400">Savings: ${r.savings.toFixed(2)}</span>}
                      {r.effort && <span className="text-gray-500">Effort: {r.effort}</span>}
                    </div>
                  </div>
                ))
              )}
            </div>

            {/* Predictions */}
            <div className="space-y-4">
              <h2 className="text-sm font-semibold text-gray-300 flex items-center gap-2">
                <span className="text-lg">🔮</span> Predictions ({predictions.length})
              </h2>
              {predictions.length === 0 ? (
                <div className="bg-gray-900 rounded-xl border border-gray-800 p-8 text-center text-gray-600 text-sm">
                  No predictions available — more data needed
                </div>
              ) : (
                predictions.map((p: any, i: number) => (
                  <div key={i} className="bg-gray-900 rounded-xl border border-gray-800 p-4">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-sm font-semibold text-white">{p.pipeline}</span>
                      <div className="flex items-center gap-2">
                        <div className="w-16 h-1.5 bg-gray-700 rounded-full overflow-hidden">
                          <div className="h-full bg-purple-500 rounded-full" style={{ width: `${(p.confidence || 0) * 100}%` }} />
                        </div>
                        <span className="text-[10px] text-gray-400">{((p.confidence || 0) * 100).toFixed(0)}%</span>
                      </div>
                    </div>
                    <p className="text-sm text-gray-400">{p.description}</p>
                    <div className="text-xs text-gray-500 mt-2">
                      Type: {p.type} • Namespace: {p.namespace}
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
