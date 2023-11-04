package main

import (
	"embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/lafriks/go-tiled"
	"image"
	"log"
	"math/rand"
	"path"
	"time"
)

//go:embed assets/*
var EmbeddedAssets embed.FS

const (
	WINDOW_WIDTH     = 750
	WINDOW_HEIGHT    = 750
	GUY_FRAME_WIDTH  = 48
	GUY_HEIGHT       = 64
	FRAME_COUNT      = 4
	FRAMES_PER_SHEET = 3
)

const mapPath = "demo.tmx"
const (
	UP = iota
	RIGHT
	DOWN
	LEFT
	IDLE
)

type PlayerSprite struct {
	spriteSheet *ebiten.Image
	xLoc        int
	yLoc        int
	direction   int
	frame       int
	frameDelay  int
}

type SoldierSprite struct {
	PlayerSprite
	moveCount int
}

type AnimatedSprite struct {
	player   PlayerSprite
	soldiers [2]SoldierSprite
	gameMap  *tiled.Map
	tileDict map[uint32]*ebiten.Image
	barriers map[int]bool
}

func (tileGame *AnimatedSprite) Update() error {
	getPlayerInput(tileGame)
	for i := range tileGame.soldiers {
		updateSoldier(&tileGame.soldiers[i], tileGame)
	}

	if tileGame.player.direction != IDLE {
		tileGame.player.frameDelay++
		if tileGame.player.frameDelay%FRAME_COUNT == 0 {
			tileGame.player.frame++
			if tileGame.player.frame >= FRAMES_PER_SHEET {
				tileGame.player.frame = 0
			}
		}
	} else {

		tileGame.player.frame = 1
	}

	switch tileGame.player.direction {
	case LEFT:
		tileGame.player.xLoc -= 5
	case RIGHT:
		tileGame.player.xLoc += 5
	case UP:
		tileGame.player.yLoc -= 5
	case DOWN:
		tileGame.player.yLoc += 5
	}

	return nil
}

func updateSoldier(soldier *SoldierSprite, game *AnimatedSprite) {

	soldier.moveCount++

	if soldier.moveCount >= 60 {
		soldier.moveCount = 0
		soldier.direction = (soldier.direction + 1) % 4
	}

	if !isBarrier(soldier.xLoc, soldier.yLoc, game) {
		switch soldier.direction {
		case LEFT:
			soldier.xLoc -= 2
		case RIGHT:
			soldier.xLoc += 2
		}

		if soldier.xLoc < 0 {
			soldier.xLoc = 0
			soldier.direction = RIGHT
		} else if soldier.xLoc > WINDOW_WIDTH-GUY_FRAME_WIDTH {
			soldier.xLoc = WINDOW_WIDTH - GUY_FRAME_WIDTH
			soldier.direction = LEFT
		}
	}
}

func getPlayerInput(game *AnimatedSprite) {

	game.player.direction = IDLE

	if !isBarrier(game.player.xLoc, game.player.yLoc, game) && !isSoldierCollision(game.player.xLoc, game.player.yLoc, game) {

		if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) && game.player.xLoc > 0 {
			game.player.direction = LEFT
		} else if ebiten.IsKeyPressed(ebiten.KeyArrowRight) && game.player.xLoc < WINDOW_WIDTH-GUY_FRAME_WIDTH {
			game.player.direction = RIGHT
		} else if ebiten.IsKeyPressed(ebiten.KeyArrowUp) && game.player.yLoc > 0 {
			game.player.direction = UP
		} else if ebiten.IsKeyPressed(ebiten.KeyArrowDown) && game.player.yLoc < WINDOW_HEIGHT-GUY_HEIGHT {
			game.player.direction = DOWN
		}
	}
}

func isBarrier(x, y int, game *AnimatedSprite) bool {
	tileX := x / game.gameMap.TileWidth
	tileY := y / game.gameMap.TileHeight
	tileIndex := tileY*game.gameMap.Width + tileX
	return game.barriers[tileIndex]
}

func isSoldierCollision(x, y int, game *AnimatedSprite) bool {
	for _, soldier := range game.soldiers {
		if x == soldier.xLoc && y == soldier.yLoc {
			return true
		}
	}
	return false
}

func (tileGame *AnimatedSprite) Draw(screen *ebiten.Image) {

	for y := 0; y < tileGame.gameMap.Height; y++ {
		for x := 0; x < tileGame.gameMap.Width; x++ {
			tile := tileGame.gameMap.Layers[0].Tiles[y*tileGame.gameMap.Width+x]
			if tileGID := tile.ID; tileGID != 0 {
				tileImage := tileGame.tileDict[tileGID]
				if tileImage != nil {
					opts := &ebiten.DrawImageOptions{}
					opts.GeoM.Translate(float64(x*tileGame.gameMap.TileWidth), float64(y*tileGame.gameMap.TileHeight))
					screen.DrawImage(tileImage, opts)
				}
			}
		}
	}

	drawSprite(screen, &tileGame.player)

	for _, soldier := range tileGame.soldiers {
		drawSprite(screen, &soldier.PlayerSprite)
	}
}

func drawSprite(screen *ebiten.Image, sprite *PlayerSprite) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Reset()
	op.GeoM.Translate(float64(sprite.xLoc), float64(sprite.yLoc))

	var spriteY int
	if sprite.direction != IDLE {
		spriteY = sprite.direction * GUY_HEIGHT
	} else {

		spriteY = 2 * GUY_HEIGHT
		sprite.frame = 1
	}

	screen.DrawImage(sprite.spriteSheet.SubImage(image.Rect(
		sprite.frame*GUY_FRAME_WIDTH,
		spriteY,
		(sprite.frame+1)*GUY_FRAME_WIDTH,
		spriteY+GUY_HEIGHT)).(*ebiten.Image), op)
}

func (tileGame AnimatedSprite) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func makeEbitenImagesFromMap(tiledMap *tiled.Map) (map[uint32]*ebiten.Image, error) {
	idToImage := make(map[uint32]*ebiten.Image)
	for _, tileset := range tiledMap.Tilesets {
		for _, tile := range tileset.Tiles {
			ebitenImageTile, _, err := ebitenutil.NewImageFromFile(tile.Image.Source)
			if err != nil {
				return nil, fmt.Errorf("error loading tile image: %s: %w", tile.Image.Source, err)
			}
			idToImage[tile.ID+tileset.FirstGID-1] = ebitenImageTile // Adjust ID by FirstGID
		}
	}
	return idToImage, nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Load the .tmx file to get the tile map data.
	gameMap, err := tiled.LoadFile(mapPath)
	if err != nil {
		log.Fatalf("error parsing map: %v", err)
	}

	// Calculate the window size based on the map dimensions.
	windowWidth := gameMap.Width * gameMap.TileWidth
	windowHeight := gameMap.Height * gameMap.TileHeight
	ebiten.SetWindowSize(windowWidth, windowHeight)
	ebiten.SetWindowTitle("Tile Map Demo")

	// Load the images for the tiles.
	ebitenImageMap, err := makeEbitenImagesFromMap(gameMap)
	if err != nil {
		log.Fatalf("error loading tile images: %v", err)
	}

	// Create a map of barrier tiles.
	barriers := make(map[int]bool)
	for y := 0; y < gameMap.Height; y++ {
		for x := 0; x < gameMap.Width; x++ {
			tile := gameMap.Layers[0].Tiles[y*gameMap.Width+x]
			if tile.ID == 1 { // Assuming '1' is the ID for barrier tiles
				barriers[y*gameMap.Width+x] = true
			}
		}
	}

	animationGuy := LoadEmbeddedImage("wizard.png")
	soldierImage := LoadEmbeddedImage("soldier1.png")

	myPlayer := PlayerSprite{
		spriteSheet: animationGuy,
		xLoc:        windowWidth/2 - GUY_FRAME_WIDTH/2,
		yLoc:        windowHeight/2 - GUY_HEIGHT/2,
		direction:   IDLE,
	}

	demo := AnimatedSprite{
		player: myPlayer,
		soldiers: [2]SoldierSprite{
			{
				PlayerSprite: PlayerSprite{
					spriteSheet: soldierImage,
					xLoc:        rand.Intn(windowWidth - GUY_FRAME_WIDTH),
					yLoc:        rand.Intn(windowHeight - GUY_HEIGHT),
					direction:   IDLE,
				},
				moveCount: 0,
			},
			{
				PlayerSprite: PlayerSprite{
					spriteSheet: soldierImage,
					xLoc:        rand.Intn(windowWidth - GUY_FRAME_WIDTH),
					yLoc:        rand.Intn(windowHeight - GUY_HEIGHT),
					direction:   IDLE,
				},
				moveCount: 0,
			},
		},
		gameMap:  gameMap,
		tileDict: ebitenImageMap,
		barriers: barriers,
	}

	// Run the game.
	if err := ebiten.RunGame(&demo); err != nil {
		log.Fatalf("failed to run game: %v", err)
	}
}

func LoadEmbeddedImage(imageName string) *ebiten.Image {
	embeddedFile, err := EmbeddedAssets.Open(path.Join("assets", imageName))
	if err != nil {
		log.Fatalf("failed to load embedded image %s: %v", imageName, err)
	}
	defer embeddedFile.Close()

	ebitenImage, _, err := ebitenutil.NewImageFromReader(embeddedFile)
	if err != nil {
		log.Fatalf("Error loading tile image: %s: %v", imageName, err)
	}
	return ebitenImage
}
