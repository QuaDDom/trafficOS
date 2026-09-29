'use client';
import { useEffect, useMemo, useRef } from 'react';
import { useFrame, useThree } from '@react-three/fiber';
import { OrbitControls } from '@react-three/drei';
import type { OrbitControls as OrbitControlsImpl } from 'three-stdlib';
import * as THREE from 'three';
import { heading, type CameraMode } from '@/lib/vehicle-view';
import type { Vehicle } from '@/lib/types';

export type RenderedPositions = Map<string, THREE.Vector3>;
export type RenderedHeadings = Map<string, number>;
export function CameraRig({ mode, vehicle, positions, headings, zoom, topView, reset, extent = 160, focus, focusPoint }: { focus:number; focusPoint?:{x:number;z:number}; extent?: number; mode: CameraMode; vehicle?: Vehicle; positions: RenderedPositions; headings: RenderedHeadings; zoom: number; topView: boolean; reset: number }) {
 const { camera, size } = useThree();
 const first = useRef(true), previous = useRef('');
 const target = useMemo(() => new THREE.Vector3(), []), look = useMemo(() => new THREE.Vector3(), []);
 const vehicleRef = useRef(vehicle); vehicleRef.current = vehicle;
 const controls=useRef<OrbitControlsImpl>(null),point=useRef(focusPoint);point.current=focusPoint;
 useEffect(() => {
  if (camera instanceof THREE.PerspectiveCamera) {
   camera.fov = mode === 'driver' ? 68 : mode === 'follow' ? 56 : size.width < 600 ? 52 : 40;
   // A cabin-sized near plane loses ground-layer depth precision at orbit distance.
   camera.near = mode === 'orbit' ? Math.max(5,extent/48) : mode === 'follow' ? 0.5 : 0.15;
   camera.far = mode === 'orbit' ? Math.max(800,extent*8) : 300;
   camera.updateProjectionMatrix();
  }
  first.current = true;
  if (mode === 'orbit') {
   const centered=focus>0&&point.current,px=centered?centered.x:0,pz=centered?centered.z:0;
   const distance = Math.max(1, 450 / size.width) * (centered?.55:Math.max(1,extent/160)) * 3.65 / zoom;
   camera.position.set(px+(topView?0:138*distance),topView?245*distance:145*distance,pz+(topView?.1:162*distance));
   camera.lookAt(px,0,pz);controls.current?.target.set(px,0,pz);controls.current?.update();
  }
 }, [camera, mode, zoom, topView, reset, size.width, extent,focus]);
 useFrame((_,dt) => {
  if (mode === 'orbit') return;
  const v = vehicleRef.current; if (!v) return;
  const p = positions.get(v.id); if (!p) return;
  const angle = headings.get(v.id) ?? v.heading;
  const [dx,dz] = angle === undefined ? heading(v.direction) : [Math.cos(angle),-Math.sin(angle)];
  if (previous.current !== v.id) { first.current = true; previous.current = v.id; }
  const bus = v.type === 'BUS';
  const behind = mode === 'driver' ? (bus ? -2.1 : 0.2) : bus ? 8.5 : 6.5;
  const side = mode === 'driver' ? 0 : 2.7;
  const eyeHeight = bus ? 2.15 : v.type === 'SUV' || v.type === 'AMBULANCE' ? 1.8 : 1.65;
  const height = mode === 'driver' ? eyeHeight : bus ? 3.8 : 3.2;
  target.set(p.x - dx * behind - dz * side, height, p.z - dz * behind + dx * side);
  const ahead = mode === 'driver' ? 22 : 2;
  look.set(p.x + dx * ahead, mode === 'driver' ? eyeHeight + .15 : bus ? 1.3 : .8, p.z + dz * ahead);
  if (first.current || mode === 'driver') camera.position.copy(target); else camera.position.lerp(target, 1 - Math.exp(-dt * 6));
  camera.lookAt(look); first.current = false;
 }, -1);
 return mode === 'orbit' ? <OrbitControls ref={controls} minDistance={45} maxDistance={Math.max(600,extent*5)} minPolarAngle={0.05} maxPolarAngle={1.38} enablePan screenSpacePanning={false} enableDamping dampingFactor={0.09} /> : null;
}
