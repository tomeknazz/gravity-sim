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
}

func (b *Body) Update(dt float64, bodies []Body) {
	b.Acc = ComputeAcceleration(*b, bodies)
	b.Vel = b.Vel.Add(b.Acc.Mul(dt))
	b.Pos = b.Pos.Add(b.Vel.Mul(dt))
}

func (b Body) Color() color.Color {
	return b.ColorC
}
