import { useQuery } from '@tanstack/react-query';
import { fetchTraces } from '../api/dashboard';

const STATUS_COLORS: Record<string, string> = {
  Succeeded: 'bg-green-500',
  Failed: 'bg-red-500',
  Running: 'bg-blue-500',
  Unknown: 'bg-gray-500',
};
const STATUS_BADGE: Record<string, string> = {
  Succeeded: 'bg-green-900/50 text-green-400 border-green-800',
  Failed: 'bg-red-900/50 text-red-400 border-red-800',
  Running: 'bg-blue-900/50 text-blue-400 border-blue-800',
  Unknown: 'bg-gray-900/50 text-gray-400 border-gray-800',
};

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const m = Math.floor(seconds / 60);
  const s = Math.round(seconds % 60);
  return `${m}m ${s}s`;
}

function formatTime(unix: number): string {
  if (!unix) return '—';
  return new Date(unix * 1000).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

/* Waterfall bar for a single span */
function SpanBar({ span, traceStart, traceDuration }: { span: any; traceStart: number; traceDuration: number }) {
  const left = traceDuration > 0 ? ((span.start_time - traceStart) / traceDuration) * 100 : 0;
  const width = traceDuration > 0 ? (span.duration / traceDuration) * 100 : 100;
  const barColor = STATUS_COLORS[span.status] || STATUS_COLORS.Unknown;

  return (
    <div className="flex items-center gap-3 py-1.5 group hover:bg-gray-800/30 rounded px-2">
      <div className="w-40 shrink-0 text-xs text-gray-400 truncate" title={span.name}>{span.name}</div>
      <div className="flex-1 h-6 bg-gray-800 rounded relative overflow-hidden">
        <div className={`absolute h-full rounded ${barColor} opacity-80 group-hover:opacity-100`}
          style={{ left: `${Math.max(0, left)}%`, width: `${Math.max(1, width)}%` }}
        />
        <div className="absolute inset-0 flex items-center text-[10px] text-gray-400 px-1 justify-end">
          {formatDuration(span.duration || 0)}
        </div>
      </div>
      <span className={`px-2 py-0.5 rounded-full text-[10px] font-medium border shrink-0 ${STATUS_BADGE[span.status] || STATUS_BADGE.Unknown}`}>
        {span.status}
      </span>
    </div>
  );
}

export default function Traces() {
  const { data: traceData, isLoading } = useQuery(['traces'], fetchTraces);

  const traces = traceData?.traces || [];

  return (
    <div className="p-8 space-y-6 max-w-[1400px]">
      <div>
        <h1 className="text-2xl font-bold text-white">Traces</h1>
        <p className="text-sm text-gray-400 mt-1">Visualize pipeline execution flow and task dependencies</p>
      </div>

      {isLoading ? (
        <div className="animate-pulse space-y-4">
          {[1,2,3].map(i => <div key={i} className="h-48 bg-gray-800 rounded-xl"></div>)}
        </div>
      ) : traces.length === 0 ? (
        <div className="bg-gray-900 rounded-xl border border-gray-800 p-16 text-center">
          <div className="text-5xl mb-4">🔍</div>
          <div className="text-lg text-gray-400 font-medium">No traces available yet</div>
          <div className="text-sm text-gray-600 mt-2 max-w-md mx-auto">
            Traces are built from PipelineRun and TaskRun data. Run some pipelines and traces will appear here with a waterfall view of task execution.
          </div>
        </div>
      ) : (
        <div className="space-y-4">
          {traces.map((trace: any) => (
            <div key={trace.trace_id} className="bg-gray-900 rounded-xl border border-gray-800 overflow-hidden">
              {/* Trace Header */}
              <div className="px-5 py-4 border-b border-gray-800 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <span className={`px-2 py-0.5 rounded-full text-xs font-medium border ${STATUS_BADGE[trace.status] || STATUS_BADGE.Unknown}`}>
                    {trace.status}
                  </span>
                  <div>
                    <span className="text-sm font-semibold text-white">{trace.pipeline_run}</span>
                    <span className="text-xs text-gray-500 ml-2">({trace.pipeline})</span>
                  </div>
                </div>
                <div className="flex items-center gap-4 text-xs text-gray-500">
                  <span>Namespace: <span className="text-gray-300">{trace.namespace}</span></span>
                  <span>Duration: <span className="text-gray-300 font-medium">{formatDuration(trace.duration || 0)}</span></span>
                  <span>Started: <span className="text-gray-300">{formatTime(trace.start_time)}</span></span>
                  <span>{trace.spans?.length || 0} tasks</span>
                </div>
              </div>

              {/* Waterfall */}
              <div className="px-5 py-3">
                {trace.spans && trace.spans.length > 0 ? (
                  <div>
                    <div className="flex items-center gap-3 pb-2 mb-2 border-b border-gray-800">
                      <div className="w-40 shrink-0 text-[10px] text-gray-600 uppercase tracking-wider">Task</div>
                      <div className="flex-1 flex justify-between text-[10px] text-gray-600">
                        <span>{formatTime(trace.start_time)}</span>
                        <span>{formatTime(trace.end_time)}</span>
                      </div>
                      <div className="w-16 shrink-0"></div>
                    </div>
                    {trace.spans.map((span: any) => (
                      <SpanBar key={span.span_id} span={span} traceStart={trace.start_time} traceDuration={trace.duration} />
                    ))}
                  </div>
                ) : (
                  <div className="text-center py-4 text-gray-600 text-sm">No task spans recorded</div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
