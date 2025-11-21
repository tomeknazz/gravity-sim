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

	// small controls
	smallBtnW = 48
	smallBtnH = 22

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

	// historie komponentów siły
	fxHistory []float64
	fyHistory []float64

	// Add mode: narzędzie dodawania nowych ciał
	addMode   bool    // czy jesteśmy w trybie dodawania
	addLocked bool    // czy nowe ciało będzie zablokowane
	addAnti   bool    // czy nowe ciało będzie anty-grawitacyjne
	addMass   float64 // domyślna masa nowego ciała
	addRadius float64 // domyślny promień nowego ciała

	// widoczność panelu skrótów
	shortcutsVisible bool

	// ścieżka do oryginalnego pliku konfiguracyjnego (do resetu)
	initialConfigPath string

	// czy modal potwierdzenia resetu jest otwarty
	resetModalOpen bool
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

	// Toggle shortcuts visibility
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		g.shortcutsVisible = !g.shortcutsVisible
	}

	// przełączniki w trybie Add (L - locked, V - anti)
	if g.addMode {
		if inpututil.IsKeyJustPressed(ebiten.KeyL) {
			g.addLocked = !g.addLocked
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyV) {
			g.addAnti = !g.addAnti
		}
	} else {
		// gdy nie w trybie add: pozwól na togglowanie Locked/Anti dla wybranego ciała (selA)
		if inpututil.IsKeyJustPressed(ebiten.KeyL) && g.selA != -1 {
			g.sim.Bodies[g.selA].Locked = !g.sim.Bodies[g.selA].Locked
			if g.sim.Bodies[g.selA].Locked {
				g.sim.Bodies[g.selA].ColorC = color.RGBA{200, 200, 200, 255}
			} else {
				g.sim.Bodies[g.selA].ColorC = color.RGBA{200, 200, 255, 255}
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyV) && g.selA != -1 {
			g.sim.Bodies[g.selA].Anti = !g.sim.Bodies[g.selA].Anti
			if g.sim.Bodies[g.selA].Anti {
				g.sim.Bodies[g.selA].ColorC = color.RGBA{255, 120, 120, 255}
			} else {
				g.sim.Bodies[g.selA].ColorC = color.RGBA{200, 200, 255, 255}
			}
		}
		// klawisze do zmiany masy/promienia dla selA
		if g.selA != -1 {
			if inpututil.IsKeyJustPressed(ebiten.KeyEqual) || inpututil.IsKeyJustPressed(ebiten.KeyK) { // = or K increase mass
				g.sim.Bodies[g.selA].Mass *= 1.1
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyMinus) || inpututil.IsKeyJustPressed(ebiten.KeyJ) { // - or J decrease mass
				g.sim.Bodies[g.selA].Mass *= 0.9
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyR) { // R increase radius
				g.sim.Bodies[g.selA].Radius *= 1.1
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyT) { // T decrease radius
				g.sim.Bodies[g.selA].Radius *= 0.9
			}
		}
	}

	// UI kliknięcia
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		// pozycyjne obiczenia przyciskow (Pause, Step, Quit, Comp, Reset, Add)
		pauseX := screenWidth - uiBtnPad - uiBtnW
		pauseY := uiBtnPad
		stepX := pauseX - uiBtnPad - uiBtnW
		stepY := uiBtnPad
		quitX := stepX - uiBtnPad - uiBtnW
		quitY := uiBtnPad
		compX := quitX - uiBtnPad - uiBtnW
		compY := uiBtnPad
		resetX := compX - uiBtnPad - uiBtnW
		addX := resetX - uiBtnPad - uiBtnW
		addY := uiBtnPad

		// small buttons to the left of Add
		massPlusX := addX - uiBtnPad - smallBtnW
		massPlusY := addY + (uiBtnH-smallBtnH)/2
		massMinusX := massPlusX - uiBtnPad - smallBtnW
		massMinusY := massPlusY
		radPlusX := massMinusX - uiBtnPad - smallBtnW
		radPlusY := massPlusY
		radMinusX := radPlusX - uiBtnPad - smallBtnW
		radMinusY := massPlusY

		// Jeśli modal potwierdzenia jest otwarty: obsłuż tylko modal
		if g.resetModalOpen {
			mw := 360
			mh := 120
			mx0 := (screenWidth - mw) / 2
			my0 := (screenHeight - mh) / 2
			yesX := mx0 + 40
			yesY := my0 + mh - 44
			noX := mx0 + mw - 40 - uiBtnW
			noY := yesY
			if pointInRect(mx, my, yesX, yesY, uiBtnW, uiBtnH) {
				// potwierdz reset
				if err := g.resetSimulation(); err != nil {
					log.Printf("Reset failed: %v", err)
				}
				return nil
			}
			if pointInRect(mx, my, noX, noY, uiBtnW, uiBtnH) {
				// anuluj modal
				g.resetModalOpen = false
				return nil
			}
			// klik poza modal zamyka modal
			g.resetModalOpen = false
			return nil
		}

		// obsłuż small buttons (założenie: działają tylko gdy jest zaznaczone selA)
		if pointInRect(mx, my, massPlusX, massPlusY, smallBtnW, smallBtnH) && g.selA != -1 {
			g.sim.Bodies[g.selA].Mass *= 1.1
			return nil
		}
		if pointInRect(mx, my, massMinusX, massMinusY, smallBtnW, smallBtnH) && g.selA != -1 {
			g.sim.Bodies[g.selA].Mass *= 0.9
			return nil
		}
		if pointInRect(mx, my, radPlusX, radPlusY, smallBtnW, smallBtnH) && g.selA != -1 {
			g.sim.Bodies[g.selA].Radius *= 1.1
			return nil
		}
		if pointInRect(mx, my, radMinusX, radMinusY, smallBtnW, smallBtnH) && g.selA != -1 {
			g.sim.Bodies[g.selA].Radius *= 0.9
			return nil
		}

		// obsłuż UI (Add/Comp/Quit/Step/Pause/Reset)
		if pointInRect(mx, my, addX, addY, uiBtnW, uiBtnH) {
			g.addMode = !g.addMode
			if g.addMode {
				g.addMass = 100.0
				g.addRadius = 8.0
				g.addLocked = false
				g.addAnti = false
			}
			return nil
		}
		if pointInRect(mx, my, compX, compY, uiBtnW, uiBtnH) {
			if g.selA != -1 && g.selB != -1 {
				g.showComponents = !g.showComponents
			}
			return nil
		}
		if pointInRect(mx, my, resetX, addY, uiBtnW, uiBtnH) {
			// otworz modal potwierdzenia resetu
			g.resetModalOpen = true
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

		// kliknięcie poza UI
		// jeśli jesteśmy w trybie add -> dodaj ciało w miejscu kursora
		if g.addMode {
			// upewnij się, że nie klikamy w obszar UI
			if !(pointInRect(mx, my, addX, addY, uiBtnW, uiBtnH) || pointInRect(mx, my, compX, compY, uiBtnW, uiBtnH) || pointInRect(mx, my, quitX, quitY, uiBtnW, uiBtnH) || pointInRect(mx, my, stepX, stepY, uiBtnW, uiBtnH) || pointInRect(mx, my, pauseX, pauseY, uiBtnW, uiBtnH)) {
				pos := physics.Vec2{X: float64(mx) - float64(screenWidth)/2, Y: float64(my) - float64(screenHeight)/2}
				// przygotuj ciało
				nb := physics.Body{
					Mass:   g.addMass,
					Pos:    pos,
					Vel:    physics.Vec2{0, 0},
					Acc:    physics.Vec2{0, 0},
					Radius: g.addRadius,
					ColorC: color.RGBA{200, 200, 255, 255},
					Locked: g.addLocked,
					Anti:   g.addAnti,
				}
				// kolor zależnie od flag
				if nb.Anti {
					nb.ColorC = color.RGBA{255, 120, 120, 255}
				} else if nb.Locked {
					nb.ColorC = color.RGBA{200, 200, 200, 255}
				}
				// dodaj do symulacji i pomocniczych tablic
				g.sim.Bodies = append(g.sim.Bodies, nb)
				g.lastPos = append(g.lastPos, nb.Pos)
				g.trails = append(g.trails, []TrailSegment{})
				// po dodaniu pozostajemy w trybie add (aby dodać kolejne) — chyba że chcesz inaczej
			}
			return nil
		}

		// normalne kliknięcie wyboru ciała (istniejąca logika)
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
				g.fxHistory = nil
				g.fyHistory = nil
			}
		}
	}

	// klawiszowa obsluga modalu
	if g.resetModalOpen {
		if inpututil.IsKeyJustPressed(ebiten.KeyY) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			if err := g.resetSimulation(); err != nil {
				log.Printf("Reset failed: %v", err)
			}
			return nil
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyN) || inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.resetModalOpen = false
			return nil
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
		// komponenty
		ux := dx / (d + 1e-12)
		uy := dy / (d + 1e-12)
		Fx := F * ux
		Fy := F * uy
		g.forceHistory = append(g.forceHistory, F)
		g.fxHistory = append(g.fxHistory, Fx)
		g.fyHistory = append(g.fyHistory, Fy)
		if g.forceHistoryMax == 0 {
			g.forceHistoryMax = 600
		}
		if len(g.forceHistory) > g.forceHistoryMax {
			start := len(g.forceHistory) - g.forceHistoryMax
			g.forceHistory = g.forceHistory[start:]
		}
		// trim fx/fy to same length
		if len(g.fxHistory) > g.forceHistoryMax {
			g.fxHistory = g.fxHistory[len(g.fxHistory)-g.forceHistoryMax:]
		}
		if len(g.fyHistory) > g.forceHistoryMax {
			g.fyHistory = g.fyHistory[len(g.fyHistory)-g.forceHistoryMax:]
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
// obsługuje autoskalowanie po osi Y, w tym zakresy ujemne.
func drawForceGraph(screen *ebiten.Image, data []float64, x, y, w, h int, lineColor color.RGBA, title string) {
	// tło i obramowanie (zawsze rysujemy panel nawet gdy brak danych)
	bg := ebiten.NewImage(w, h)
	bg.Fill(color.RGBA{8, 8, 16, 200})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(bg, op)

	border := ebiten.NewImage(w-2, h-2)
	border.Fill(color.RGBA{30, 30, 40, 80})
	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(float64(x+1), float64(y+1))
	screen.DrawImage(border, op2)

	if title != "" {
		text.Draw(screen, title, basicfont.Face7x13, x+6, y+14, color.RGBA{220, 220, 220, 200})
	}

	if len(data) == 0 {
		// brak danych - pozostaw panel
		return
	}

	// znajdź min i max
	minV := data[0]
	maxV := data[0]
	for _, v := range data {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}

	// jeśli mamy wartości po obu stronach 0, ustaw symetryczny zakres wokół zera
	if minV < 0 && maxV > 0 {
		b := math.Max(math.Abs(minV), math.Abs(maxV))
		minV = -b
		maxV = b
	}

	// jeśli min==max, rozszerz zakres
	if minV == maxV {
		maxV = maxV + 1.0
		minV = minV - 1.0
	} else {
		pad := 0.05 * (maxV - minV)
		maxV += pad
		minV -= pad
	}

	padding := 6
	gw := float64(w - padding*2)
	gh := float64(h - padding*2)

	// rysuj siatkę (4 poziome linie)
	for i := 0; i <= 4; i++ {
		yy := float64(y+padding) + gh*float64(i)/4.0
		drawLine(screen, float64(x+padding), yy, float64(x+w-padding), yy, color.RGBA{40, 40, 60, 120})
	}

	// linia zero (jeśli w zakresie)
	if minV <= 0 && maxV >= 0 {
		t := (0 - minV) / (maxV - minV)
		zy := float64(y+padding) + gh*(1.0-t)
		drawLine(screen, float64(x+padding), zy, float64(x+w-padding), zy, color.RGBA{150, 150, 150, 140})
	}

	// rysuj dane
	n := len(data)
	if n >= 2 {
		stepX := gw / float64(n-1)
		var px, py float64
		for i, v := range data {
			nx := float64(x+padding) + stepX*float64(i)
			t := (v - minV) / (maxV - minV)
			ny := float64(y+padding) + gh*(1.0-t)
			if i > 0 {
				drawLine(screen, px, py, nx, ny, lineColor)
			}
			px = nx
			py = ny
		}
	}

	// etykieta zakresu
	lbl := fmt.Sprintf("%.3e..%.3e", minV, maxV)
	text.Draw(screen, lbl, basicfont.Face7x13, x+6, y+h-6, color.RGBA{180, 180, 200, 180})
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
		// ikony Locked / Anti - małe symbole obok ciała
		iconX := x + b.Radius + 6
		iconY := y - b.Radius - 6
		if b.Locked {
			// rysuj prostą kłódkę: mały prostokąt z uchwytem
			lockW, lockH := 12.0, 8.0
			// prostokąt
			for yy := 0; yy < int(lockH); yy++ {
				for xx := 0; xx < int(lockW); xx++ {
					screen.Set(int(iconX)+xx, int(iconY)+yy, color.RGBA{180, 180, 180, 220})
				}
			}
			// uchwyt (linia)
			drawLine(screen, iconX+2, iconY-4, iconX+lockW-2, iconY-4, color.RGBA{180, 180, 180, 220})
		}
		if b.Anti {
			// rysuj kółko z minusem
			r := 6.0
			// circle outline
			drawLine(screen, iconX+20, iconY, iconX+20+r, iconY, color.RGBA{220, 120, 120, 220})
			// minus
			drawLine(screen, iconX+20-3, iconY, iconX+20+3, iconY, color.RGBA{220, 120, 120, 220})
		}
	}

	// UI
	ebitenutil.DebugPrint(screen, fmt.Sprintf("Env: %s\nPaused: %v", g.sim.Name, g.paused))
	drawShortcuts(screen, g)
	// rysowanie przycisków w prawym górnym rogu (dopisz Add)
	pauseX := screenWidth - uiBtnPad - uiBtnW
	pauseY := uiBtnPad
	stepX := pauseX - uiBtnPad - uiBtnW
	stepY := uiBtnPad
	quitX := stepX - uiBtnPad - uiBtnW
	quitY := uiBtnPad
	compX := quitX - uiBtnPad - uiBtnW
	compY := uiBtnPad
	resetX := compX - uiBtnPad - uiBtnW
	addX := resetX - uiBtnPad - uiBtnW
	addY := uiBtnPad
	massPlusX := addX - uiBtnPad - smallBtnW
	massPlusY := addY + (uiBtnH-smallBtnH)/2
	massMinusX := massPlusX - uiBtnPad - smallBtnW
	massMinusY := massPlusY
	radPlusX := massMinusX - uiBtnPad - smallBtnW
	radPlusY := massPlusY
	radMinusX := radPlusX - uiBtnPad - smallBtnW
	radMinusY := massPlusY

	// wykryj, czy kursor jest nad którymś przyciskiem
	mx, my := ebiten.CursorPosition()
	hoverAdd := pointInRect(mx, my, addX, addY, uiBtnW, uiBtnH)
	hoverComp := pointInRect(mx, my, compX, compY, uiBtnW, uiBtnH)
	hoverQuit := pointInRect(mx, my, quitX, quitY, uiBtnW, uiBtnH)
	hoverStep := pointInRect(mx, my, stepX, stepY, uiBtnW, uiBtnH)
	hoverPause := pointInRect(mx, my, pauseX, pauseY, uiBtnW, uiBtnH)
	hoverReset := pointInRect(mx, my, resetX, addY, uiBtnW, uiBtnH)
	compDisabled := !(g.selA != -1 && g.selB != -1)
	drawButton(screen, addX, addY, uiBtnW, uiBtnH, "Add", g.addMode, false, hoverAdd)
	drawButton(screen, compX, compY, uiBtnW, uiBtnH, "Comp", g.showComponents, compDisabled, hoverComp)
	drawButton(screen, quitX, quitY, uiBtnW, uiBtnH, "Quit", false, false, hoverQuit)
	drawButton(screen, stepX, stepY, uiBtnW, uiBtnH, "Step", false, !g.paused, hoverStep)
	pauseLabel := "Pause"
	if g.paused {
		pauseLabel = "Resume"
	}
	drawButton(screen, pauseX, pauseY, uiBtnW, uiBtnH, pauseLabel, g.paused, false, hoverPause)
	drawButton(screen, resetX, addY, uiBtnW, uiBtnH, "Reset", false, false, hoverReset)

	// rysuj small buttons (działają tylko dla zaznaczonego selA)
	drawButton(screen, massPlusX, massPlusY, smallBtnW, smallBtnH, "M+", false, g.selA == -1, pointInRect(mx, my, massPlusX, massPlusY, smallBtnW, smallBtnH))
	drawButton(screen, massMinusX, massMinusY, smallBtnW, smallBtnH, "M-", false, g.selA == -1, pointInRect(mx, my, massMinusX, massMinusY, smallBtnW, smallBtnH))
	drawButton(screen, radPlusX, radPlusY, smallBtnW, smallBtnH, "R+", false, g.selA == -1, pointInRect(mx, my, radPlusX, radPlusY, smallBtnW, smallBtnH))
	drawButton(screen, radMinusX, radMinusY, smallBtnW, smallBtnH, "R-", false, g.selA == -1, pointInRect(mx, my, radMinusX, radMinusY, smallBtnW, smallBtnH))

	// jeśli w trybie Add - pokaż podgląd pozycji i ustawienia
	if g.addMode {
		// kursorem nad ekranem
		mx, my := ebiten.CursorPosition()
		px := float64(mx)
		py := float64(my)
		col := color.RGBA{200, 200, 255, 160}
		if g.addAnti {
			col = color.RGBA{255, 120, 120, 180}
		} else if g.addLocked {
			col = color.RGBA{200, 200, 200, 200}
		}
		// rysuj podgląd koła
		preview := ebiten.NewImage(int(g.addRadius*2), int(g.addRadius*2))
		preview.Fill(col)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(px-g.addRadius, py-g.addRadius)
		screen.DrawImage(preview, op)
		// instrukcje
		text.Draw(screen, "Add mode: L toggle Locked, V toggle Anti", basicfont.Face7x13, 12, 60, color.RGBA{220, 220, 220, 200})
		settings := fmt.Sprintf("Mass: %.1f  Radius: %.1f  Locked: %v  Anti: %v", g.addMass, g.addRadius, g.addLocked, g.addAnti)
		text.Draw(screen, settings, basicfont.Face7x13, 12, 80, color.RGBA{200, 200, 200, 200})
	}

	// arrow + force + graph
	if g.selA != -1 && g.selB != -1 {
		b1 := g.sim.Bodies[g.selA]
		b2 := g.sim.Bodies[g.selB]
		x1 := float64(screenWidth)/2 + b1.Pos.X
		y1 := float64(screenHeight)/2 + b1.Pos.Y
		x2 := float64(screenWidth)/2 + b2.Pos.X
		y2 := float64(screenHeight)/2 + b2.Pos.Y
		// narysuj strzałkę od 1 do 2
		arrowColor := color.RGBA{255, 200, 0, 220}
		drawArrowWithHead(screen, x1, y1, x2, y2, arrowColor)
		// oblicz wartość siły i narysuj tekst w połowie
		dx := b2.Pos.X - b1.Pos.X
		dy := b2.Pos.Y - b1.Pos.Y
		dist := math.Hypot(dx, dy)
		eps := 1e-6
		force := physics.G * b1.Mass * b2.Mass / (dist*dist + eps)
		midX := (x1 + x2) / 2
		midY := (y1 + y2) / 2
		label := fmt.Sprintf("F = %.3e", force)
		text.Draw(screen, label, basicfont.Face7x13, int(midX)-len(label)*4, int(midY)-6, color.RGBA{255, 255, 200, 255})

		// jeśli komponenty włączone - wyświetl Fx/Fy i osobne wykresy
		graphX := screenWidth - graphW - 16
		baseY := screenHeight - graphH - 16
		step := graphH + 8
		if g.showComponents {
			// Fx (top)
			drawForceGraph(screen, g.fxHistory, graphX, baseY-step*2, graphW, graphH, color.RGBA{255, 100, 100, 255}, "Fx")
			// Fy (middle)
			drawForceGraph(screen, g.fyHistory, graphX, baseY-step, graphW, graphH, color.RGBA{100, 255, 100, 255}, "Fy")
			// F (bottom)
			drawForceGraph(screen, g.forceHistory, graphX, baseY, graphW, graphH, color.RGBA{100, 100, 255, 255}, "F")
		} else {
			// tylko F
			drawForceGraph(screen, g.forceHistory, graphX, baseY, graphW, graphH, color.RGBA{100, 100, 255, 255}, "")
		}
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

	// rysuj modal potwierdzenia resetu, jeśli otwarty
	if g.resetModalOpen {
		drawResetModal(screen)
	}
}

func drawShortcuts(screen *ebiten.Image, g *Game) {
	if !g.shortcutsVisible {
		return
	}
	// Zbierz linie kontekstowe (tylko klawisze, bez etykiet przyciskow)
	lines := []string{}
	if g.addMode {
		lines = append(lines, "ADD MODE")
		lines = append(lines, "L - toggle Locked (new body)")
		lines = append(lines, "V - toggle Anti (new body)")
		lines = append(lines, "Click - place new body")
		lines = append(lines, "K / =  - mass +")
		lines = append(lines, "J / -  - mass -")
		lines = append(lines, "R - radius +")
		lines = append(lines, "T - radius -")
		lines = append(lines, "H - hide shortcuts")
	} else {
		lines = append(lines, "GLOBAL")
		lines = append(lines, "P - Pause/Resume")
		lines = append(lines, "N - Step (when paused)")
		lines = append(lines, "L - toggle Locked (selected)")
		lines = append(lines, "V - toggle Anti (selected)")
		lines = append(lines, "K / =  - mass + (selected)")
		lines = append(lines, "J / -  - mass - (selected)")
		lines = append(lines, "R - radius + (selected)")
		lines = append(lines, "T - radius - (selected)")
		lines = append(lines, "H - hide shortcuts")
	}

	// Styl panelu
	pad := 6
	charW := 7
	lineH := 14
	maxLen := 0
	for _, l := range lines {
		if len(l) > maxLen {
			maxLen = len(l)
		}
	}
	w := maxLen*charW + pad*2
	h := len(lines)*lineH + pad*2
	// ogranicz rozmiar jeśli za duzy
	if w > 600 {
		w = 600
	}
	if h > 400 {
		h = 400
	}

	// stworz obraz panelu
	panel := ebiten.NewImage(w, h)
	panel.Fill(color.RGBA{10, 10, 20, 200})
	inner := ebiten.NewImage(w-2, h-2)
	inner.Fill(color.RGBA{30, 30, 40, 80})
	opInner := &ebiten.DrawImageOptions{}
	opInner.GeoM.Translate(1, 1)
	panel.DrawImage(inner, opInner)

	// narysuj tekst
	for i, l := range lines {
		x := pad
		y := pad + (i+1)*lineH - 2
		text.Draw(panel, l, basicfont.Face7x13, x, y, color.RGBA{220, 220, 220, 255})
	}

	// Pozycja: przesun doliej (nie zaslania DebugPrint w lewym gorze)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(12), float64(64))
	screen.DrawImage(panel, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

// resetSimulation przeładowuje konfigurację z initialConfigPath i resetuje stan gry
func (g *Game) resetSimulation() error {
	if g.initialConfigPath == "" {
		return fmt.Errorf("no initial config path set")
	}
	sim, err := simulation.LoadConfig(g.initialConfigPath)
	if err != nil {
		return err
	}
	// apply loaded simulator
	g.sim = sim
	// reinit helper arrays
	g.lastPos = make([]physics.Vec2, len(g.sim.Bodies))
	g.trails = make([][]TrailSegment, len(g.sim.Bodies))
	for i := range g.sim.Bodies {
		g.lastPos[i] = g.sim.Bodies[i].Pos
		g.trails[i] = []TrailSegment{}
		if g.sim.Bodies[i].ColorC == (color.RGBA{}) {
			g.sim.Bodies[i].ColorC = color.RGBA{200, 200, 255, 255}
		}
	}
	// clear selections and histories
	g.selA = -1
	g.selB = -1
	g.forceHistory = nil
	g.fxHistory = nil
	g.fyHistory = nil
	// close modal and reset modes
	g.addMode = false
	g.resetModalOpen = false
	g.paused = false
	return nil
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
		sim:               sim,
		trails:            trails,
		lastPos:           lastPos,
		selA:              -1,
		selB:              -1,
		forceHistoryMax:   600,
		shortcutsVisible:  true,
		initialConfigPath: configPath,
	}
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Gravity Simulation - " + sim.Name)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

// drawResetModal rysuje modal potwierdzenia resetu (nowa, czysta wersja)
func drawResetModal(screen *ebiten.Image) {
	w := 360
	h := 120
	x := (screenWidth - w) / 2
	y := (screenHeight - h) / 2
	panel := ebiten.NewImage(w, h)
	panel.Fill(color.RGBA{20, 20, 20, 220})
	inner := ebiten.NewImage(w-4, h-4)
	inner.Fill(color.RGBA{36, 36, 44, 200})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x+2), float64(y+2))
	panel.DrawImage(inner, op)

	text.Draw(panel, "Reset simulation?", basicfont.Face7x13, 16, 28, color.RGBA{230, 230, 230, 255})
	text.Draw(panel, "Reload initial config and remove added bodies.", basicfont.Face7x13, 16, 48, color.RGBA{190, 190, 190, 200})

	yesX := 40
	noX := w - 40 - uiBtnW
	btnY := h - 44
	drawButton(panel, yesX, btnY, uiBtnW, uiBtnH, "Yes", false, false, false)
	drawButton(panel, noX, btnY, uiBtnW, uiBtnH, "No", false, false, false)

	op2 := &ebiten.DrawImageOptions{}
	op2.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(panel, op2)
}
