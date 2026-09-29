import * as THREE from 'three';
import { mergeGeometries } from 'three/examples/jsm/utils/BufferGeometryUtils.js';

export const vehicleStyles = ['sedan', 'hatch', 'suv', 'ambulance', 'bus'] as const;
export type VehicleStyle = typeof vehicleStyles[number];
export const partNames = ['paint', 'glass', 'trim', 'metal', 'head', 'tail', 'accent'] as const;
export type VehiclePart = typeof partNames[number];
type XYZ = [number, number, number];

function box(size: XYZ, at: XYZ, rotation: XYZ = [0, 0, 0]) {
 const geometry = new THREE.BoxGeometry(...size);
 geometry.rotateX(rotation[0]).rotateY(rotation[1]).rotateZ(rotation[2]);
 return geometry.translate(...at);
}
function beam(a: XYZ, b: XYZ, width: number) {
 const start = new THREE.Vector3(...a), end = new THREE.Vector3(...b);
 const geometry = new THREE.BoxGeometry(width, start.distanceTo(end), width);
 geometry.applyQuaternion(new THREE.Quaternion().setFromUnitVectors(new THREE.Vector3(0,1,0), end.clone().sub(start).normalize()));
 return geometry.translate(...start.add(end).multiplyScalar(.5).toArray());
}
function join(parts: THREE.BufferGeometry[]) {
 const normalized = parts.map(g => { const n=g.index?g.toNonIndexed():g.clone(); n.deleteAttribute('uv'); return n; });
 const geometry = mergeGeometries(normalized)!;
 for (const g of [...parts,...normalized]) g.dispose();
 geometry.computeBoundingSphere();
 return geometry;
}
function shell(profile: [number,number][], width: number, bevel = .035) {
 const shape = new THREE.Shape();
 profile.forEach(([x,y],i)=>i===0?shape.moveTo(x,y):shape.lineTo(x,y)); shape.closePath();
 const geometry = new THREE.ExtrudeGeometry(shape,{depth:width-2*bevel,bevelEnabled:bevel>0,bevelSegments:2,steps:1,bevelSize:bevel,bevelThickness:bevel});
 return geometry.translate(0,0,-width/2+bevel);
}
function body(length:number,width:number,height:number,axle=.98,radius=.37) {
 const half=length/2, shape=new THREE.Shape();
 shape.moveTo(-half,.39); shape.lineTo(-half,.65); shape.lineTo(-half+.16,height-.04);
 shape.lineTo(-.9,height); shape.lineTo(.7,height); shape.lineTo(half-.15,height-.12);
 shape.lineTo(half,.62); shape.lineTo(half,.39); shape.lineTo(axle+radius,.39);
 shape.absarc(axle,.39,radius,0,Math.PI,false); shape.lineTo(-axle+radius,.39);
 shape.absarc(-axle,.39,radius,0,Math.PI,false); shape.closePath();
 return new THREE.ExtrudeGeometry(shape,{depth:width-.08,bevelEnabled:true,bevelSegments:2,steps:1,bevelSize:.04,bevelThickness:.04,curveSegments:8}).translate(0,0,-width/2+.04);
}

/** Shared, baked geometry: detail does not add scene nodes for each car. Front is +X. */
export function createVehicleModel(style: VehicleStyle): Record<VehiclePart,THREE.BufferGeometry> {
 if(style==='bus')return createBusModel();
 const ambulance=style==='ambulance', suv=style==='suv', hatch=style==='hatch';
 const length=ambulance?3.5:suv?3.26:3.16, width=ambulance?1.58:suv?1.56:1.48;
 const belt=suv?.96:.86, roof=ambulance?1.94:suv?1.65:hatch?1.49:1.4;
 const back=ambulance?-1.54:hatch?-1.18:suv?-1.12:-.98;
 const roofBack=ambulance?-1.48:hatch?-.99:suv?-.9:-.62;
 const roofFront=ambulance?.65:suv?.27:.12, front=ambulance?1.24:.72;
 const parts:Record<VehiclePart,THREE.BufferGeometry[]>={paint:[],glass:[],trim:[],metal:[],head:[],tail:[],accent:[]};
 parts.paint.push(body(length,width,belt));
 parts.glass.push(shell([[back,belt],[roofBack,roof-.08],[roofFront,roof-.08],[front,belt]],width-.2,.015));
 parts.paint.push(shell([[roofBack-.025,roof-.09],[roofBack+.02,roof],[roofFront-.02,roof],[roofFront+.035,roof-.09]],width-.22,.025));
 for(const side of [-1,1]) {
  const z=side*(width/2-.085);
  // A/C pillars, window sill, B-pillar, door seams and handles.
  parts.paint.push(beam([back,belt,z],[roofBack,roof-.07,z],.065),beam([front,belt,z],[roofFront,roof-.07,z],.06));
  parts.trim.push(beam([-.28,belt,z*1.015],[-.28,roof-.08,z*1.015],.055),box([front-back,.045,.035],[(front+back)/2,belt,z]));
  parts.trim.push(box([.012,.29,.008],[-.27,belt-.16,side*(width/2+.041)]));
  for(const x of [-.56,.35]) parts.metal.push(box([.14,.028,.026],[x,belt-.09,side*(width/2+.045)]));
  parts.trim.push(beam([.47,belt+.07,z],[.46,belt+.11,side*(width/2+.16)],.04));
  parts.paint.push(box([.19,.12,.14],[.46,belt+.12,side*(width/2+.18)]));
  parts.metal.push(box([.012,.075,.1],[.365,belt+.12,side*(width/2+.18)]));
  parts.trim.push(box([1.24,.065,.05],[0,.39,side*(width/2+.02)]));
  parts.head.push(box([.065,.12,.36],[length/2-.025,.66,side*.49]));
  parts.tail.push(box([.045,.1,.3],[-length/2-.025,.65,side*.51]));
  parts.tail.push(box([.045,.19,.055],[-length/2-.026,.72,side*.64]));
  parts.accent.push(box([.025,.07,.085],[length/2+.008,.65,side*.69]));
 }
 parts.trim.push(box([.075,.12,width-.23],[length/2-.01,.43,0]),box([.055,.11,width-.12],[-length/2-.015,.43,0]));
 parts.trim.push(box([.035,.16,.59],[length/2+.006,.62,0]));
 for(const y of [.57,.62,.67])parts.metal.push(box([.04,.014,.51],[length/2+.028,y,0]));
 for(const x of [-length/2-.048,length/2+.055])parts.metal.push(box([.012,.1,.25],[x,.47,0]));
 // Wipers are on the sloping windshield, not floating in front of it.
 for(const side of [-1,1])parts.trim.push(beam([front-.035,belt+.025,side*.49],[front-.095,belt+.09,side*.14],.017));
 if(suv){for(const side of [-1,1])parts.trim.push(box([1.03,.045,.055],[-.28,roof+.07,side*.5]));}
 if(hatch)parts.paint.push(box([.16,.055,width-.16],[roofBack-.06,roof-.04,0]));
 if(ambulance){
  parts.paint.push(shell([[-1.56,.8],[-1.56,1.94],[.36,1.94],[.4,.8]],1.5,.035));
  for(const side of [-1,1]){
   const z=side*.789;
   parts.accent.push(box([1.84,.15,.02],[-.59,1.03,z]),box([.36,.13,.02],[-.54,1.54,z]),box([.13,.36,.02],[-.54,1.54,z]));
   parts.glass.push(box([.47,.27,.025],[-1.1,1.61,z]));
   parts.trim.push(box([.012,.84,.016],[-.09,1.35,z]));
  }
  parts.glass.push(box([.022,.34,1.02],[-1.605,1.6,0]));
  parts.trim.push(box([.03,1.12,.025],[-1.627,1.3,0]),box([.3,.055,1.12],[.53,roof+.045,0]));
 }else{
  parts.tail.push(box([.02,.045,.34],[back-.02,belt+.07,0]));
 }
 return Object.fromEntries(partNames.map(name=>[name,join(parts[name])])) as Record<VehiclePart,THREE.BufferGeometry>;
}
function createBusModel():Record<VehiclePart,THREE.BufferGeometry>{
 const parts:Record<VehiclePart,THREE.BufferGeometry[]>={paint:[],glass:[],trim:[],metal:[],head:[],tail:[],accent:[]};
 parts.paint.push(body(6.4,1.9,.95,2.05,.46),shell([[-3.14,.94],[-3.14,2.52],[2.88,2.52],[3.14,2.25],[3.14,.94]],1.88,.035));
 for(const side of [-1,1]){
  const z=side*.969;
  parts.glass.push(box([5.78,.91,.018],[-.06,1.88,z]));
  parts.accent.push(box([6.18,.22,.026],[0,1.16,z]),box([6.14,.09,.026],[0,2.4,z]));
  for(const x of [-2.95,-1.96,-.98,0,.98,1.94,2.8])parts.paint.push(box([.085,1.07,.031],[x,1.9,z]));
  if(side===1){for(const x of [-.65,2.2]){
   parts.trim.push(box([.69,1.67,.035],[x,1.22,z+.02]));
   parts.glass.push(box([.57,1.49,.04],[x,1.24,z+.04]));
   parts.metal.push(box([.025,1.5,.046],[x,1.24,z+.065]));
  }}
  parts.head.push(box([.055,.18,.33],[3.2,.8,side*.63]));
  parts.tail.push(box([.045,.34,.11],[-3.2,.86,side*.74]));
  parts.trim.push(beam([2.9,2.2,side*.97],[3.2,2.17,side*1.14],.05),box([.15,.32,.13],[3.19,2.05,side*1.18]));
 }
 parts.glass.push(box([.032,1.07,1.61],[3.179,1.71,0]),box([.032,.78,1.45],[-3.179,1.97,0]));
 parts.trim.push(box([.05,.27,1.5],[3.2,2.36,0]),box([.1,.15,1.75],[3.2,.48,0]),box([.1,.15,1.75],[-3.2,.48,0]));
 // Amber destination display and roof-mounted air conditioning identify the bus at both scales.
 parts.accent.push(box([.065,.08,.67],[3.225,2.36,0]));
 parts.metal.push(box([1.5,.18,1.14],[-.2,2.67,0]),box([.03,.11,.32],[-3.265,.54,0]));
 for(const x of [-.8,-.5,-.2,.1,.4])parts.trim.push(box([.035,.025,.95],[x,2.77,0]));
 return Object.fromEntries(partNames.map(name=>[name,join(parts[name])])) as Record<VehiclePart,THREE.BufferGeometry>;
}
export function createWheelGeometry() {
 const tire = new THREE.TorusGeometry(.253,.072,6,16);
 const rimParts:THREE.BufferGeometry[]=[];
 for(const side of [-1,1]){
  const ring=new THREE.TorusGeometry(.193,.018,4,16);ring.translate(0,0,side*.065);rimParts.push(ring);
  const hub=new THREE.CylinderGeometry(.065,.065,.012,10);hub.rotateX(Math.PI/2).translate(0,0,side*.072);rimParts.push(hub);
  for(let i=0;i<5;i++){const spoke=box([.17,.027,.018],[.11,0,side*.07]);spoke.rotateZ(i*Math.PI*2/5);rimParts.push(spoke);}
 }
 return {tire,rim:join(rimParts)};
}
