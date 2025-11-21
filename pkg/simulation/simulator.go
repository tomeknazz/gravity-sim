package simulation

import (
	"gravity-sim/pkg/physics"
)

// --- Główna struktura symulatora ---
type Simulator struct {
	Name   string
	Dt     float64
	Bodies []physics.Body
}

// --- Tworzenie symulatora z konfiguracji ---
func NewSimulator(cfg EnvironmentConfig) *Simulator {
	bodies := make([]physics.Body, len(cfg.Bodies))

	for i, b := range cfg.Bodies {
		bodies[i] = physics.Body{
			Mass:   b.Mass,
			Pos:    physics.Vec2{X: b.Pos[0], Y: b.Pos[1]},
			Vel:    physics.Vec2{X: b.Vel[0], Y: b.Vel[1]},
			Radius: b.Radius,
			ColorC: parseColor(b.Color), // parseColor zwraca teraz color.RGBA
		}
	}

	return &Simulator{
		Name:   cfg.Name,
		Dt:     cfg.Dt,
		Bodies: bodies,
	}
}

// --- Aktualizacja symulacji ---
func (s *Simulator) Update() {
	s.Bodies = physics.IntegrateEulerSymplectic(s.Bodies, s.Dt)
}
