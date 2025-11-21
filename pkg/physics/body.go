package physics

import (
	"image/color"
	"math"
)

// --- Wektor 2D ---
type Vec2 struct {
	X, Y float64
}

func (v Vec2) Add(o Vec2) Vec2 {
	return Vec2{v.X + o.X, v.Y + o.Y}
}

func (v Vec2) Sub(o Vec2) Vec2 {
	return Vec2{v.X - o.X, v.Y - o.Y}
}

func (v Vec2) Mul(s float64) Vec2 {
	return Vec2{v.X * s, v.Y * s}
}

func (v Vec2) Len() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}

func (v Vec2) Normalize() Vec2 {
	l := v.Len()
	if l == 0 {
		return Vec2{0, 0}
	}
	return Vec2{v.X / l, v.Y / l}
}

// --- Ciało fizyczne ---
type Body struct {
	Mass   float64
	Pos    Vec2
	Vel    Vec2
	Acc    Vec2
	Radius float64
	ColorC color.RGBA

	// jeśli Locked == true, ciało jest unieruchomione i nie porusza się
	Locked bool
	// jeśli Anti == true, ciało generuje anty-grawitację (odpycha zamiast przyciągać)
	Anti bool
}

func (b *Body) Update(dt float64, bodies []Body) {
	if b.Locked {
		// nie poruszamy zablokowanego ciała
		b.Acc = ComputeAcceleration(*b, bodies)
		b.Vel = Vec2{0, 0}
		return
	}
	b.Acc = ComputeAcceleration(*b, bodies)
	b.Vel = b.Vel.Add(b.Acc.Mul(dt))
	b.Pos = b.Pos.Add(b.Vel.Mul(dt))
}

func (b Body) Color() color.Color {
	return b.ColorC
}
