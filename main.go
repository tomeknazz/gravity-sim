package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"gravity-sim/pkg/physics"
	"gravity-sim/pkg/simulation"
)

const (
	screenWidth  = 2500
	screenHeight = 1400
	trailMaxLife = 120.0 // czas życia śladu w sekundach
)

// --- Segment śladu planety ---
type TrailSegment struct {
	X0, Y0, X1, Y1 float64
	Life           float64
	Color          color.RGBA
}

// --- Gra / symulacja ---
type Game struct {
	sim     *simulation.Simulator
	trails  [][]TrailSegment
	lastPos []physics.Vec2
}

// --- Aktualizacja symulacji i śladów ---
func (g *Game) Update() error {
	g.sim.Update()

	for i, b := range g.sim.Bodies {
		// dodaj nowy segment od ostatniej pozycji do bieżącej
		seg := TrailSegment{
			X0:    float64(screenWidth)/2 + g.lastPos[i].X,
			Y0:    float64(screenHeight)/2 + g.lastPos[i].Y,
			X1:    float64(screenWidth)/2 + b.Pos.X,
			Y1:    float64(screenHeight)/2 + b.Pos.Y,
			Life:  trailMaxLife,
			Color: b.ColorC,
		}
		g.trails[i] = append(g.trails[i], seg)
		g.lastPos[i] = b.Pos

		// zmniejsz Life segmentów i usuń wygasłe
		newTrail := g.trails[i][:0]
		for j := range g.trails[i] {
			g.trails[i][j].Life -= g.sim.Dt
			if g.trails[i][j].Life > 0 {
				newTrail = append(newTrail, g.trails[i][j])
			}
		}
		g.trails[i] = newTrail
	}

	return nil
}

// --- Rysowanie okręgu pikselami ---
func drawCircle(screen *ebiten.Image, cx, cy, r float64, clr color.RGBA) {
	for y := -int(r); y <= int(r); y++ {
		for x := -int(r); x <= int(r); x++ {
			if x*x+y*y <= int(r*r) {
				screen.Set(int(cx)+x, int(cy)+y, clr)
			}
		}
	}
}

// --- Rysowanie segmentu z gładkim fade ---
func drawSegment(screen *ebiten.Image, s TrailSegment) {
	// użycie pierwiastka dla wolniejszego zanikania
	alpha := uint8(255 * (s.Life / trailMaxLife))
	if alpha > 255 {
		alpha = 255
	}
	c := s.Color
	c.A = alpha
	drawLine(screen, s.X0, s.Y0, s.X1, s.Y1, c)
}

// --- Rysowanie linii między dwoma punktami ---
func drawLine(img *ebiten.Image, x0, y0, x1, y1 float64, clr color.RGBA) {
	dx := x1 - x0
	dy := y1 - y0
	steps := int(max(abs(dx), abs(dy)))
	if steps == 0 {
		img.Set(int(x0), int(y0), clr)
		return
	}
	xInc := dx / float64(steps)
	yInc := dy / float64(steps)
	x := x0
	y := y0
	for i := 0; i <= steps; i++ {
		img.Set(int(x), int(y), clr)
		x += xInc
		y += yInc
	}
}

func abs(a float64) float64 {
	if a < 0 {
		return -a
	}
	return a
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// --- Rysowanie planet i ich śladów ---
func (g *Game) Draw(screen *ebiten.Image) {
	// najpierw rysowanie śladów
	for _, trail := range g.trails {
		for _, s := range trail {
			drawSegment(screen, s)
		}
	}

	// potem planety
	for _, b := range g.sim.Bodies {
		x := float64(screenWidth)/2 + b.Pos.X
		y := float64(screenHeight)/2 + b.Pos.Y
		drawCircle(screen, x, y, b.Radius, b.ColorC)
	}

	// nazwa środowiska
	ebitenutil.DebugPrint(screen, fmt.Sprintf("Environment: %s", g.sim.Name))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// --- Główny program ---
func main() {
	envName := flag.String("env", "solar", "Wybór środowiska (np. solar, binary, chaos)")
	flag.Parse()

	configPath := filepath.Join("pkg/assets", fmt.Sprintf("%s.json", *envName))

	// Wczytaj środowisko
	sim, err := simulation.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Błąd wczytywania środowiska: %v", err)
	}

	// Inicjalizacja trailów i pozycji
	lastPos := make([]physics.Vec2, len(sim.Bodies))
	trails := make([][]TrailSegment, len(sim.Bodies))
	for i, b := range sim.Bodies {
		lastPos[i] = b.Pos
		trails[i] = []TrailSegment{}
		// domyślny kolor jeśli brak w JSON
		if b.ColorC == (color.RGBA{}) {
			b.ColorC = color.RGBA{200, 200, 255, 255}
		}
	}

	game := &Game{
		sim:     sim,
		trails:  trails,
		lastPos: lastPos,
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Gravity Simulation - " + sim.Name)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
