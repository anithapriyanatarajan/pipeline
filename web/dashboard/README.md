# Tekton Dashboard - Frontend

React-based frontend for the Tekton Unified Observability Dashboard.

## Features

- 📊 Real-time metrics visualization with Recharts
- 💰 Cost analysis and trending
- 🤖 AI-powered insights display
- 🔄 WebSocket-powered live updates
- 📱 Responsive design with Tailwind CSS

## Development

### Prerequisites

- Node.js 18+ 
- npm or yarn

### Install Dependencies

```bash
npm install
```

### Run Development Server

```bash
npm run dev
```

The app will be available at http://localhost:5173

### Build for Production

```bash
npm run build
```

Build output will be in `build/` directory.

### Environment Variables

Create a `.env` file:

```env
REACT_APP_API_URL=http://localhost:8080/api/v1
```

## Project Structure

```
src/
├── App.tsx                 # Main app with routing
├── api/
│   └── dashboard.ts       # API client
└── pages/
    ├── Dashboard.tsx      # Overview page
    ├── Pipelines.tsx      # Pipeline metrics
    ├── Costs.tsx          # Cost analysis
    ├── Traces.tsx         # Distributed tracing
    └── Insights.tsx       # AI insights
```

## Technologies

- **React 18**: UI framework
- **TypeScript**: Type safety
- **React Router**: Navigation
- **TanStack Query**: Data fetching
- **Recharts**: Charts and graphs
- **Axios**: HTTP client
- **Tailwind CSS**: Styling
- **Vite**: Build tool

## Contributing

See [CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## License

Apache 2.0
