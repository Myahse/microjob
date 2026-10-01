# MicroJob web (React + TypeScript)

Web app for clients and workers, plus the admin dashboard (`/admin`, admin accounts only).

```bash
npm install
npm run dev     # http://localhost:5173, proxies /api to http://localhost:8080
npm run build
```

Set `VITE_API_URL` when the API is served from another origin.
