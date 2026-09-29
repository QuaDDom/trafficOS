'use client';
import { memo, useEffect, useMemo } from 'react';
import * as THREE from 'three';
import type { StreetMap } from '@/lib/types';
type Vec3 = [number, number, number];
export function Block({ at, size, color, rotation, material }: { at: Vec3; size: Vec3; color?: string; rotation?: Vec3; material?: THREE.Material | THREE.Material[] }) {
 return <mesh position={at} rotation={rotation} material={material} castShadow receiveShadow><boxGeometry args={size} />{!material && <meshStandardMaterial color={color ?? '#d0c7b1'} roughness={0.88} />}</mesh>;
}
function texture(paint: (ctx: CanvasRenderingContext2D) => void, width = 256, height = 256): THREE.CanvasTexture {
 const canvas = document.createElement('canvas'); canvas.width = width; canvas.height = height; const ctx = canvas.getContext('2d')!; paint(ctx); const result = new THREE.CanvasTexture(canvas); result.colorSpace = THREE.SRGBColorSpace; result.anisotropy = 4; return result;
}
export const StreetSign = memo(function StreetSign({ text, at, rotation = [0,0,0], color = '#244840', width = 5, height = 1.2 }: { text: string; at: Vec3; rotation?: Vec3; color?: string; width?: number; height?: number }) {
 const map = useMemo(() => texture(ctx => { ctx.fillStyle = color; ctx.fillRect(0,0,512,128); ctx.strokeStyle='#e6dfc6';ctx.lineWidth=3;ctx.strokeRect(5,5,502,118);ctx.fillStyle='#f5edda';ctx.font='bold 42px Arial';ctx.textAlign='center';ctx.textBaseline='middle';ctx.fillText(text,256,66,480); },512,128),[text,color]);
 useEffect(()=>()=>map.dispose(),[map]);
 return <group position={at} rotation={rotation}>{[0,Math.PI].map(angle=><mesh key={angle} rotation={[0,angle,0]}><planeGeometry args={[width,height]} /><meshStandardMaterial map={map} roughness={0.8} /></mesh>)}</group>;
});
const palettes = ['#b7775e','#d4c6a1','#97a8a0','#b3aba0','#839896','#c2a575','#d5cfc0','#9b6955'];
const Building = memo(function Building({ x,z,h,w,d,style,index }: {x:number;z:number;h:number;w:number;d:number;style:number;index:number}) {
 const materials = useMemo(()=>{
  const map=texture(ctx=>{
   ctx.fillStyle=palettes[style];ctx.fillRect(0,0,256,256);
   for(let y=0;y<256;y+=16){ctx.fillStyle='#352f2513';ctx.fillRect(0,y,256,1);for(let x=(y%32?16:0);x<256;x+=32)ctx.fillRect(x,y,1,16);}
   for(let row=0;row<4;row++)for(let col=0;col<3;col++){
    const left=col*80+25,top=row*62+13;ctx.fillStyle='#574e4266';ctx.fillRect(left-3,top-3,40,46);ctx.fillStyle=(row+col+index)%5===0?'#c9bb7f':'#39494b';ctx.fillRect(left,top,33,38);ctx.fillStyle='#d9d0bb';ctx.fillRect(left+15,top,2,38);ctx.fillRect(left,top+19,33,2);ctx.fillStyle='#ede1c488';ctx.fillRect(left-3,top+39,40,3);
   }
  });
  const facade=new THREE.MeshStandardMaterial({map,roughness:.92});const roof=new THREE.MeshStandardMaterial({color:'#a9a99b',roughness:1});return [facade,facade,roof,roof,facade,facade];
 },[style,index]);
 useEffect(()=>()=>{for(const m of new Set(materials)){m.map?.dispose();m.dispose();}},[materials]);
 return <group>
  <Block at={[x,h/2+.45,z]} size={[w,h,d]} material={materials}/>
  <Block at={[x,h+.7,z]} size={[w+.6,.5,d+.6]} color="#bfb9a7"/>
  <Block at={[x,h+1.05,z]} size={[w-.8,.2,d-.8]} color="#757970"/>
  <Block at={[x+1,h+1.5,z-1]} size={[2.3,.9,1.8]} color="#7c837d"/>
  {index%4===0&&<group position={[x-2,h+1,z+1]}><mesh position={[0,1.1,0]}><cylinderGeometry args={[1.15,1.15,1.9,10]}/><meshStandardMaterial color="#5c655a"/></mesh><mesh position={[0,2.25,0]}><coneGeometry args={[1.4,.55,10]}/><meshStandardMaterial color="#777b63"/></mesh></group>}
  {h<15&&<><Block at={[x,1.9,z+d/2+.08]} size={[w-.5,3,.2]} color="#253e3c"/><Block at={[x,3.5,z+d/2+.6]} size={[w+1,.2,1.6]} color={index%2?'#d89a62':'#537c6a'} rotation={[.12,0,0]}/><StreetSign text={['ALMACÉN','CAFÉ ESQUINA','TALLER 24','KIOSCO','RADIO SUR'][index%5]} at={[x,2.9,z+d/2+.22]} width={w-.8} height={.8} color={index%2?'#854b38':'#234b45'}/></>}
 </group>;
});
function Tree({at,variant=0}: {at:Vec3;variant?:number}){return <group position={at}><Block at={[0,1.3,0]} size={[.3,2.6,.3]} color="#76644e"/><mesh position={[0,3.3,0]} castShadow><icosahedronGeometry args={[1.8+variant*.2,1]}/><meshStandardMaterial color={['#6e8250','#87945a','#5e785b'][variant%3]} roughness={1}/></mesh></group>}
function RoadSurface({a,b,avenue,boulevard}:{a:{x:number;z:number};b:{x:number;z:number};avenue:boolean;boulevard:boolean}){
 const length=Math.hypot(b.x-a.x,b.z-a.z),angle=Math.atan2(-(b.z-a.z),b.x-a.x);
 return <group position={[(a.x+b.x)/2,.06,(a.z+b.z)/2]} rotation={[0,angle,0]}>
  <mesh rotation={[-Math.PI/2,0,0]} receiveShadow><planeGeometry args={[length,12]}/><meshStandardMaterial color={avenue?'#454b48':'#505751'} roughness={1}/></mesh>
  {Array.from({length:Math.max(0,Math.floor((length-14)/4))},(_,i)=>-length/2+9+i*4).map(x=><group key={x}>
   {[-.23,.23].map(z=><Block key={z} at={[x,.025,z]} size={[2.6,.025,.09]} color="#d9c17b"/>)}
   {[-4.8,4.8].map(z=><Block key={z} at={[x,.025,z]} size={[1.5,.025,.1]} color="#d9d8c5"/>)}
  </group>)}
  {boulevard&&avenue&&length>30&&<><Block at={[0,.2,0]} size={[length-22,.28,.8]} color="#babaa1"/><Block at={[0,.38,0]} size={[length-24,.2,.5]} color="#7e8b50"/></>}
 </group>;
}
export const Scenery=memo(function Scenery({map}:{map:StreetMap}){
 return <group>
  <Block at={[0,-1,0]} size={[map.width+4,2,map.depth+4]} color={map.id==='barrio'?'#929d74':'#b8b3a0'}/>
  {map.roads.map(r=><RoadSurface key={r.id} {...r} boulevard={map.id==='avenida'}/>)}
  {map.nodes.map(n=><group key={n.id}>
   <Block at={[n.x,.065,n.z]} size={[12,.04,12]} color="#4b514c"/>
   {map.roads.filter(r=>r.from===n.id||r.to===n.id).map(r=>{const other=r.from===n.id?r.b:r.a;const dx=Math.sign(other.x-n.x),dz=Math.sign(other.z-n.z);return <group key={r.id} position={[n.x+dx*8,.105,n.z+dz*8]} rotation={[0,Math.atan2(-dz,dx),0]}>{[-4,-2.6,-1.2,.2,1.6,3,4.4].map(z=><Block key={z} at={[0,0,z]} size={[2,.025,.65]} color="#e4dfc9"/>)}</group>})}
   <StreetSign text={n.z===0?'AV. CENTRAL':map.id==='barrio'?'BARRIO JARDÍN':n.z<0?'CALLE NORTE':'CALLE SUR'} at={[n.x+6.5,4,n.z+6.5]} width={4.5} height={.75}/>
  </group>)}
  {map.lots.map((lot,index)=>{
   const {x,z,width:w,depth:d,kind}=lot;const residential=map.id==='barrio';
   return <group key={index}>
    <Block at={[x,.23,z]} size={[w,.44,d]} color="#cbc6b4"/>
    <Block at={[x,.47,z]} size={[w-1.5,.04,d-1.5]} color={kind==='park'?'#8b9c6a':'#bcbbaa'}/>
    {kind==='buildings'&&[0,1,2].map(k=>{const h=residential?4+(index+k)%4:map.id==='avenida'?9+(index*3+k*5)%17:7+(index*7+k*3)%13;return <Building key={k} x={x+(k===0?-w/4:w/4)} z={z+(k===0?0:k===1?-d/4:d/4)} w={w/2-3.5} d={k===0?d-6:d/2-4} h={h} style={(index*3+k)%palettes.length} index={index*3+k}/>})}
    {kind==='park'&&<><Block at={[x,.52,z]} size={[2.6,.05,d-2]} color="#d4c7a5"/><Block at={[x,.54,z]} size={[w-2,.05,2.2]} color="#d4c7a5"/>{[-1,1].flatMap(a=>[-1,1].map(b=><Tree key={a+','+b} at={[x+a*w*.3,.5,z+b*d*.3]} variant={(index+3+a+b)%3}/>))}<StreetSign text="PLAZA / RESPIRÁ" at={[x,2.8,z+d/2-1]} width={7} height={.8} color="#516448"/></>}
    {kind==='hospital'&&<><Block at={[x,4.5,z]} size={[w-4,8,d-6]} color="#dedbc9"/><Block at={[x,9.3,z]} size={[w-10,2,d-12]} color="#abc0ac"/><Block at={[x,10.4,z]} size={[2,.2,8]} color="#cf684c"/><Block at={[x,10.42,z]} size={[8,.2,2]} color="#cf684c"/><StreetSign text="HOSPITAL CENTRAL" at={[x,5,z-d/2+2.95]} rotation={[0,Math.PI,0]} width={w-6} height={1.5}/></>}
    {kind==='depot'&&<><Block at={[x,3,z]} size={[w-4,5.5,d-5]} color="#b77653"/><Block at={[x,6,z]} size={[w-2,.6,d-3]} color="#605d4c"/><StreetSign text="EMERGENCIAS / 107" at={[x,4,z+d/2-2.45]} width={w-6} height={1.4} color="#9a4a32"/></>}
    {[-1,1].map(k=><group key={k}><Tree at={[x+k*(w/2-2),.45,z+d/2-1.5]} variant={(index+3+k)%3}/><Block at={[x+k*(w/2-2),1,z-d/2+1]} size={[2.5,.2,.7]} color="#897558"/><Block at={[x+w/2-1,2.9,z+k*(d/2-2)]} size={[.15,5.8,.15]} color="#68725f"/><Block at={[x+w/2-2,5.8,z+k*(d/2-2)]} size={[2,.15,.4]} color="#d9cda5"/></group>)}
   </group>;
  })}
  <StreetSign text={map.id==='barrio'?'BAJÁ UN CAMBIO.':map.id==='avenida'?'LA CIUDAD SE MUEVE.':'BUEN VIAJE.'} at={[map.columns[0]+20,19,map.rows[0]-25]} width={22} height={4.5} color="#be633d"/>
  <Block at={[map.columns[0]+20,10,map.rows[0]-25.5]} size={[.35,18,.35]} color="#4e5d4c"/>
  <mesh rotation={[-Math.PI/2,0,0]} position={[0,-2.05,0]} receiveShadow><planeGeometry args={[1500,1500]}/><meshStandardMaterial color="#aab09b" roughness={1}/></mesh>
 </group>;
},(previous,next)=>previous.map.id===next.map.id);
