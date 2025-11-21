package physics

const G = 6.67430e-1 // sztucznie zwiększone dla wizualizacji

func ComputeAcceleration(b1 Body, others []Body) Vec2 {
	force := Vec2{0, 0}
	epsilon := 5.0 // parametr softeningu, dostosuj do skali układu

	for _, b2 := range others {
		if &b1 == &b2 {
			continue
		}

		dir := b2.Pos.Sub(b1.Pos)
		dist2 := dir.Len()*dir.Len() + epsilon*epsilon // softening
		f := G * b1.Mass * b2.Mass / dist2
		force = force.Add(dir.Normalize().Mul(f / b1.Mass))
	}

	return force
}
