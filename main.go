package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

//go:embed assets/*
var assets embed.FS

func mustLoadImage(name string) *ebiten.Image {
	f, err := assets.Open(name)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}

	return ebiten.NewImageFromImage(img)
}

var healthIcon = Sprite{
	Img:      mustLoadImage("assets/images/heart-icon.png"),
	Position: Vector{10, 20},
}

type Vector struct {
	X float64
	Y float64
}

var (
	scoreFont  = mustLoadFont("assets/fonts/InputMono-Regular.ttf", 48)
	healthFont = mustLoadFont("assets/fonts/VictorMono-Italic.ttf", 32)
)

func mustLoadFont(filePath string, fontSize float64) *text.GoTextFace {
	f, err := assets.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	t, err := text.NewGoTextFaceSource(bytes.NewReader(f))
	if err != nil {
		panic(err)
	}

	s := &text.GoTextFace{
		Source: t,
		Size:   fontSize,
	}

	return s
}

type Timer struct {
	currentTicks int
	targetTicks  int
}

func NewTimer(d time.Duration) *Timer {
	return &Timer{
		currentTicks: 0,
		targetTicks:  int(d.Milliseconds()) * ebiten.TPS() / 1000,
	}
}

func (t *Timer) Update() {
	if t.currentTicks < t.targetTicks {
		t.currentTicks++
	}
}

func (t *Timer) IsReady() bool {
	return t.currentTicks >= t.targetTicks
}

func (t *Timer) Reset() {
	t.currentTicks = 0
}

type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

func NewRect(x, y, width, height float64) Rect {
	return Rect{
		X:      x,
		Y:      y,
		Width:  width,
		Height: height,
	}
}

func (r Rect) MaxX() float64 {
	return r.X + r.Width
}

func (r Rect) MaxY() float64 {
	return r.Y + r.Height
}

func (r Rect) Intersects(other Rect) bool {
	return r.X <= other.MaxX() &&
		other.X <= r.MaxX() &&
		r.Y <= other.MaxY() &&
		other.Y <= r.MaxY()
}

type Sprite struct {
	Img      *ebiten.Image
	Position Vector
}

type Player struct {
	*Sprite
	Lives uint16
}

func initPlayer() *Player {
	playerSprite := mustLoadImage("assets/images/robo.png")

	bounds := playerSprite.Bounds()
	halfW := float64(bounds.Dx()) / 2

	pos := Vector{
		X: screenWidth/2 - halfW,
		Y: 420,
	}

	return &Player{
		Sprite: &Sprite{
			Img:      playerSprite,
			Position: pos,
		},
		Lives: 3,
	}
}

func (p *Player) Update() {
	speed := float64(600 / ebiten.TPS())
	rightOffset := 25

	if p.Position.X > 0 {
		if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
			p.Position.X -= speed
		}
	}

	if p.Position.X < float64(screenWidth-p.Img.Bounds().Dx()-rightOffset) {
		if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
			p.Position.X += speed
		}
	}
}

func (p *Player) Draw(screen *ebiten.Image) {
	bounds := p.Img.Bounds()
	halfW := float64(bounds.Dx()) / 2
	halfH := float64(bounds.Dy()) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1.5, 1.5)
	op.GeoM.Translate(-halfW, -halfH)
	op.GeoM.Translate(halfW, halfH)

	op.Filter = ebiten.FilterLinear
	op.GeoM.Translate(p.Position.X, p.Position.Y)

	screen.DrawImage(p.Img, op)
}

func (p *Player) Collider() Rect {
	bounds := p.Img.Bounds()

	return NewRect(
		p.Position.X,
		p.Position.Y,
		float64(bounds.Dx()),
		float64(bounds.Dy()),
	)
}

type Food struct {
	*Sprite
}

func newFood() *Food {
	sprite := mustLoadImage("assets/images/bomba.png")
	x := rand.Float64() * (screenWidth - float64(sprite.Bounds().Dx()))
	y := -20.0

	return &Food{
		&Sprite{
			Img:      sprite,
			Position: Vector{X: x, Y: y},
		},
	}
}

func (f *Food) Draw(screen *ebiten.Image) {
	bounds := f.Img.Bounds()
	halfW := float64(bounds.Dx()) / 2
	halfH := float64(bounds.Dy()) / 2

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-halfW, -halfH)
	op.GeoM.Translate(halfW, halfH)

	op.GeoM.Scale(1.5, 1.5)
	op.Filter = ebiten.FilterLinear
	op.GeoM.Translate(f.Position.X, f.Position.Y)

	screen.DrawImage(f.Img, op)
}

func (f *Food) Update() {
	gravity := 10.0
	f.Position.Y += gravity
}

func (f *Food) Collider() Rect {
	bounds := f.Img.Bounds()

	return NewRect(
		f.Position.X,
		f.Position.Y,
		float64(bounds.Dx()),
		float64(bounds.Dy()),
	)
}

type Game struct {
	player         *Player
	food           []*Food
	foodSpawnTimer *Timer
	score          uint16
}

func initGame() *Game {
	g := &Game{
		player:         initPlayer(),
		foodSpawnTimer: NewTimer(1 * time.Second),
		score:          0,
	}
	g.Layout(screenWidth, screenHeight)
	return g
}

func (g *Game) Update() error {
	g.player.Update()

	g.foodSpawnTimer.Update()
	if g.foodSpawnTimer.IsReady() {
		g.foodSpawnTimer.Reset()

		f := newFood()
		g.food = append(g.food, f)
	}

	for i, f := range g.food {
		if f.Collider().Intersects(g.player.Collider()) {
			g.food = append(g.food[:i], g.food[i+1:]...)
			g.score++
		}
	}

	for _, f := range g.food {
		f.Update()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{100, 149, 237, 255})
	for _, f := range g.food {
		f.Draw(screen)
	}
	g.player.Draw(screen)

	to := &text.DrawOptions{}
	to.GeoM.Translate(screenWidth/2-100, 20)
	to.ColorScale.ScaleWithColor(color.White)

	text.Draw(screen, fmt.Sprintf("%06d", g.score), scoreFont, to)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(0.03, 0.03)
	op.GeoM.Translate(healthIcon.Position.X, healthIcon.Position.Y)
	op.Filter = ebiten.FilterLinear

	ho := &text.DrawOptions{}
	ho.GeoM.Translate(50, 17)
	ho.ColorScale.ScaleWithColor(color.RGBA{255, 0, 0, 255})

	text.Draw(screen, fmt.Sprintf("0%d", g.player.Lives), healthFont, ho)
	screen.DrawImage(healthIcon.Img, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func main() {
	g := initGame()

	err := ebiten.RunGame(g)
	if err != nil {
		panic(err)
	}
}
