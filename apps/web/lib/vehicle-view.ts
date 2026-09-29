import type { Vehicle, World } from './types';

export type CameraMode = 'orbit' | 'follow' | 'driver';
export const heading = (direction: string): [number, number] => {
 switch (direction) { case 'W': return [-1, 0]; case 'N': return [0, -1]; case 'S': return [0, 1]; default: return [1, 0]; }
};
export function vehicleCode(id: string): string { return `T–${id.split('_').at(-1)?.slice(-4) ?? '0000'}`; }
export function chooseVehicle(vehicles: Vehicle[]): Vehicle | undefined {
 // A recent arrival gives the visitor the longest remaining journey.
 return [...vehicles].sort((a, b) => a.position - b.position || a.id.localeCompare(b.id))[0];
}
export function nextSignal(vehicle: Vehicle, world: World): { label: string; color: string; distance: number } {
 const intersection = world.intersections.find(i => i.id === vehicle.route[vehicle.next]);
 if (!intersection) return { label: 'Leaving the district', color: 'muted', distance: 0 };
 const ns = vehicle.direction === 'N' || vehicle.direction === 'S';
 const hasAxis = intersection.phase.startsWith(ns ? 'NS_' : 'EW_');
 const color = hasAxis && intersection.phase.endsWith('GREEN') ? 'green' : hasAxis && intersection.phase.endsWith('YELLOW') ? 'amber' : 'red';
 const stop = vehicle.stops?.[vehicle.next] ?? (vehicle.next - 1) * 40 - 7;
 if (vehicle.position > stop + 0.001) return { label: 'Clearing the junction', color: 'muted', distance: 0 };
 const waiting = vehicle.speed < 0.1 ? ({ QUEUE: 'Green · queue ahead', JUNCTION_BUSY: 'Green · junction occupied', EXIT_BLOCKED: 'Green · exit blocked', SIGNAL: 'Waiting for clearance' } as const)[vehicle.waitReason!] : undefined;
 return { label: color === 'green' ? (waiting ?? 'Right of way') : color === 'amber' ? 'Signal clearing' : 'Stop at the signal', color, distance: Math.max(0, stop - vehicle.position) };
}
