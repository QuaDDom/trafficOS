'use client';
import { ArrowUpRight, ChevronRight, Crosshair, Radio, TrafficCone } from 'lucide-react';
import { number } from '@/lib/format';
import type { Decision, Intersection, StreetMap } from '@/lib/types';
const modes:Record<string,string>={ADAPTIVE:'Adaptativo',FIXED:'Ciclo fijo',SAFE_MODE:'Ciclo de respaldo',EMERGENCY_OVERRIDE:'Paso de emergencia'};
const actions:Record<string,string>={EXTEND_PHASE:'Mantener este verde',SHORTEN_PHASE:'Dar paso al otro eje',HOLD:'Mantener la fase',NEXT_PHASE:'Preparar el siguiente eje'};
export function Inspector({intersection:i,map,onSelect,onFocus,onDecision,onIncident,onEvents,busy}:{intersection?:Intersection;map?:StreetMap;onSelect:(id:string)=>void;onFocus:()=>void;onDecision:(d:Decision)=>void;onIncident:()=>void;onEvents:()=>void;busy:boolean}) {
 if(!i) return <aside className="inspector"><h2>Qué pasa en el cruce</h2><p>Conectando con los semáforos…</p></aside>;
 const green=i.phase.endsWith('GREEN'),yellow=i.phase.endsWith('YELLOW');
 const axis=i.phase.startsWith('NS_')?'Norte ↔ Sur':'Este ↔ Oeste';
 const approaching=i.lanes.reduce((sum,l)=>sum+l.vehicleCount,0);
 const average=approaching?i.lanes.reduce((sum,l)=>sum+l.averageWait*l.vehicleCount,0)/approaching:0;
 const d=i.controllerMode==='ADAPTIVE'?i.lastDecision:undefined;
 return <aside className="inspector">
  <div className="section-kicker"><span>02 / LEÉ LA CALLE</span><Radio size={16}/></div>
  <div className="intersection-title"><h2>Qué pasa en el cruce</h2><span className={'status-tag '+(i.status==='ONLINE'?'green':'red')}>{i.status==='ONLINE'?'En línea':'Respaldo'}</span></div>
  <div className="intersection-picker"><select aria-label="Cruce para explorar" value={i.id} onChange={e=>onSelect(e.target.value)}>{map?.nodes.map(n=><option key={n.id} value={n.id}>{n.id} · {n.name}</option>)}</select><button onClick={onFocus} title="Acercar la cámara a este cruce" aria-label="Centrar cruce"><Crosshair size={19}/></button></div>
  <div className="phase-card"><div className="traffic-lamp" aria-hidden="true"><i className={!green&&!yellow?'red-light':''}/><i className={yellow?'amber-light':''}/><i className={green?'green-light':''}/></div><div><span className="section-kicker">{green?'TIENEN PASO':yellow?'TERMINANDO EL PASO':'CRUCE EN DESPEJE'}</span><strong>{green||yellow?axis:'Todos en rojo'}</strong><small>{number(i.phaseAge,1)} s <span>/ {green?'máximo 45 s':yellow?'3 s de amarillo':'2 s de seguridad'}</span></small></div></div>
  <p className="phase-explanation">{green?'El otro eje espera. Los vehículos avanzan si el cruce y su salida están libres.':yellow?'Los vehículos que ya entraron terminan de cruzar.':'Se despeja el cruce antes de habilitar el siguiente eje.'}</p>
  <div className="queue-heading"><span className="section-kicker">VEHÍCULOS EN ESPERA</span><span>personas a bordo</span></div>
  <div className="lanes">{i.lanes.map(l=><div className="lane" key={l.id}><span className="lane-direction">{({N:'Norte',S:'Sur',E:'Este',W:'Oeste'} as Record<string,string>)[l.direction]}</span><div className="lane-track"><span style={{width:Math.min(100,l.queueLength/l.capacity*100)+'%'}} className={l.queueLength>8?'congested':''}/></div><strong title={(l.people??0)+' personas detectadas; '+(l.busCount??0)+' buses'}>{l.queueLength}<small className="lane-people">{l.queuedPeople??0} pers.</small></strong></div>)}</div>
  <div className="small-metrics"><div><span>Espera en estos accesos</span><strong>{number(average,1)}<small> s</small></strong></div><div><span>A menos de 40 m</span><strong>{approaching}<small> veh.</small></strong></div></div>
  {d&&<button className="decision-preview" onClick={()=>onDecision(d)}><div><span>POR QUÉ CAMBIA EL SEMÁFORO</span><ChevronRight size={16}/></div><strong>{actions[d.action]??'Respetar el ciclo seguro'}</strong><small>{d.reason.type==='MINIMUM_GREEN'?'Se está cumpliendo el verde mínimo.':d.reason.type==='MAXIMUM_GREEN'?'Se alcanzó el tiempo máximo de verde.':d.reason.type==='BALANCED_DEMAND'?'Los dos ejes tienen una demanda similar.':d.reason.type==='BALANCED_CYCLE'?'Con demanda equilibrada, se alterna el paso.':d.action==='EXTEND_PHASE'?'El eje actual necesita más tiempo de paso.':d.action==='SHORTEN_PHASE'?'La demanda del otro eje justifica el cambio.':'Se mantienen los tiempos de seguridad.'} Ver detalle ↗</small></button>}
  <div className="inspector-actions"><button onClick={onEvents}>Historial del cruce <ArrowUpRight size={15}/></button><button disabled={busy} onClick={onIncident} aria-label="Crear obra en este cruce" title="Crear una obra para reducir la velocidad"><TrafficCone size={18}/></button></div>
  <div className="inspector-foot"><span>Control: {modes[i.controllerMode]??i.controllerMode}</span><span>v{i.version}</span></div>
 </aside>;
}
