'use client';
import { useEffect, useState } from 'react';
import { ArrowUpRight, Map as MapIcon } from 'lucide-react';
import { API } from '@/lib/api';
import type { StreetMap } from '@/lib/types';

export function MapSelector({active,busy,onChange}:{active?:StreetMap;busy:boolean;onChange:(id:string)=>void}){
 const [maps,setMaps]=useState<StreetMap[]>([]),[error,setError]=useState(false),[attempt,setAttempt]=useState(0);
 useEffect(()=>{const abort=new AbortController();fetch(`${API}/api/v1/maps`,{signal:abort.signal}).then(async r=>{if(!r.ok)throw new Error('maps');setMaps(await r.json() as StreetMap[]);setError(false);}).catch(()=>{if(!abort.signal.aborted)setError(true)});return()=>abort.abort();},[active?.id,attempt]);
 return <section className="district-picker" aria-label="Elegir mapa de la ciudad">
  <div className="district-picker-heading"><span><MapIcon size={13}/> CADA BARRIO, UN DESAFÍO</span><small>Cambiar de mapa reinicia el tráfico.</small></div>
  <div className="district-options">{maps.map((map,index)=><button key={map.id} className={`district-option ${map.id===active?.id?'selected':''} ${index>=3?'city-scale':''}`} aria-pressed={map.id===active?.id} disabled={busy||!active||map.id===active.id} onClick={()=>onChange(map.id)}>
   <svg viewBox={`${-map.width/2-4} ${-map.depth/2-4} ${map.width+8} ${map.depth+8}`} aria-hidden="true">{map.lots.filter(l=>l.kind==='park').map((l,k)=><rect key={k} x={l.x-l.width/2} y={l.z-l.depth/2} width={l.width} height={l.depth} className="mini-park"/>)}{map.roads.map(r=><line key={r.id} x1={r.a.x} y1={r.a.z} x2={r.b.x} y2={r.b.z} strokeWidth={r.avenue?5:2.5}/>)}{map.nodes.map(n=><circle key={n.id} cx={n.x} cy={n.z} r={2.8}/>)}</svg>
   <span className="district-copy"><small>0{index+1} / {map.id===active?.id?'MAPA ACTUAL':map.nodes.length>16?'CIUDAD':'DISTRITO'}</small><strong>{map.name}</strong><span>{map.nodes.length} cruces · {Math.round(map.width)} × {Math.round(map.depth)} m</span></span><ArrowUpRight className="district-arrow" size={18}/>
  </button>)}</div>
  {!maps.length&&!error&&<p className="catalog-loading">Cargando los seis mapas…</p>}{error&&<button className="text-button" onClick={()=>setAttempt(v=>v+1)}>No se pudo cargar el catálogo · Reintentar</button>}
 </section>;
}
