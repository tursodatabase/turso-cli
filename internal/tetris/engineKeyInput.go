package tetris

import (
	"runtime"

	"github.com/gdamore/tcell"
)

// ProcessEventKey process the key input event
func (engine *Engine) ProcessEventKey(eventKey *tcell.EventKey) {
	if eventKey.Key() == tcell.KeyCtrlL {
		// Ctrl l (lower case L) to log stack trace
		buffer := make([]byte, 1<<16)
		length := runtime.Stack(buffer, true)
		logger.Println("Stack trace")
		logger.Println(string(buffer[:length]))
		return
	}

	switch engine.mode {

	// game over
	case engineModeGameOver, engineModePreview:

		switch eventKey.Key() {
		case tcell.KeyCtrlC:
			engine.Stop()
		case tcell.KeyRune:
			switch eventKey.Rune() {
			case 'q':
				engine.Stop()
			case ' ':
				engine.NewGame()
			}
		}

	// paused
	case engineModePaused:

		switch eventKey.Rune() {
		case 'q':
			engine.Stop()
		case 'p':
			engine.UnPause()
		}

	// run
	case engineModeRun:

		switch eventKey.Key() {
		case tcell.KeyUp:
			board.MinoDrop()
		case tcell.KeyDown:
			board.MinoMoveDown()
		case tcell.KeyLeft:
			board.MinoMoveLeft()
		case tcell.KeyRight:
			board.MinoMoveRight()
		case tcell.KeyRune:
			switch eventKey.Rune() {
			case 'q':
				engine.Stop()
			case ' ':
				board.MinoDrop()
			case 'z':
				board.MinoRotateLeft()
			case 'x':
				board.MinoRotateRight()
			case 'p':
				engine.Pause()
			}
		}
	}

}
