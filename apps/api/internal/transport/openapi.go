package transport

import "trafficos/packages/network"

// OpenAPI is generated from the small, explicitly supported resource surface.
func OpenAPI() map[string]any {
	paths := map[string]any{}
	for _, path := range []string{"/health/live", "/health/ready", "/api/v1/health", "/api/v1/snapshot", "/api/v1/maps", "/api/v1/intersections", "/api/v1/events", "/api/v1/decisions", "/api/v1/requests", "/api/v1/traffic/metrics", "/api/v1/simulation/config"} {
		paths[path] = map[string]any{"get": map[string]any{"summary": path, "responses": map[string]any{"200": map[string]string{"description": "JSON resource"}}}}
	}
	schemas := map[string]any{
		"simulation/map":     map[string]any{"mapId": map[string]any{"type": "string", "enum": network.IDs()}},
		"simulation/profile": map[string]any{"profile": map[string]any{"type": "string", "enum": []string{"LOW", "NORMAL", "RUSH_HOUR", "GRIDLOCK"}}},
		"simulation/control": map[string]any{"priorityMode": map[string]any{"type": "string", "enum": []string{"VEHICLES", "PEOPLE"}}, "mode": map[string]any{"type": "string", "enum": []string{"FIXED", "ADAPTIVE"}}, "speed": map[string]any{"type": "number", "enum": []float64{0.5, 1, 2, 5, 10}}, "paused": map[string]string{"type": "boolean"}},
		"vehicles":           map[string]any{"type": map[string]any{"type": "string", "enum": []string{"CAR", "SUV", "BUS"}}, "occupants": map[string]any{"type": "integer", "minimum": 1, "maximum": 60, "description": "CAR 1–4, SUV 1–6, BUS 1–60; includes everyone on board"}, "road": map[string]any{"type": "integer", "minimum": 0, "description": "Zero-based entry index; bounded by the selected map rows (E/W) or columns (N/S)"}, "direction": map[string]any{"type": "string", "enum": []string{"N", "S", "E", "W"}}},
		"simulation/reset":   map[string]any{}, "emergency-vehicles": map[string]any{"priority": map[string]any{"type": "integer", "minimum": 1, "maximum": 3}},
		"incidents":         map[string]any{"intersectionId": map[string]string{"type": "string"}, "type": map[string]any{"type": "string", "enum": []string{"ROADWORK", "COLLISION"}}},
		"incidents/resolve": map[string]any{"intersectionId": map[string]string{"type": "string", "description": "Incident ID to resolve"}},
		"signals/commands":  map[string]any{"intersectionId": map[string]string{"type": "string"}, "phase": map[string]string{"type": "string"}, "version": map[string]string{"type": "integer"}},
		"chaos":             map[string]any{"type": map[string]any{"type": "string", "enum": []string{"optimizer", "intersection", "latency"}}, "online": map[string]string{"type": "boolean"}, "intersectionId": map[string]string{"type": "string"}, "latencyMs": map[string]any{"type": "integer", "minimum": 0, "maximum": 1000}},
		"comparisons":       map[string]any{"priorityMode": map[string]any{"type": "string", "enum": []string{"VEHICLES", "PEOPLE"}}, "mapId": map[string]any{"type": "string", "enum": network.IDs()}, "seed": map[string]any{"type": "integer", "minimum": 1}, "duration": map[string]any{"type": "number", "minimum": 30, "maximum": 600}, "profile": map[string]string{"type": "string"}},
		"telemetry":         map[string]any{"epoch": map[string]string{"type": "string", "description": "Epoch from config; stale frames are acknowledged without advancing the new map"}, "session": map[string]string{"type": "string"}, "sequence": map[string]string{"type": "integer"}, "frames": map[string]any{"type": "array", "items": map[string]any{"type": "object", "description": "A dt (0 < dt <= 0.21) and arrivals array of {road: 0..rows-1 for E/W, 0..columns-1 for N/S, direction: N/S/E/W, type: CAR/SUV/BUS (optional), occupants: 1–4/6/60 (0 or omitted uses default)}"}}},
	}
	for path, properties := range schemas {
		paths["/api/v1/"+path] = map[string]any{"post": map[string]any{"summary": path, "parameters": []any{map[string]any{"in": "header", "name": "Idempotency-Key", "required": path == "emergency-vehicles", "schema": map[string]string{"type": "string"}}}, "requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "properties": properties}}}}, "responses": map[string]any{"200": map[string]string{"description": "Command applied"}, "201": map[string]string{"description": "Resource created"}, "400": map[string]string{"description": "Validation error"}, "409": map[string]string{"description": "Conflict"}, "503": map[string]string{"description": "Dependency unavailable"}}}}
	}
	for _, path := range []string{"/api/v1/intersections/{id}", "/api/v1/intersections/{id}/state", "/api/v1/decisions/{id}", "/api/v1/database/{table}"} {
		name := "id"
		if path == "/api/v1/database/{table}" {
			name = "table"
		}
		paths[path] = map[string]any{"get": map[string]any{"parameters": []any{map[string]any{"in": "path", "name": name, "required": true, "schema": map[string]string{"type": "string"}}}, "responses": map[string]any{"200": map[string]string{"description": "JSON resource"}, "404": map[string]string{"description": "Not found"}}}}
	}
	return map[string]any{"openapi": "3.0.3", "info": map[string]string{"title": "TrafficOS API", "version": "1.0.0", "description": "Backend-authoritative traffic simulation. See /ws for snapshot and event messages. Telemetry requires a simulator bearer token."}, "paths": paths}
}
