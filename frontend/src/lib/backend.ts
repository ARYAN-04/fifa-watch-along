export function getBackendUrl(): string {
  const url = process.env.BACKEND_URL;
  if (!url) {
    throw new Error('BACKEND_URL environment variable is not set');
  }
  return url.replace(/\/+$/, '');
}
