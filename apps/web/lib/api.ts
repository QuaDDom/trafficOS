export const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
export async function request<T>(path: string, body?: unknown, key?: string): Promise<T> {
 const res = await fetch(`${API}/api/v1/${path}`, { method: body === undefined ? 'GET' : 'POST', headers: { 'Content-Type': 'application/json', ...(key ? { 'Idempotency-Key': key } : {}) }, body: body === undefined ? undefined : JSON.stringify(body), signal: AbortSignal.timeout(15000) });
 const data: unknown = await res.json();
 if (!res.ok) { const error = data as { error?: { message?: string } }; throw new Error(error.error?.message || `Request failed (${res.status})`); }
 return data as T;
}
