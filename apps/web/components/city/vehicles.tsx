'use client';
import { useMemo, useRef } from 'react';
import { useFrame, type ThreeEvent } from '@react-three/fiber';
import * as THREE from 'three';
import type { Vehicle } from '@/lib/types';
import type { RenderedPositions, RenderedHeadings } from './camera-rig';
import { createVehicleModel, createWheelGeometry, partNames, vehicleStyles, type VehicleStyle, type VehiclePart } from './vehicle-model';

const palette=['#e3d5b9','#af4f35','#47796a','#d5ab46','#59748b','#ece8dc','#294c49','#823e3d'];
const materials={paint:{roughness:.3,metalness:.35},glass:{color:'#42666f',roughness:.2,metalness:.18},trim:{color:'#222823',roughness:.85},metal:{color:'#b9beb5',roughness:.27,metalness:.75},head:{color:'#fff0cc',emissive:'#ffdf9e',emissiveIntensity:.75},tail:{color:'#ffffff',emissive:'#ff3116',emissiveIntensity:.35},accent:{color:'#e68b32',roughness:.4}};
const capacity=600;
export function Vehicles({vehicles,positions,headings,selectedId,inside,onSelect}:{vehicles:Vehicle[];positions:RenderedPositions;headings:RenderedHeadings;selectedId:string|null;inside:boolean;onSelect:(id:string)=>void}) {
 const models=useMemo(()=>vehicleStyles.map(style=>({style,parts:createVehicleModel(style)})),[]);
 const wheelGeometry=useMemo(createWheelGeometry,[]);
 const meshes=useRef(new Map<string,THREE.InstancedMesh>()), slots=useRef(new Map<VehicleStyle,Vehicle[]>());
 const current=useRef(vehicles);current.current=vehicles;
 const motion=useRef(new Map<string,{roll:number;angle:number}>());
 const scratch=useMemo(()=>({object:new THREE.Object3D(),color:new THREE.Color(),target:new THREE.Vector3(),offset:new THREE.Vector3(),axis:new THREE.Vector3(0,1,0)}),[]);
 useFrame((state,dt)=>{
  const {object,color,target,offset,axis}=scratch;
  let rimCount=0;
  const alive=new Set<string>();const counts:Record<VehicleStyle,number>={sedan:0,hatch:0,suv:0,ambulance:0,bus:0};
  for(const style of vehicleStyles)slots.current.set(style,[]);
  const place=(mesh:THREE.InstancedMesh|undefined,index:number,p:THREE.Vector3,angle:number,hidden:boolean,x=0,y=0,z=0,roll=0,steer=0,scale=1)=>{
   if(!mesh)return;object.rotation.set(0,angle+steer,0);if(roll)object.rotateZ(roll);
   offset.set(x,y,z).applyAxisAngle(axis,angle);object.position.copy(p).add(offset);object.scale.setScalar(hidden?0:scale);object.updateMatrix();mesh.setMatrixAt(index,object.matrix);
  };
  const tint=(mesh:THREE.InstancedMesh|undefined,index:number,value:string)=>mesh?.setColorAt(index,color.set(value));
  for(let idx=0;idx<current.current.length;idx++){
   const v=current.current[idx];alive.add(v.id);
   target.set(v.x,0,v.z);const p=positions.get(v.id)??target.clone();p.lerp(target,1-Math.exp(-Math.min(dt,.1)*12));positions.set(v.id,p);
   const desiredAngle=v.heading??({E:0,W:Math.PI,N:Math.PI/2,S:-Math.PI/2} as Record<string,number>)[v.direction];
   const previousAngle=headings.get(v.id)??desiredAngle,delta=Math.atan2(Math.sin(desiredAngle-previousAngle),Math.cos(desiredAngle-previousAngle));
   const angle=previousAngle+delta*(1-Math.exp(-Math.min(dt,.1)*12));headings.set(v.id,angle);
   const hidden=inside&&selectedId===v.id, ambulance=v.type==='AMBULANCE';
   const seed=Number(v.id.split('_').at(-1))||0,style:VehicleStyle=ambulance?'ambulance':v.type==='BUS'?'bus':v.type==='SUV'?'suv':seed%2?'hatch':'sedan';
   const slot=counts[style]++;slots.current.get(style)!.push(v);
   const baseColor=ambulance?'#eeeade':style==='bus'?'#d7dfbb':palette[seed%palette.length];
   for(const part of partNames){
    const mesh=meshes.current.get(`${style}:${part}`);place(mesh,slot,p,angle,hidden);
    if(part==='paint')tint(mesh,slot,baseColor);
    if(part==='tail')tint(mesh,slot,v.state==='WAITING'||v.speed<1?'#ff4226':'#762c24');
   }
   const bus=style==='bus';
   const last=motion.current.get(v.id)??{roll:0,angle};
   last.roll=(last.roll-v.speed*Math.min(dt,.1)/(bus?.39:.325))%(Math.PI*2);
   const yawRate=Math.atan2(Math.sin(angle-last.angle),Math.cos(angle-last.angle))/Math.max(dt,.001);
   const steer=v.speed>.4?THREE.MathUtils.clamp(Math.atan(yawRate*(bus?4.1:1.96)/v.speed),-.48,.48):0;
   last.angle=angle;motion.current.set(v.id,last);
   const axleWidth=bus?.97:ambulance?.795:style==='suv'?.795:.755;
   for(let wheel=0;wheel<4;wheel++){
    const front=wheel>=2,x=(front?1:-1)*(bus?2.05:.98),z=(wheel%2?1:-1)*axleWidth;
    place(meshes.current.get('tire'),idx*4+wheel,p,angle,hidden,x,bus?.47:.405,z,last.roll,front?steer:0,bus?1.2:1);
    if(!hidden&&state.camera.position.distanceToSquared(p)<3600)place(meshes.current.get('rim'),rimCount++,p,angle,false,x,bus?.47:.405,z,last.roll,front?steer:0,bus?1.2:1);
   }
   if(ambulance){for(let side=0;side<2;side++){
    const beacon=meshes.current.get('beacon');place(beacon,slot*2+side,p,angle,hidden,.53,2.065,side?.35:-.35);
    tint(beacon,slot*2+side,Math.sin(state.clock.elapsedTime*12+side*Math.PI)>0?(side?'#ff4e32':'#57c6ff'):'#263941');
   }}
  }
  for(const id of positions.keys())if(!alive.has(id)){positions.delete(id);headings.delete(id);motion.current.delete(id);}
  for(const [key,mesh] of meshes.current){
   mesh.count=key==='rim'?rimCount:key==='tire'?current.current.length*4:key==='beacon'?counts.ambulance*2:counts[key.split(':')[0] as VehicleStyle];
   mesh.instanceMatrix.needsUpdate=true;if(mesh.instanceColor)mesh.instanceColor.needsUpdate=true;
  }
 },-2);
 const ref=(key:string)=>(mesh:THREE.InstancedMesh|null)=>{if(mesh){mesh.instanceMatrix.setUsage(THREE.DynamicDrawUsage);meshes.current.set(key,mesh);}else meshes.current.delete(key);};
 const pick=(style:VehicleStyle)=>(e:ThreeEvent<MouseEvent>)=>{e.stopPropagation();if(e.instanceId!==undefined){const v=slots.current.get(style)?.[e.instanceId];if(v)onSelect(v.id);}};
 return <group>
  {models.flatMap(({style,parts})=>partNames.map((part:VehiclePart)=><instancedMesh key={`${style}:${part}`} ref={ref(`${style}:${part}`)} args={[parts[part],undefined,capacity]} castShadow={part==='paint'} frustumCulled={false} onClick={pick(style)}><meshStandardMaterial {...materials[part]}/></instancedMesh>))}
  <instancedMesh ref={ref('tire')} args={[wheelGeometry.tire,undefined,capacity*4]} frustumCulled={false}><meshStandardMaterial color="#20231f" roughness={.95}/></instancedMesh>
  <instancedMesh ref={ref('rim')} args={[wheelGeometry.rim,undefined,capacity*4]} frustumCulled={false}><meshStandardMaterial color="#bec3ba" metalness={.8} roughness={.25}/></instancedMesh>
  <instancedMesh ref={ref('beacon')} args={[undefined,undefined,capacity*2]} frustumCulled={false}><boxGeometry args={[.22,.11,.32]}/><meshBasicMaterial toneMapped={false}/></instancedMesh>
 </group>;
}
