import { BrowserRouter as Router, Routes, Route, Link, useLocation } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import Dashboard from './pages/Dashboard';
import Pipelines from './pages/Pipelines';
import Costs from './pages/Costs';
import Traces from './pages/Traces';
import Insights from './pages/Insights';
import ControlPlane from './pages/ControlPlane';
import './App.css';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchInterval: 10000,
      refetchOnWindowFocus: true,
      retry: 2,
    },
  },
});

const navigation = [
  { name: 'Dashboard',      path: '/',              icon: '📊' },
  { name: 'Pipelines',      path: '/pipelines',     icon: '🔄' },
  { name: 'Control Plane',  path: '/controlplane',  icon: '🏗️' },
  { name: 'Cost Analysis',  path: '/costs',         icon: '💰' },
  { name: 'Traces',         path: '/traces',        icon: '🔍' },
  { name: 'AI Insights',    path: '/insights',      icon: '🤖' },
];

function Sidebar() {
  const location = useLocation();
  return (
    <nav className="w-64 bg-gray-900 min-h-screen flex flex-col border-r border-gray-800">
      {/* Logo */}
      <div className="px-5 py-5 flex items-center border-b border-gray-800">
        <div className="h-9 w-9 rounded-lg bg-blue-600 flex items-center justify-center text-white font-bold text-lg">T</div>
        <div className="ml-3">
          <div className="text-sm font-bold text-white leading-tight">Tekton</div>
          <div className="text-xs text-gray-400 leading-tight">Observability</div>
        </div>
      </div>
      {/* Nav Items */}
      <div className="flex-1 py-4 space-y-1 px-3">
        {navigation.map((item) => {
          const active = location.pathname === item.path;
          return (
            <Link key={item.name} to={item.path}
              className={`flex items-center px-3 py-2.5 rounded-lg text-sm font-medium ${
                active
                  ? 'bg-blue-600 text-white shadow-lg shadow-blue-600/30'
                  : 'text-gray-400 hover:bg-gray-800 hover:text-white'
              }`}
            >
              <span className="text-lg mr-3">{item.icon}</span>
              {item.name}
            </Link>
          );
        })}
      </div>
      {/* Footer */}
      <div className="px-5 py-4 border-t border-gray-800">
        <div className="flex items-center text-xs text-gray-500">
          <span className="w-2 h-2 rounded-full bg-green-500 mr-2 animate-pulse-dot"></span>
          Connected to cluster
        </div>
      </div>
    </nav>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Router>
        <div className="flex h-screen bg-gray-950 text-white overflow-hidden">
          <Sidebar />
          <main className="flex-1 overflow-y-auto bg-gray-950">
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/pipelines" element={<Pipelines />} />
              <Route path="/controlplane" element={<ControlPlane />} />
              <Route path="/costs" element={<Costs />} />
              <Route path="/traces" element={<Traces />} />
              <Route path="/insights" element={<Insights />} />
            </Routes>
          </main>
        </div>
      </Router>
    </QueryClientProvider>
  );
}

export default App;
