package simulation

import (
	"encoding/json"
	"fmt"
	"math"
	"os"

	"image/color"
)

// --- Struktura konfiguracji środowiska ---
type EnvironmentConfig struct {
	Name      string       `json:"name"`
	Dt        float64      `json:"dt"`
	Bodies    []BodyConfig `json:"bodies"`
	AutoOrbit bool         `json:"auto_orbit,omitempty"`
}

type BodyConfig struct {
	Mass   float64    `json:"mass"`
	Pos    [2]float64 `json:"pos"`
	Vel    [2]float64 `json:"vel"`
	Color  string     `json:"color"`
	Radius float64
}

func SetOrbitalVelocities(bodies []BodyConfig) {
	if len(bodies) == 0 {
		return
	}
	central := bodies[0] // pierwsze ciało traktujemy jako centralne
	G := 6.67430e-1
	for i := 1; i < len(bodies); i++ {
		b := (bodies[i].Vel[0] == 0) && bodies[i].Vel[1] == 0
		if !b {
			continue
		}

		dx := bodies[i].Pos[0] - central.Pos[0]
		dy := bodies[i].Pos[1] - central.Pos[1]
		r := math.Hypot(dx, dy)
		v := math.Sqrt(G * central.Mass / r)
		// skierowanie prędkości prostopadle do wektora pozycji
		bodies[i].Vel[0] = -dy / r * v
		bodies[i].Vel[1] = dx / r * v
	}
}

// --- Wczytanie pliku konfiguracyjnego ---
func LoadConfig(path string) (*Simulator, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("błąd odczytu pliku: %v", err)
	}

	var env EnvironmentConfig
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("błąd parsowania JSON: %v", err)
	}

	if env.AutoOrbit {
		SetOrbitalVelocities(env.Bodies)
	}

	sim := NewSimulator(env)
	return sim, nil
}

// --- Parser koloru HEX ---
func parseColor(hex string) color.RGBA {
	var r, g, b uint8
	if len(hex) == 7 && hex[0] == '#' {
		n, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
		if err == nil && n == 3 {
			return color.RGBA{r, g, b, 255}
		}
	}
	return color.RGBA{200, 200, 255, 255}
}
