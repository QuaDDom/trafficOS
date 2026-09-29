export function number(value: number | undefined, digits = 0): string { return value === undefined || !Number.isFinite(value) ? '—' : value.toLocaleString('es-AR', { maximumFractionDigits: digits, minimumFractionDigits: digits }); }
export function clock(seconds: number): string { const whole = Math.floor(seconds); return `${Math.floor(whole / 3600).toString().padStart(2, '0')}:${Math.floor(whole / 60 % 60).toString().padStart(2, '0')}:${(whole % 60).toString().padStart(2, '0')}`; }
export function time(value: string): string { return new Date(value).toLocaleTimeString('en-GB', { hour12: false }); }
export function human(value: string): string { return value.toLowerCase().replaceAll('_', ' '); }
export function eventColor(type: string): string { return /rejected|unavailable|offline/.test(type) ? 'red' : /emergency/.test(type) ? 'blue' : /optimizer/.test(type) ? 'purple' : /signal/.test(type) ? 'green' : 'muted'; }
export function appendUnique<T extends { id: string }>(items: T[], incoming: T, limit: number): T[] { if (items.some(item => item.id === incoming.id)) return items; return [...items, incoming].slice(-limit); }

export function appendUniqueBatch<T extends { id: string }>(items: T[], incoming: T[], limit: number): T[] { const ids = new Set(items.map(item => item.id)); const added = incoming.filter(item => { if(ids.has(item.id)) return false; ids.add(item.id); return true; }); return added.length ? [...items, ...added].slice(-limit) : items; }
