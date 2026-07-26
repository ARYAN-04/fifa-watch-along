export function getBackendUrl(): string {
  const url = process.env.BACKEND_URL || 'http://localhost:8000';
  return url.replace(/\/+$/, '');
}
