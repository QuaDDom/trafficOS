'use client';
import { useMemo } from 'react';
import { Canvas } from '@react-three/fiber';
import { Html, Line } from '@react-three/drei';
import * as THREE from 'three';
import type { Intersection, World, StreetMap } from '@/lib/types';
import type { CameraMode } from '@/lib/vehicle-view';
import { CameraRig } from './city/camera-rig';
import { Scenery, Block } from './city/scenery';
import { Vehicles } from './city/vehicles';
import { LargeScenery } from './city/large-scenery';
import { LargeSignals } from './city/large-signals';
function Signal({ intersection:i, selected, onSelect, labels, roads }: {intersection:Intersection;selected:boolean;onSelect:()=>void;labels:boolean;roads:StreetMap['roads']}){
 return <group position={[i.x,0,i.z]}>
  {selected&&labels&&<Line points={[[-6,.16,-6],[6,.16,-6],[6,.16,6],[-6,.16,6],[-6,.16,-6]]} color="#ec6a36" lineWidth={2}/>}
  {[{x:-5.8,z:-5.8,ns:true,angle:Math.PI},{x:5.8,z:5.8,ns:true,angle:0},{x:-5.8,z:5.8,ns:false,angle:-Math.PI/2},{x:5.8,z:-5.8,ns:false,angle:Math.PI/2}].filter((s,index)=>roads.some(r=>{if(r.from!==i.id&&r.to!==i.id)return false;const end=r.from===i.id?r.b:r.a;return index===0?end.z<i.z:index===1?end.z>i.z:index===2?end.x<i.x:end.x>i.x;})).map((s,index)=>{
   const axis=i.phase.startsWith(s.ns?'NS_':'EW_');const active=axis&&i.phase.endsWith('GREEN')?2:axis&&i.phase.endsWith('YELLOW')?1:0;
   return <group key={index} position={[s.x,0,s.z]} rotation={[0,s.angle,0]}><Block at={[0,2.2,0]} size={[.16,4.4,.16]} color="#465448"/><Block at={[0,4.6,0]} size={[.66,1.65,.36]} color="#30372d"/>{['#ed573b','#f1c258','#9ecb71'].map((color,k)=><mesh key={color} position={[0,5.13-k*.52,.2]}><circleGeometry args={[.19,16]}/><meshStandardMaterial color={k===active?color:'#262c24'} emissive={color} emissiveIntensity={k===active?2:0} side={THREE.DoubleSide}/></mesh>)}</group>;
  })}
  {labels&&<Html position={[0,.5,-10.5]} center zIndexRange={[8,0]}><button onClick={onSelect} className={`intersection-marker ${selected?'selected':''}`} aria-label={`Select ${i.id}`}>{i.id.replace('INT-','')}{i.controllerMode==='EMERGENCY_OVERRIDE'?' +':''}</button></Html>}
 </group>;
}
export default function City({world,selected,onSelect,zoom,topView,reset,mode,vehicleId,onVehicleSelect,focus}:{focus:number;world:World|null;selected:string;onSelect:(id:string)=>void;zoom:number;topView:boolean;reset:number;mode:CameraMode;vehicleId:string|null;onVehicleSelect:(id:string)=>void}){
 const positions=useMemo(()=>new Map<string,THREE.Vector3>(),[]);
 const headings=useMemo(()=>new Map<string,number>(),[]);
 const extent=Math.max(world?.map.width??160,world?.map.depth??160),large=(world?.map.nodes.length??0)>16;
 const focusPoint=world?.intersections.find(i=>i.id===selected);
 const vehicle=world?.vehicles.find(v=>v.id===vehicleId);
 return <Canvas key={world?.map.id??'loading'} camera={{position:[138,145,162],fov:40,near:5,far:800}} shadows="percentage" dpr={[1,1.7]} gl={{antialias:true}} onCreated={({gl})=>{gl.toneMapping=THREE.ACESFilmicToneMapping;gl.toneMappingExposure=1.05;}}>
  <color attach="background" args={['#bec6b5']}/><fog attach="fog" args={mode==='orbit'?['#bec6b5',Math.max(500,extent*3),Math.max(1000,extent*7)]:['#bec6b5',100,280]}/>
  <hemisphereLight args={['#e5ece2','#8c8064',1.7]}/><ambientLight intensity={.35}/><directionalLight position={[-extent*.35,extent*.7,extent*.3]} intensity={2.5} color="#ffdfad" castShadow shadow-mapSize={[2048,2048]} shadow-camera-left={-extent*.65} shadow-camera-right={extent*.65} shadow-camera-top={extent*.65} shadow-camera-bottom={-extent*.65} shadow-camera-far={extent*3} shadow-normalBias={.07} shadow-bias={-.0002}/>
  {world?.map&&(large?<LargeScenery key={"scenery-"+world.map.id} map={world.map}/>:<Scenery key={"scenery-"+world.map.id} map={world.map}/>)}
  {large&&world?<LargeSignals key={"signals-"+world.map.id} map={world.map} intersections={world.intersections} selected={selected} labels={mode==='orbit'} onSelect={onSelect}/>:world?.intersections.map(i=><Signal key={i.id} intersection={i} selected={i.id===selected} onSelect={()=>onSelect(i.id)} labels={mode==='orbit'} roads={world.map?.roads??[]}/>)}
  <Vehicles vehicles={world?.vehicles??[]} positions={positions} headings={headings} selectedId={vehicleId} inside={mode==='driver'} onSelect={onVehicleSelect}/>
  <CameraRig focus={focus} focusPoint={focusPoint} mode={mode} vehicle={vehicle} positions={positions} headings={headings} zoom={zoom} topView={topView} reset={reset} extent={Math.max(world?.map?.width??160,world?.map?.depth??160)}/>
  {mode==='orbit'&&world?.corridors.filter(c=>c.status==='ACTIVE').flatMap(c=>c.reservations.map(r=>{const i=world.intersections.find(i=>i.id===r.intersectionId);return r.released||!i?null:<Line key={`${c.id}-${r.intersectionId}`} points={[[i.x-14,.22,i.z+2],[i.x+14,.22,i.z+2]]} color="#8bc8be" lineWidth={2.5} dashed dashSize={2} gapSize={1}/>;}))}
 </Canvas>;
}
