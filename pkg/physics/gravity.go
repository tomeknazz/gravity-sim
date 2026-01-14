package physics

const G = 6.67430e-1 // stala grawitacji

func ComputeAcceleration(b1 Body, others []Body) Vec2 {
	force := Vec2{0, 0}
	epsilon := 5.0 // parametr softeningu, dostosuj do skali układu

	for _, b2 := range others {
		// porównujemy adresy przez wartości — jeśli to to samo ciało, pomiń
		if &b1 == &b2 {
			continue
		}

		dir := b2.Pos.Sub(b1.Pos)
		d2 := dir.Len()*dir.Len() + epsilon*epsilon // softening
		fmag := G * b1.Mass * b2.Mass / d2
		// jeśli b2.Anti -> odpychanie: zmień znak siły
		if b2.Anti {
			fmag = -fmag
		}
		acc := dir.Normalize().Mul(fmag / b1.Mass)
		force = force.Add(acc)
	}

	return force
}
