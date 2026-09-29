'use client';
import { memo, useEffect, useLayoutEffect, useMemo, useRef } from 'react';
import * as THREE from 'three';
import type { StreetMap } from '@/lib/types';
import { StreetSign } from './scenery';
type Item={p:[number,number,number];s:[number,number,number];color:string};
function Instances({items,foliage=false,facade=false}:{items:Item[];foliage?:boolean;facade?:boolean}){
 const ref=useRef<THREE.InstancedMesh>(null);
 const texture=useMemo(()=>{
  if(!facade)return undefined;
  const canvas=document.createElement('canvas');canvas.width=128;canvas.height=256;
  const ctx=canvas.getContext('2d')!;ctx.fillStyle='#eeeeee';ctx.fillRect(0,0,128,256);
  for(let y=8;y<256;y+=32)for(let x=12;x<128;x+=32){ctx.fillStyle=(x+y)%5===0?'#f1df9e':'#41545a';ctx.fillRect(x,y,15,22);ctx.fillStyle='#cfcec1';ctx.fillRect(x+7,y,1,22);ctx.fillRect(x,y+10,15,1);}
  const map=new THREE.CanvasTexture(canvas);map.colorSpace=THREE.SRGBColorSpace;map.anisotropy=4;return map;
 },[facade]);
 useEffect(()=>()=>texture?.dispose(),[texture]);
 useLayoutEffect(()=>{const mesh=ref.current;if(!mesh)return;const o=new THREE.Object3D(),c=new THREE.Color();items.forEach((item,i)=>{o.position.set(...item.p);o.scale.set(...item.s);o.updateMatrix();mesh.setMatrixAt(i,o.matrix);mesh.setColorAt(i,c.set(item.color));});mesh.instanceMatrix.needsUpdate=true;if(mesh.instanceColor)mesh.instanceColor.needsUpdate=true;mesh.computeBoundingSphere();},[items]);
 return <instancedMesh ref={ref} args={[undefined,undefined,items.length]} castShadow receiveShadow>{foliage?<icosahedronGeometry args={[1,0]}/>:<boxGeometry args={[1,1,1]}/>}<meshStandardMaterial map={texture} roughness={.9}/></instancedMesh>;
}
export const LargeScenery=memo(function LargeScenery({map}:{map:StreetMap}){
 const data=useMemo(()=>{
  const boxes:Item[]=[],walls:Item[]=[],trees:Item[]=[];
  const box=(x:number,y:number,z:number,w:number,h:number,d:number,color:string)=>boxes.push({p:[x,y,z],s:[w,h,d],color});
  const tree=(x:number,z:number,k:number)=>{box(x,1.6,z,.35,3,.35,'#71604c');trees.push({p:[x,4.2,z],s:[2.1,2.8,2.1],color:['#6a8058','#8c9d67','#55725a'][k%3]});};
  box(0,-1,0,map.width+4,2,map.depth+4,'#aeb197');
  if(map.id==='ribera'){
   const left=map.columns[2]+6.5,right=map.columns[3]-6.5;
   box((left+right)/2,.025,0,right-left,.05,map.depth,'#659ba2');
   box(left,.18,0,.8,.36,map.depth,'#c9c7af');box(right,.18,0,.8,.36,map.depth,'#c9c7af');
  }
  for(const r of map.roads){const horizontal=r.a.z===r.b.z,length=Math.hypot(r.b.x-r.a.x,r.b.z-r.a.z),cx=(r.a.x+r.b.x)/2,cz=(r.a.z+r.b.z)/2;
   const roadBox=(offset:number,lateral:number,w:number,d:number,color:string,y=.195)=>box(cx+(horizontal?offset:lateral),y,cz+(horizontal?lateral:offset),horizontal?w:d,.03,horizontal?d:w,color);
   box(cx,.08,cz,horizontal?length:12,.08,horizontal?12:length,r.avenue?'#414b49':'#505852');
   for(let k=-length/2+10;k<length/2-8;k+=6){roadBox(k,-.23,3,.12,'#dbc17b');roadBox(k,.23,3,.12,'#dbc17b');for(const side of [-4.8,4.8])roadBox(k,side,2,.12,'#dfdfc9');}
   if(r.avenue&&length>36){roadBox(0,0,length-24,.75,'#bcbba4',.29);roadBox(0,0,length-26,.45,'#798b59',.35);}
   if(map.id==='ribera'&&horizontal&&r.a.x===map.columns[2]&&r.b.x===map.columns[3])for(const side of [-6,6]){box(cx,1,cz+side,length-14,.18,.22,'#abb7ad');for(let x=r.a.x+8;x<r.b.x-6;x+=7)box(x,.6,cz+side,.2,1.2,.2,'#7c9186');}
  }
  for(const n of map.nodes){box(n.x,.145,n.z,12,.07,12,'#4b5350');for(const road of map.roads.filter(r=>r.from===n.id||r.to===n.id)){const o=road.from===n.id?road.b:road.a,dx=Math.sign(o.x-n.x),dz=Math.sign(o.z-n.z);for(const k of [-4,-2.6,-1.2,.2,1.6,3,4.4])box(n.x+dx*8+(dz?k:0),.215,n.z+dz*8+(dx?k:0),dx?2:.65,.025,dz?2:.65,'#e4dfc9');}}
  const palette=['#c4a684','#b8785e','#a5b2ab','#c9c5ad','#97a4a0','#b5aa93'];
  map.lots.forEach((l,index)=>{const {x,z,width:w,depth:d,kind}=l;if(kind==='water')return;
   box(x,.27,z,w,.5,d,'#c5c6b1');
   if(kind==='park'){box(x,.54,z,w-2,.04,d-2,'#91a574');box(x,.59,z,3,.05,d-2,'#d9cbaa');box(x,.6,z,w-2,.05,3,'#d9cbaa');for(const a of [-.32,.32])for(const b of [-.32,.32])tree(x+a*w,z+b*d,index);if(index%2===0){box(x,.8,z,7,.4,7,'#cdd4cb');box(x,1.02,z,5,.04,5,'#78a9aa');}return;}
   if(kind==='hospital'||kind==='depot'||kind==='industrial'){const h=kind==='hospital'?12:6;walls.push({p:[x,h/2+.5,z],s:[w-6,h,d-6],color:kind==='hospital'?'#e0ddc9':'#ac8c6d'});box(x,h+.65,z,w-5,.4,d-5,'#93998a');if(kind==='hospital'){box(x,h+.9,z,3,.1,12,'#d57253');box(x,h+1,z,12,.1,3,'#d57253');}else for(let k=-1;k<=1;k++)box(x+k*5,h+1.2,z,2,.7,d-10,'#677a76');}
   else for(let k=0;k<3;k++){const bw=w/2-5,bd=k===0?d-8:d/2-6,bx=x+(k===0?-w/4:w/4),bz=z+(k===0?0:k===1?-d/4:d/4);const h=kind==='towers'?22+(index*7+k*11)%33:5+(index*3+k*5)%13;
    walls.push({p:[bx,h/2+.5,bz],s:[bw,h,bd],color:palette[(index+k)%palette.length]});box(bx,h+.8,bz,bw+.5,.55,bd+.5,'#c7c7b6');box(bx,h+1.1,bz,bw-1,.1,bd-1,'#79877d');box(bx+2,h+1.7,bz,3,1.2,2.3,'#85938b');if(h<15){box(bx,1.9,bz+bd/2+.1,bw-1,2.8,.2,'#334d46');box(bx,3.5,bz+bd/2+.7,bw,.2,1.5,index%2?'#ce895b':'#628975');}}
   tree(x-w/2+2,z+d/2-2,index);tree(x+w/2-2,z+d/2-2,index+1);
  });
  return {boxes,walls,trees};
 },[map.id]);
 return <group><Instances items={data.boxes}/><Instances items={data.walls} facade/><Instances items={data.trees} foliage/>{map.nodes.filter((_,i)=>i%map.columns.length===1).map((n,i)=><StreetSign key={n.id} text={i===1?'AV. CENTRAL':`DISTRITO ${String(i+1).padStart(2,'0')}`} at={[n.x+7,4,n.z+7]} width={5} height={.8}/>)}<mesh rotation={[-Math.PI/2,0,0]} position={[0,-2.05,0]} receiveShadow><planeGeometry args={[Math.max(1500,map.width*4),Math.max(1500,map.depth*4)]}/><meshStandardMaterial color="#aab09b" roughness={1}/></mesh></group>;
},(a,b)=>a.map.id===b.map.id);
