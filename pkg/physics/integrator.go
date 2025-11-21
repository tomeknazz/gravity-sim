package physics

// IntegrateEulerSymplectic wykonuje symulację metodą semi-implicit Euler
func IntegrateEulerSymplectic(bodies []Body, dt float64) []Body {
	// Aktualizacja dla każdego ciała
	for i := range bodies {
		// Oblicz przyspieszenie na podstawie aktualnych pozycji wszystkich ciał
		bodies[i].Acc = ComputeAcceleration(bodies[i], bodies)

		if bodies[i].Locked {
			// nie aktualizujemy prędkości i pozycji zablokowanego ciała
			bodies[i].Vel = Vec2{0, 0}
			continue
		}

		// Semi-implicit Euler: najpierw aktualizujemy prędkość
		bodies[i].Vel = bodies[i].Vel.Add(bodies[i].Acc.Mul(dt))

		// Następnie aktualizujemy pozycję według nowej prędkości
		bodies[i].Pos = bodies[i].Pos.Add(bodies[i].Vel.Mul(dt))
	}
	return bodies
}
