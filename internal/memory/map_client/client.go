package map_client

import (
	"encoding/json"
	"github.com/hectorgimenez/koolo/internal/config"
	"github.com/hectorgimenez/koolo/internal/game"
	"github.com/hectorgimenez/koolo/internal/game/area"
	"github.com/hectorgimenez/koolo/internal/game/difficulty"
	"github.com/hectorgimenez/koolo/internal/game/npc"
	"github.com/hectorgimenez/koolo/internal/game/object"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"strings"
)

func GetMapData(seed string, difficulty difficulty.Difficulty) MapData {
	stdout, err := exec.Command("./koolo-map.exe", config.Config.D2LoDPath, "-s", seed, "-d", getDifficultyAsNum(difficulty)).Output()
	if err != nil {
		panic(err)
	}

	stdoutLines := strings.Split(string(stdout), "\r\n")

	lvls := make([]serverLevel, 0)
	for _, line := range stdoutLines {
		var lvl serverLevel
		err = json.Unmarshal([]byte(line), &lvl)
		// Discard empty lines or lines that don't contain level information
		if err == nil && lvl.Type != "" && len(lvl.Map) > 0 {
			lvls = append(lvls, lvl)
		}
	}

	return lvls
}

func getDifficultyAsNum(df difficulty.Difficulty) string {
	switch df {
	case difficulty.Normal:
		return "0"
	case difficulty.Nightmare:
		return "1"
	case difficulty.Hell:
		return "2"
	}

	return "0"
}

type MapData []serverLevel

func renderCG(cg [][]bool) {
	img := image.NewRGBA(image.Rect(0, 0, len(cg[0]), len(cg)))
	draw.Draw(img, img.Bounds(), img, image.Point{}, draw.Over)

	for y := 0; y < len(cg); y++ {
		for x := 0; x < len(cg[0]); x++ {
			if cg[y][x] {
				img.Set(x, y, color.White)
			} else {
				img.Set(x, y, color.Black)
			}
		}
	}

	outFile, _ := os.Create("cg.png")
	defer outFile.Close()
	png.Encode(outFile, img)
}
func (md MapData) CollisionGrid(area area.Area) [][]bool {
	level := md.getLevel(area)

	var cg [][]bool

	for y := 0; y < level.Size.Height; y++ {
		var row []bool
		for x := 0; x < level.Size.Width; x++ {
			row = append(row, false)
		}

		// Let's do super weird and complicated mappings in the name of "performance" because we love performance
		// but we don't give a fuck about making things easy to read and understand. We came to play.
		if len(level.Map) > y {
			mapRow := level.Map[y]
			isWalkable := false
			xPos := 0
			for k, xs := range mapRow {
				if k != 0 {
					for xOffset := 0; xOffset < xs; xOffset++ {
						row[xPos+xOffset] = isWalkable
					}
				}
				isWalkable = !isWalkable
				xPos += xs
			}
			for xPos < len(row) {
				row[xPos] = isWalkable
				xPos++
			}
		}

		cg = append(cg, row)
	}

	return cg
}

func (md MapData) NPCsExitsAndObjects(areaOrigin game.Position, a area.Area) (game.NPCs, []game.Level, []game.Object, []game.Room) {
	var npcs []game.NPC
	var exits []game.Level
	var objects []game.Object
	var rooms []game.Room

	level := md.getLevel(a)

	for _, r := range level.Rooms {
		rooms = append(rooms, game.Room{
			Position: game.Position{X: r.X,
				Y: r.Y,
			},
			Width:  r.Width,
			Height: r.Height,
		})
	}

	for _, obj := range level.Objects {
		switch obj.Type {
		case "npc":
			n := game.NPC{
				ID:   npc.ID(obj.ID),
				Name: obj.Name,
				Positions: []game.Position{{
					X: obj.X + areaOrigin.X,
					Y: obj.Y + areaOrigin.Y,
				}},
			}
			npcs = append(npcs, n)
		case "exit":
			lvl := game.Level{
				Area: area.Area(obj.ID),
				Position: game.Position{
					X: obj.X + areaOrigin.X,
					Y: obj.Y + areaOrigin.Y,
				},
			}
			exits = append(exits, lvl)
		case "object":
			o := game.Object{
				Name: object.Name(obj.ID),
				Position: game.Position{
					X: obj.X + areaOrigin.X,
					Y: obj.Y + areaOrigin.Y,
				},
			}
			objects = append(objects, o)
		}
	}

	return npcs, exits, objects, rooms
}

func (md MapData) Origin(area area.Area) game.Position {
	level := md.getLevel(area)

	return game.Position{
		X: level.Offset.X,
		Y: level.Offset.Y,
	}
}

func (md MapData) getLevel(area area.Area) serverLevel {
	for _, level := range md {
		if level.ID == int(area) {
			return level
		}
	}

	return serverLevel{}
}
