import test from 'node:test';
import assert from 'node:assert/strict';
import { appendUniqueBatch, appendUnique, clock, number } from '../lib/format';
test('unavailable metrics never masquerade as zero', () => { assert.equal(number(undefined), '—'); assert.equal(number(NaN), '—'); assert.equal(number(0), '0'); });
test('simulation clock has consistent rollover', () => { assert.equal(clock(3661.9), '01:01:01'); assert.equal(clock(0), '00:00:00'); });
test('replayed event IDs are deduplicated and history stays bounded', () => { const events = [{ id: 'a' }, { id: 'b' }]; assert.deepEqual(appendUnique(events, { id: 'a' }, 2), events); assert.deepEqual(appendUnique(events, { id: 'c' }, 2), [{ id: 'b' }, { id: 'c' }]); });

test('city event batches deduplicate within a batch and retain the newest bounded history',()=>{const initial=[{id:'a'},{id:'b'}];assert.deepEqual(appendUniqueBatch(initial,[{id:'b'},{id:'c'},{id:'c'},{id:'d'}],3),[{id:'b'},{id:'c'},{id:'d'}]);assert.equal(appendUniqueBatch(initial,[{id:'a'}],3),initial);});
