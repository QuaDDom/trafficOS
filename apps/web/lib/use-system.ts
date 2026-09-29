'use client';
import { useCallback, useEffect, useRef, useState } from 'react';
import { API, request } from './api';
import { appendUniqueBatch } from './format';
import type { Decision, DomainEvent, Health, World } from './types';
export function useSystem() {
 const [world, setWorld] = useState<World | null>(null);
 const [events, setEvents] = useState<DomainEvent[]>([]);
 const [decisions, setDecisions] = useState<Decision[]>([]);
 const [connection, setConnection] = useState<'connecting' | 'live' | 'reconnecting'>('connecting');
 const [health, setHealth] = useState<Health | null>(null);
 const [lastMessage, setLastMessage] = useState<string | null>(null);
 const [error, setError] = useState<string | null>(null);
 const [busy, setBusy] = useState(false);
 const epoch = useRef('');
 const pending = useRef(0);
 const reconnect = useRef<() => void>(() => {});
 useEffect(() => {
  let disposed = false, socket: WebSocket | null = null, attempts = 0;
  let timer: ReturnType<typeof setTimeout>;
  const connect = () => {
   if (disposed) return;
   socket = new WebSocket(API.replace(/^http/, 'ws') + '/ws');
   socket.onopen = () => { attempts = 0; setConnection('live'); };
   socket.onmessage = ({ data }: MessageEvent<string>) => {
    if (disposed) return;
    try {
     const message = JSON.parse(data) as { type: string; epoch?: string; payload: World | DomainEvent | DomainEvent[] };
     setLastMessage(new Date().toISOString());
     if (message.type === 'snapshot') { const next = message.payload as World; if (epoch.current && epoch.current !== next.epoch) { setEvents([]); setDecisions([]); } epoch.current = next.epoch; setWorld(next); }
     if (message.type === 'event' || message.type === 'events') { if (message.epoch && message.epoch !== epoch.current) return; const batch = message.type === 'events' ? message.payload as DomainEvent[] : [message.payload as DomainEvent]; setEvents(previous => appendUniqueBatch(previous, batch, 350)); const nextDecisions = batch.filter(event => event.type === 'optimizer.decision.created').map(event => event.payload as Decision); if (nextDecisions.length) setDecisions(previous => appendUniqueBatch(previous, nextDecisions, 200)); }
    } catch { setError('An invalid realtime message was rejected.'); }
   };
   socket.onclose = () => { if (!disposed) { setConnection('reconnecting'); timer = setTimeout(connect, Math.min(15000, 500 * 2 ** attempts++) + Math.random() * 200); } };
   socket.onerror = () => socket?.close();
  };
  reconnect.current = () => socket?.close();
  connect();
  const refreshHealth = async () => { try { const result = await fetch(`${API}/api/v1/health`, { signal: AbortSignal.timeout(3000) }); const h = await result.json() as Health; if (!disposed) setHealth(h); } catch { if (!disposed) setHealth(null); } };
  void refreshHealth(); const healthTimer = setInterval(refreshHealth, 5000);
  return () => { disposed = true; clearTimeout(timer); clearInterval(healthTimer); socket?.close(); };
 }, []);
 const command = useCallback(async <T,>(path: string, body: unknown): Promise<T | undefined> => { pending.current++; setBusy(true); setError(null); try { return await request<T>(path, body, crypto.randomUUID()); } catch (cause) { setError(cause instanceof Error ? cause.message : 'Request failed'); return undefined; } finally { pending.current--; setBusy(pending.current > 0); } }, []);
 return { world, events, decisions, connection, health, lastMessage, error, setError, busy, command, reconnect: () => reconnect.current() };
}
