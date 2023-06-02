package tetris

import (
	"log"
	"time"

	"github.com/gdamore/tcell"
)

const (
	boardXOffset  = 4
	boardYOffset  = 2
	aiTickDivider = 8

	// MinoPreview is for the preview mino
	MinoPreview MinoType = iota
	// MinoCurrent is for the current mino
	MinoCurrent = iota

	colorBlank   = tcell.ColorBlack
	colorCyan    = tcell.ColorAqua    // I
	colorBlue    = tcell.ColorBlue    // J
	colorWhite   = tcell.ColorWhite   // L
	colorYellow  = tcell.ColorYellow  // O
	colorGreen   = tcell.ColorLime    // S
	colorMagenta = tcell.ColorFuchsia // T
	colorRed     = tcell.ColorRed     // Z

	engineModeRun engineMode = iota
	engineModeStopped
	engineModeGameOver
	engineModePaused
	engineModePreview
	engineModeEdit
)

type (
	engineMode int

	// MinoType is the type of mino
	MinoType int
	// MinoBlocks is the blocks of the mino
	MinoBlocks [][]tcell.Color
	// MinoRotation is the rotation of the mino
	MinoRotation [4]MinoBlocks

	// Mino is a mino
	Mino struct {
		x            int
		y            int
		length       int
		rotation     int
		minoRotation MinoRotation
	}

	// Minos is a bag of minos
	Minos struct {
		minoBag  [7]MinoRotation
		bagRand  []int
		bagIndex int
	}

	// Board is the Tetris board
	Board struct {
		boardsIndex  int
		width        int
		height       int
		colors       [][]tcell.Color
		rotation     [][]int
		previewMino  *Mino
		currentMino  *Mino
		dropDistance int
		fullLinesY   []bool
	}

	// Boards holds all the boards
	Boards struct {
		name     string
		colors   [][]tcell.Color
		rotation [][]int
	}

	// BoardsJSON is for JSON format of boards
	BoardsJSON struct {
		Name     string
		Mino     [][]string
		Rotation [][]int
	}

	// View is the display engine
	View struct {
	}

	// Ranking holds the ranking scores
	Ranking struct {
		scores []uint64
	}

	// Engine is the Tetirs game engine
	Engine struct {
		stopped      bool
		chanStop     chan struct{}
		chanEventKey chan *tcell.EventKey
		ranking      *Ranking
		timer        *time.Timer
		tickTime     time.Duration
		mode         engineMode
		score        int
		level        int
		deleteLines  int
	}

	// Settings is the JSON load/save file
	Settings struct {
		Boards []BoardsJSON
	}

	// EventGame is an game event
	EventGame struct {
		when time.Time
	}
)

// When returns event when
func (EventGame *EventGame) When() time.Time {
	return EventGame.when
}

var (
	logger *log.Logger
	screen tcell.Screen
	minos  *Minos
	board  *Board
	view   *View
	engine *Engine

	boards            []Boards
	numInternalBoards int
)
