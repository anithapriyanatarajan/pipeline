import axios from 'axios';

// Use relative URLs — works behind port-forward and in-cluster
const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
});

// Metrics
export const fetchOverviewMetrics = async () => {
  const { data } = await api.get('/metrics/overview');
  return data;
};
export const fetchPipelineMetrics = async () => {
  const { data } = await api.get('/metrics/pipelines');
  return data;
};
export const fetchTaskMetrics = async () => {
  const { data } = await api.get('/metrics/tasks');
  return data;
};
export const fetchMetricsHistory = async (duration: string) => {
  const { data } = await api.get('/metrics/history', { params: { duration } });
  return data;
};

// Costs
export const fetchCostBreakdown = async () => {
  const { data } = await api.get('/costs/breakdown');
  return data;
};
export const fetchCostTrend = async (duration: string = '24h') => {
  const { data } = await api.get('/costs/trend', { params: { duration } });
  return data;
};

// Traces
export const fetchTraces = async () => {
  const { data } = await api.get('/traces');
  return data;
};

// Insights
export const fetchInsights = async () => {
  const { data } = await api.get('/insights');
  return data;
};
export const fetchAnomalies = async () => {
  const { data } = await api.get('/insights/anomalies');
  return data;
};
export const fetchRecommendations = async () => {
  const { data } = await api.get('/insights/recommendations');
  return data;
};

// Control Plane
export const fetchControlPlaneStatus = async () => {
  const { data } = await api.get('/controlplane/status');
  return data;
};

// WebSocket
const getWsBase = () => {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${window.location.host}`;
};
export const createMetricsStream = (onMessage: (data: any) => void) => {
  const ws = new WebSocket(`${getWsBase()}/api/v1/stream/metrics`);
  ws.onmessage = (e) => onMessage(JSON.parse(e.data));
  ws.onerror = (err) => console.error('WS error:', err);
  return ws;
};

export default api;
