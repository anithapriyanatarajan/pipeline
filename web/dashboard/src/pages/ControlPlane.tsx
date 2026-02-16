import { useQuery } from '@tanstack/react-query';
import { fetchControlPlaneStatus } from '../api/dashboard';

/* ─── tiny helpers ─── */

const healthColor: Record<string, string> = {
  Healthy:      'bg-green-500',
  Degraded:     'bg-yellow-500',
  Unhealthy:    'bg-red-500',
  'Scaled Down':'bg-gray-500',
  Unknown:      'bg-gray-600',
};
const healthBorder: Record<string, string> = {
  Healthy:      'border-green-500/30',
  Degraded:     'border-yellow-500/30',
  Unhealthy:    'border-red-500/30',
  'Scaled Down':'border-gray-500/30',
  Unknown:      'border-gray-600/30',
};
const healthText: Record<string, string> = {
  Healthy:      'text-green-400',
  Degraded:     'text-yellow-400',
  Unhealthy:    'text-red-400',
  'Scaled Down':'text-gray-400',
  Unknown:      'text-gray-500',
};
const phaseColor: Record<string, string> = {
  Running:   'text-green-400',
  Succeeded: 'text-blue-400',
  Pending:   'text-yellow-400',
  Failed:    'text-red-400',
  Unknown:   'text-gray-400',
};
const stateColor: Record<string, string> = {
  running:    'text-green-400',
  waiting:    'text-yellow-400',
  terminated: 'text-red-400',
};

function formatAge(seconds: number) {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
  return `${Math.floor(seconds / 86400)}d ${Math.floor((seconds % 86400) / 3600)}h`;
}

/* ─── Skeletons ─── */

function SkeletonCard() {
  return (
    <div className="bg-gray-900 border border-gray-800 rounded-xl p-5 animate-pulse">
      <div className="h-5 w-32 bg-gray-800 rounded mb-4" />
      <div className="h-4 w-24 bg-gray-800 rounded mb-2" />
      <div className="h-4 w-40 bg-gray-800 rounded" />
    </div>
  );
}

/* ─── Main Page ─── */

export default function ControlPlane() {
  const { data: status, isLoading } = useQuery({
    queryKey: ['controlplane'],
    queryFn: fetchControlPlaneStatus,
  });

  if (isLoading) {
    return (
      <div className="p-8 space-y-6">
        <div className="h-8 w-64 bg-gray-800 rounded animate-pulse" />
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5">
          {Array.from({ length: 6 }).map((_, i) => <SkeletonCard key={i} />)}
        </div>
      </div>
    );
  }

  const components = status?.components ?? [];
  const healthy   = components.filter((c: any) => c.health === 'Healthy').length;
  const degraded  = components.filter((c: any) => c.health === 'Degraded').length;
  const unhealthy = components.filter((c: any) => c.health === 'Unhealthy').length;

  return (
    <div className="p-8 space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold">Control Plane</h1>
          <p className="text-sm text-gray-400 mt-1">
            Tekton component health &amp; status
            {status?.operator_managed && (
              <span className="ml-2 px-2 py-0.5 rounded-full text-xs font-medium bg-purple-500/20 text-purple-300 border border-purple-500/30">
                Operator Managed
              </span>
            )}
          </p>
        </div>
        <div className="flex items-center gap-4">
          {status?.tekton_version && status.tekton_version !== 'unknown' && (
            <span className="text-sm text-gray-400">
              Tekton <span className="font-mono text-blue-400">{status.tekton_version}</span>
            </span>
          )}
        </div>
      </div>

      {/* Summary Banner */}
      <div className={`flex items-center gap-4 rounded-xl border p-4 ${
        status?.overall_health === 'Healthy'
          ? 'bg-green-500/5 border-green-500/20'
          : status?.overall_health === 'Degraded'
            ? 'bg-yellow-500/5 border-yellow-500/20'
            : 'bg-red-500/5 border-red-500/20'
      }`}>
        <div className={`w-3 h-3 rounded-full ${healthColor[status?.overall_health ?? 'Unknown']} ${
          status?.overall_health === 'Healthy' ? 'animate-pulse-dot' : ''
        }`} />
        <span className={`text-lg font-semibold ${healthText[status?.overall_health ?? 'Unknown']}`}>
          {status?.overall_health ?? 'Unknown'}
        </span>
        <span className="text-sm text-gray-400 ml-2">
          {healthy} healthy · {degraded} degraded · {unhealthy} unhealthy ·{' '}
          {components.length} total components
        </span>
      </div>

      {/* No components */}
      {components.length === 0 && (
        <div className="flex flex-col items-center justify-center py-24 text-gray-500">
          <span className="text-5xl mb-4">🔧</span>
          <p className="text-lg font-medium">No Tekton components detected</p>
          <p className="text-sm mt-1">Are you running in the tekton-pipelines namespace?</p>
        </div>
      )}

      {/* Component Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-5">
        {components.map((comp: any) => (
          <ComponentCard key={comp.component + comp.namespace} comp={comp} />
        ))}
      </div>
    </div>
  );
}

/* ─── Component Card ─── */

function ComponentCard({ comp }: { comp: any }) {
  return (
    <div className={`bg-gray-900 border rounded-xl overflow-hidden ${healthBorder[comp.health] || 'border-gray-800'}`}>
      {/* Header */}
      <div className="px-5 pt-5 pb-3 flex items-start justify-between">
        <div className="flex items-center gap-3 min-w-0">
          <div className={`w-2.5 h-2.5 rounded-full flex-shrink-0 ${healthColor[comp.health] || 'bg-gray-600'} ${
            comp.health === 'Healthy' ? 'animate-pulse-dot' : ''
          }`} />
          <div className="min-w-0">
            <h3 className="text-base font-semibold text-white truncate">{comp.name}</h3>
            <p className="text-xs text-gray-500 font-mono truncate">{comp.namespace}/{comp.component}</p>
          </div>
        </div>
        <span className={`text-xs font-medium px-2 py-1 rounded-md flex-shrink-0 ${
          comp.health === 'Healthy'   ? 'bg-green-500/10 text-green-400' :
          comp.health === 'Degraded'  ? 'bg-yellow-500/10 text-yellow-400' :
          comp.health === 'Unhealthy' ? 'bg-red-500/10 text-red-400' :
                                        'bg-gray-700/50 text-gray-400'
        }`}>
          {comp.health}
        </span>
      </div>

      {/* Info */}
      <div className="px-5 pb-3 grid grid-cols-2 gap-y-2 text-sm">
        <div>
          <span className="text-gray-500 text-xs">Replicas</span>
          <p className="font-mono text-white">
            <span className={comp.ready_replicas >= comp.desired_replicas ? 'text-green-400' : 'text-yellow-400'}>
              {comp.ready_replicas}
            </span>
            <span className="text-gray-600"> / {comp.desired_replicas}</span>
          </p>
        </div>
        <div>
          <span className="text-gray-500 text-xs">Version</span>
          <p className="font-mono text-blue-400 truncate text-xs mt-0.5">{comp.version || '—'}</p>
        </div>
      </div>

      {/* Pods */}
      {comp.pods && comp.pods.length > 0 && (
        <div className="border-t border-gray-800">
          <div className="px-5 py-2">
            <span className="text-xs font-medium text-gray-500 uppercase tracking-wider">Pods</span>
          </div>
          <div className="px-5 pb-4 space-y-2">
            {comp.pods.map((pod: any) => (
              <PodRow key={pod.name} pod={pod} />
            ))}
          </div>
        </div>
      )}

      {/* Conditions (collapsed by default) */}
      {comp.conditions && comp.conditions.length > 0 && (
        <details className="border-t border-gray-800 group">
          <summary className="px-5 py-2.5 text-xs text-gray-500 cursor-pointer hover:text-gray-300 select-none">
            Conditions ({comp.conditions.length})
          </summary>
          <div className="px-5 pb-3 space-y-1">
            {comp.conditions.map((cond: any, i: number) => (
              <div key={i} className="flex items-center gap-2 text-xs">
                <span className={cond.status === 'True' ? 'text-green-400' : 'text-red-400'}>●</span>
                <span className="text-gray-300">{cond.type}</span>
                {cond.reason && <span className="text-gray-600">— {cond.reason}</span>}
              </div>
            ))}
          </div>
        </details>
      )}
    </div>
  );
}

/* ─── Pod Row ─── */

function PodRow({ pod }: { pod: any }) {
  return (
    <div className="bg-gray-800/50 rounded-lg px-3 py-2">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 min-w-0">
          <span className={`text-xs ${pod.ready ? 'text-green-400' : 'text-yellow-400'}`}>
            {pod.ready ? '✓' : '○'}
          </span>
          <span className="text-xs font-mono text-gray-300 truncate">{pod.name}</span>
        </div>
        <div className="flex items-center gap-3 flex-shrink-0 text-xs">
          <span className={phaseColor[pod.phase] || 'text-gray-400'}>{pod.phase}</span>
          {pod.restarts > 0 && (
            <span className={`${pod.restarts > 3 ? 'text-red-400' : 'text-yellow-400'}`}>
              {pod.restarts} restart{pod.restarts !== 1 ? 's' : ''}
            </span>
          )}
          <span className="text-gray-600">{formatAge(pod.age)}</span>
        </div>
      </div>

      {/* Container details */}
      {pod.containers && pod.containers.length > 0 && (
        <div className="mt-1.5 pl-5 space-y-0.5">
          {pod.containers.map((c: any) => (
            <div key={c.name} className="flex items-center gap-2 text-xs">
              <span className={stateColor[c.state] || 'text-gray-500'}>■</span>
              <span className="text-gray-400">{c.name}</span>
              <span className={stateColor[c.state] || 'text-gray-500'}>{c.state}</span>
              {c.reason && <span className="text-gray-600">({c.reason})</span>}
            </div>
          ))}
        </div>
      )}

      {/* Node */}
      {pod.node && (
        <div className="mt-1 pl-5 text-xs text-gray-600">
          node: {pod.node}{pod.ip ? ` · ${pod.ip}` : ''}
        </div>
      )}
    </div>
  );
}
