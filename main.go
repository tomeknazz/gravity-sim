package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"

	"golang.org/x/image/font/basicfont"

	"gravity-sim/pkg/physics"
	"gravity-sim/pkg/simulation"
)

const (
	screenWidth  = 1920
	screenHeight = 1000
	trailMaxLife = 120.0 // czas życia śladu w sekundach

	// UI
	uiBtnW   = 100
	uiBtnH   = 28
	uiBtnPad = 12

	// wykres
	graphW = 360
	graphH = 120
)

// TrailSegment ---
type TrailSegment struct {
	X0, Y0, X1, Y1 float64
	Life           float64
	Color          color.RGBA
}

// Game ---
type Game struct {
	sim     *simulation.Simulator
	trails  [][]TrailSegment
	lastPos []physics.Vec2
	paused  bool

	selA int
	selB int

	showComponents  bool
	forceHistory    []float64
	forceHistoryMax int
}

// Update ---
func (g *Game) Update() error {
	// klawisze
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.paused = !g.paused
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyN) && g.paused {
		g.advanceOneStep()
	}

	// UI kliknięcia
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		// przyciski od prawej: Pause, Step, Quit, Comp (od lewej do prawej: Comp Quit Step Pause)
		pauseX := screenWidth - uiBtnPad - uiBtnW
		pauseY := uiBtnPad
		stepX := pauseX - uiBtnPad - uiBtnW
		stepY := uiBtnPad
		quitX := stepX - uiBtnPad - uiBtnW
		quitY := uiBtnPad
		compX := quitX - uiBtnPad - uiBtnW
		compY := uiBtnPad

		// obsłuż UI
		if pointInRect(mx, my, compX, compY, uiBtnW, uiBtnH) {
			if g.selA != -1 && g.selB != -1 {
				g.showComponents = !g.showComponents
			}
			return nil
		}
		if pointInRect(mx, my, quitX, quitY, uiBtnW, uiBtnH) {
			os.Exit(0)
			return nil
		}
		if pointInRect(mx, my, stepX, stepY, uiBtnW, uiBtnH) {
			if g.paused {
				g.advanceOneStep()
			}
			return nil
		}
		if pointInRect(mx, my, pauseX, pauseY, uiBtnW, uiBtnH) {
			g.paused = !g.paused
			return nil
		}

		// kliknięcie poza UI: wybieranie ciała
		mouse := physics.Vec2{X: float64(mx) - float64(screenWidth)/2, Y: float64(my) - float64(screenHeight)/2}
		clicked := -1
		minD := 1e18
		for i := range g.sim.Bodies {
			b := &g.sim.Bodies[i]
			d := b.Pos.Sub(mouse).Len()
			if d <= b.Radius && d < minD {
				clicked = i
				minD = d
			}
		}
		if clicked >= 0 {
			prevA, prevB := g.selA, g.selB
			if g.selA == -1 {
				g.selA = clicked
			} else if g.selB == -1 {
				if clicked == g.selA {
					g.selA = -1
				} else {
					g.selB = clicked
				}
			} else {
				if clicked == g.selA {
					g.selA = -1
					g.selB = -1
				} else if clicked == g.selB {
					g.selB = -1
				} else {
					g.selA = clicked
					g.selB = -1
				}
			}
			if g.selA != prevA || g.selB != prevB {
				g.forceHistory = nil
			}
		}
	}

	if g.paused {
		return nil
	}

	g.advanceOneStep()
	return nil
}

// advanceOneStep ---
func (g *Game) advanceOneStep() {
	g.sim.Update()
	// jeśli zaznaczone 2 ciała, oblicz siłę
	if g.selA != -1 && g.selB != -1 {
		b1 := g.sim.Bodies[g.selA]
		b2 := g.sim.Bodies[g.selB]
		dx := b2.Pos.X - b1.Pos.X
		dy := b2.Pos.Y - b1.Pos.Y
		d := math.Hypot(dx, dy)
		eps := 1e-6
		F := physics.G * b1.Mass * b2.Mass / (d*d + eps)
		g.forceHistory = append(g.forceHistory, F)
		if g.forceHistoryMax == 0 {
			g.forceHistoryMax = 600
		}
		if len(g.forceHistory) > g.forceHistoryMax {
			start := len(g.forceHistory) - g.forceHistoryMax
			g.forceHistory = g.forceHistory[start:]
		}
	}

	// update śladów
	for i := range g.sim.Bodies {
		b := g.sim.Bodies[i]
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
		// trim
		newTrail := g.trails[i][:0]
		for j := range g.trails[i] {
			g.trails[i][j].Life -= g.sim.Dt
			if g.trails[i][j].Life > 0 {
				newTrail = append(newTrail, g.trails[i][j])
			}
		}
		g.trails[i] = newTrail
	}
}

// Draw helpers ---
func drawCircle(screen *ebiten.Image, cx, cy, r float64, clr color.RGBA) {
	for y := -int(r); y <= int(r); y++ {
		for x := -int(r); x <= int(r); x++ {
			if x*x+y*y <= int(r*r) {
				screen.Set(int(cx)+x, int(cy)+y, clr)
			}
		}
	}
}

func drawLine(img *ebiten.Image, x0, y0, x1, y1 float64, clr color.RGBA) {
	dx := x1 - x0
	dy := y1 - y0
	steps := int(maxf(abs(dx), abs(dy)))
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
func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func pointInRect(px, py, rx, ry, rw, rh int) bool {
	return px >= rx && px <= rx+rw && py >= ry && py <= ry+rh
}

func drawButton(screen *ebiten.Image, x, y, w, h int, label string, active bool, disabled bool, hover bool) {
	btn := ebiten.NewImage(w, h)
	bg := color.RGBA{20, 20, 20, 200}
	textColor := color.RGBA{240, 240, 240, 255}
	if disabled {
		bg = color.RGBA{60, 60, 60, 160}
		textColor = color.RGBA{160, 160, 160, 200}
	} else {
		if active {
			bg = color.RGBA{60, 120, 60, 220}
		}
		if hover {
			if active {
				bg = color.RGBA{100, 190, 100, 240}
			} else {
				bg = color.RGBA{90, 90, 90, 230}
			}
		}
	}
	btn.Fill(bg)
	inner := ebiten.NewImage(w-2, h-2)
	inner.Fill(color.RGBA{40, 40, 40, 120})
	opInner := &ebiten.DrawImageOptions{}
	opInner.GeoM.Translate(1, 1)
	btn.DrawImage(inner, opInner)
	charW := 7
	cw := len(label) * charW
	xText := (w - cw) / 2
	yText := (h + 8) / 2
	text.Draw(btn, label, basicfont.Face7x13, xText, yText, textColor)
	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(btn, op2)
}

func drawArrowWithHead(img *ebiten.Image, x0, y0, x1, y1 float64, clr color.RGBA) {
	drawLine(img, x0, y0, x1, y1, clr)
	dx := x1 - x0
	dy := y1 - y0
	d := math.Sqrt(dx*dx + dy*dy)
	if d == 0 {
		return
	}
	ux := dx / d
	uY := dy / d
	sz := 10.0
	px := -uY
	py := ux
	p1x := x1 - ux*sz + px*(sz*0.6)
	p1y := y1 - uY*sz + py*(sz*0.6)
	p2x := x1 - ux*sz - px*(sz*0.6)
	p2y := y1 - uY*sz - py*(sz*0.6)
	drawLine(img, x1, y1, p1x, p1y, clr)
	drawLine(img, x1, y1, p2x, p2y, clr)
}

// drawForceGraph rysuje prosty wykres liniowy w podanym prostokącie
func drawForceGraph(screen *ebiten.Image, data []float64, x, y, w, h int) {
	if len(data) == 0 {
		return
	}
	bg := ebiten.NewImage(w, h)
	bg.Fill(color.RGBA{8, 8, 16, 200})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(bg, op)

	// obramowanie
	border := ebiten.NewImage(w-2, h-2)
	border.Fill(color.RGBA{30, 30, 40, 80})
	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(float64(x+1), float64(y+1))
	screen.DrawImage(border, op2)

	maxV := 0.0
	for _, v := range data {
		if v > maxV {
			maxV = v
		}
	}
	if maxV <= 0 {
		maxV = 1
	}
	padding := 6
	gw := float64(w - padding*2)
	gh := float64(h - padding*2)
	for i := 0; i <= 4; i++ {
		yy := float64(y+padding) + gh*float64(i)/4.0
		drawLine(screen, float64(x+padding), yy, float64(x+w-padding), yy, color.RGBA{40, 40, 60, 120})
	}
	n := len(data)
	if n < 2 {
		return
	}
	stepX := gw / float64(n-1)
	var px, py float64
	for i, v := range data {
		nx := float64(x+padding) + stepX*float64(i)
		nv := v / maxV
		ny := float64(y+padding) + gh*(1.0-nv)
		if i > 0 {
			drawLine(screen, px, py, nx, ny, color.RGBA{180, 220, 255, 255})
		}
		px = nx
		py = ny
	}
	lbl := fmt.Sprintf("max=%.3e", maxV)
	text.Draw(screen, lbl, basicfont.Face7x13, x+8, y+12, color.RGBA{180, 180, 200, 255})
}

// Draw ---
func (g *Game) Draw(screen *ebiten.Image) {
	// trails
	for _, trail := range g.trails {
		for _, s := range trail {
			drawLine(screen, s.X0, s.Y0, s.X1, s.Y1, s.Color)
		}
	}
	// bodies
	for i := range g.sim.Bodies {
		b := g.sim.Bodies[i]
		x := float64(screenWidth)/2 + b.Pos.X
		y := float64(screenHeight)/2 + b.Pos.Y
		drawCircle(screen, x, y, b.Radius, b.ColorC)
		if i == g.selA || i == g.selB {
			drawCircle(screen, x, y, b.Radius+3, color.RGBA{255, 255, 255, 180})
		}
	}

	// UI
	ebitenutil.DebugPrint(screen, fmt.Sprintf("Env: %s\nPaused: %v", g.sim.Name, g.paused))
	pauseX := screenWidth - uiBtnPad - uiBtnW
	pauseY := uiBtnPad
	stepX := pauseX - uiBtnPad - uiBtnW
	stepY := uiBtnPad
	quitX := stepX - uiBtnPad - uiBtnW
	quitY := uiBtnPad
	compX := quitX - uiBtnPad - uiBtnW
	compY := uiBtnPad
	mx, my := ebiten.CursorPosition()
	hoverComp := pointInRect(mx, my, compX, compY, uiBtnW, uiBtnH)
	hoverQuit := pointInRect(mx, my, quitX, quitY, uiBtnW, uiBtnH)
	hoverStep := pointInRect(mx, my, stepX, stepY, uiBtnW, uiBtnH)
	hoverPause := pointInRect(mx, my, pauseX, pauseY, uiBtnW, uiBtnH)
	compDisabled := !(g.selA != -1 && g.selB != -1)
	drawButton(screen, compX, compY, uiBtnW, uiBtnH, "Comp", g.showComponents, compDisabled, hoverComp)
	drawButton(screen, quitX, quitY, uiBtnW, uiBtnH, "Quit", false, false, hoverQuit)
	drawButton(screen, stepX, stepY, uiBtnW, uiBtnH, "Step", false, !g.paused, hoverStep)
	pauseLabel := "Pause"
	if g.paused {
		pauseLabel = "Resume"
	}
	drawButton(screen, pauseX, pauseY, uiBtnW, uiBtnH, pauseLabel, g.paused, false, hoverPause)

	// arrow + force + graph
	if g.selA != -1 && g.selB != -1 {
		b1 := g.sim.Bodies[g.selA]
		b2 := g.sim.Bodies[g.selB]
		x1 := float64(screenWidth)/2 + b1.Pos.X
		y1 := float64(screenHeight)/2 + b1.Pos.Y
		x2 := float64(screenWidth)/2 + b2.Pos.X
		y2 := float64(screenHeight)/2 + b2.Pos.Y
		drawArrowWithHead(screen, x1, y1, x2, y2, color.RGBA{255, 200, 0, 220})
		dx := b2.Pos.X - b1.Pos.X
		dy := b2.Pos.Y - b1.Pos.Y
		d := math.Hypot(dx, dy)
		F := physics.G * b1.Mass * b2.Mass / (d*d + 1e-6)
		midX := (x1 + x2) / 2
		midY := (y1 + y2) / 2
		label := fmt.Sprintf("F = %.3e", F)
		text.Draw(screen, label, basicfont.Face7x13, int(midX)-len(label)*4, int(midY)-6, color.RGBA{255, 255, 200, 255})
		if g.showComponents {
			uX := dx / (d + 1e-12)
			uY := dy / (d + 1e-12)
			Fx := F * uX
			Fy := F * uY
			text.Draw(screen, fmt.Sprintf("Fx=%.3e", Fx), basicfont.Face7x13, int(midX)-30, int(midY)+8, color.RGBA{200, 255, 200, 255})
			text.Draw(screen, fmt.Sprintf("Fy=%.3e", Fy), basicfont.Face7x13, int(midX)-30, int(midY)+20, color.RGBA{200, 255, 200, 255})
		}
		// graph bottom-right
		graphX := screenWidth - graphW - 16
		graphY := screenHeight - graphH - 16
		drawForceGraph(screen, g.forceHistory, graphX, graphY, graphW, graphH)
	}

	// tooltip podczas pauzy
	if g.paused {
		mx, my := ebiten.CursorPosition()
		mouse := physics.Vec2{X: float64(mx) - float64(screenWidth)/2, Y: float64(my) - float64(screenHeight)/2}
		var hovered *physics.Body
		minD := 1e18
		for i := range g.sim.Bodies {
			b := &g.sim.Bodies[i]
			d := b.Pos.Sub(mouse).Len()
			if d <= b.Radius && d < minD {
				hovered = b
				minD = d
			}
		}
		if hovered != nil {
			lines := []string{
				fmt.Sprintf("Mass: %.3e", hovered.Mass),
				fmt.Sprintf("Pos: (%.2f, %.2f)", hovered.Pos.X, hovered.Pos.Y),
				fmt.Sprintf("Vel: (%.2f, %.2f)", hovered.Vel.X, hovered.Vel.Y),
				fmt.Sprintf("Speed: %.2f", hovered.Vel.Len()),
				fmt.Sprintf("Radius: %.2f", hovered.Radius),
			}
			pad := 6
			charW := 7
			lineH := 13
			maxLen := 0
			for _, l := range lines {
				if len(l) > maxLen {
					maxLen = len(l)
				}
			}
			tw := maxLen*charW + pad*2
			th := len(lines)*lineH + pad*2
			tooltip := ebiten.NewImage(tw, th)
			tooltip.Fill(color.RGBA{10, 10, 10, 200})
			inner := ebiten.NewImage(tw-2, th-2)
			inner.Fill(color.RGBA{30, 30, 30, 120})
			opInner := &ebiten.DrawImageOptions{}
			opInner.GeoM.Translate(1, 1)
			tooltip.DrawImage(inner, opInner)
			for i, l := range lines {
				x := pad
				y := pad + (i+1)*lineH - 2
				text.Draw(tooltip, l, basicfont.Face7x13, x, y, color.RGBA{230, 230, 230, 255})
			}
			drawX := mx + 12
			drawY := my + 12
			if drawX+tw > screenWidth {
				drawX = screenWidth - tw - 8
			}
			if drawY+th > screenHeight {
				drawY = screenHeight - th - 8
			}
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(drawX), float64(drawY))
			screen.DrawImage(tooltip, op)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	envName := flag.String("env", "solar", "Wybór środowiska (np. solar, binary, chaos)")
	flag.Parse()
	configPath := filepath.Join("pkg/assets", fmt.Sprintf("%s.json", *envName))

	sim, err := simulation.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Błąd wczytywania środowiska: %v", err)
	}
	lastPos := make([]physics.Vec2, len(sim.Bodies))
	trails := make([][]TrailSegment, len(sim.Bodies))
	for i := range sim.Bodies {
		lastPos[i] = sim.Bodies[i].Pos
		trails[i] = []TrailSegment{}
		if sim.Bodies[i].ColorC == (color.RGBA{}) {
			sim.Bodies[i].ColorC = color.RGBA{200, 200, 255, 255}
		}
	}
	game := &Game{
		sim:             sim,
		trails:          trails,
		lastPos:         lastPos,
		selA:            -1,
		selB:            -1,
		forceHistoryMax: 600,
	}
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Gravity Simulation - " + sim.Name)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
