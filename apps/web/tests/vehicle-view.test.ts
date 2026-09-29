import test from 'node:test';
import assert from 'node:assert/strict';
import { heading, chooseVehicle, nextSignal } from '../lib/vehicle-view';
import type { Vehicle, World } from '../lib/types';
const vehicle: Vehicle = {id:'veh_1_0042', type:'CAR', direction:'E', road:1, position:-50,x:-50,z:2,speed:0,state:'WAITING',route:['INT-04','INT-05','INT-06'],next:0,wait:4,born:0};
test('heading follows the server coordinate system',()=>{assert.deepEqual(heading('N'),[0,-1]);assert.deepEqual(heading('E'),[1,0]);assert.deepEqual(heading('S'),[0,1]);assert.deepEqual(heading('W'),[-1,0]);});
test('boarding picks an available vehicle with a long remaining route',()=>{assert.equal(chooseVehicle([]),undefined);assert.equal(chooseVehicle([vehicle,{...vehicle,id:'new',position:-65}])?.id,'new');});
test('onboard signal display never treats a conflicting green as right of way',()=>{const world={intersections:[{id:'INT-04',phase:'NS_GREEN'}]} as World;assert.equal(nextSignal(vehicle,world).color,'red');assert.equal(nextSignal(vehicle,world).distance,3);world.intersections[0].phase='EW_GREEN';assert.equal(nextSignal(vehicle,world).color,'green');world.intersections[0].phase='ALL_RED_EW_TO_NS';assert.equal(nextSignal(vehicle,world).color,'red');});
test('finished route has an explicit destination state',()=>{assert.equal(nextSignal({...vehicle,next:3},{intersections:[]} as unknown as World).label,'Leaving the district');});
test('onboard distances use the selected map route, including turns',()=>{const world={intersections:[{id:'INT-05',phase:'EW_GREEN'}]} as World;const v={...vehicle,next:1,position:4,stops:[-47,18,81]};assert.equal(nextSignal(v,world).distance,14);assert.equal(nextSignal({...v,position:20},world).label,'Clearing the junction');});

test('green signal explains downstream traffic instead of promising movement',()=>{const world={intersections:[{id:'INT-04',phase:'EW_GREEN'}]} as World;for(const [waitReason,label] of [['QUEUE','Green · queue ahead'],['JUNCTION_BUSY','Green · junction occupied'],['EXIT_BLOCKED','Green · exit blocked']] as const){assert.equal(nextSignal({...vehicle,waitReason},world).label,label);assert.equal(nextSignal({...vehicle,waitReason,speed:2},world).label,'Right of way');}world.intersections[0].phase='NS_GREEN';assert.equal(nextSignal({...vehicle,waitReason:'QUEUE'},world).label,'Stop at the signal');});
