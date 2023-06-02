package tetris

import (
	"time"

	"github.com/gdamore/tcell"
)

// NewBoard creates a new clear board
func NewBoard() {
	board = &Board{}
	board.Clear()
}

// ChangeBoardSize changes board size
func ChangeBoardSize(width int, height int) {
	if board.width == width && board.height == height {
		return
	}

	newBoard := &Board{width: width, height: height, boardsIndex: board.boardsIndex}
	newBoard.colors = make([][]tcell.Color, width)
	for i := 0; i < width; i++ {
		newBoard.colors[i] = make([]tcell.Color, height)
		for j := 0; j < height; j++ {
			if i < board.width && j < board.height {
				newBoard.colors[i][j] = board.colors[i][j]
			} else {
				newBoard.colors[i][j] = colorBlank
			}
		}
	}
	newBoard.rotation = make([][]int, width)
	for i := 0; i < width; i++ {
		newBoard.rotation[i] = make([]int, height)
		for j := 0; j < height; j++ {
			if i < board.width && j < board.height {
				newBoard.rotation[i][j] = board.rotation[i][j]
			} else {
				break
			}
		}
	}

	board = newBoard
	board.fullLinesY = make([]bool, board.height)
	board.previewMino = NewMino()
	board.currentMino = NewMino()
}

// Clear clears the board to original state
func (board *Board) Clear() {
	board.width = len(boards[board.boardsIndex].colors)
	board.height = len(boards[board.boardsIndex].colors[0])
	board.colors = make([][]tcell.Color, board.width)
	for i := 0; i < board.width; i++ {
		board.colors[i] = make([]tcell.Color, board.height)
		copy(board.colors[i], boards[board.boardsIndex].colors[i])
	}
	board.rotation = make([][]int, board.width)
	for i := 0; i < board.width; i++ {
		board.rotation[i] = make([]int, board.height)
		copy(board.rotation[i], boards[board.boardsIndex].rotation[i])
	}
	board.fullLinesY = make([]bool, board.height)
	board.previewMino = NewMino()
	board.currentMino = NewMino()
}

// EmptyBoard removes all blocks/colors from the board
func (board *Board) EmptyBoard() {
	for i := 0; i < board.width; i++ {
		for j := 0; j < board.height; j++ {
			board.colors[i][j] = colorBlank
		}
	}
	for i := 0; i < board.width; i++ {
		for j := 0; j < board.height; j++ {
			board.rotation[i][j] = 0
		}
	}
}

// PreviousBoard switches to previous board
func (board *Board) PreviousBoard() {
	board.boardsIndex--
	if board.boardsIndex < 0 {
		board.boardsIndex = len(boards) - 1
	}
	engine.PreviewBoard()
	board.Clear()
}

// NextBoard switches to next board
func (board *Board) NextBoard() {
	board.boardsIndex++
	if board.boardsIndex == len(boards) {
		board.boardsIndex = 0
	}
	engine.PreviewBoard()
	board.Clear()
}

// MinoMoveLeft moves mino left
func (board *Board) MinoMoveLeft() {
	board.dropDistance = 0
	mino := board.currentMino.CloneMoveLeft()
	if mino.ValidLocation(false) {
		board.currentMino = mino
		board.StartLockDelayIfBottom()
	}
}

// MinoMoveRight moves mino right
func (board *Board) MinoMoveRight() {
	board.dropDistance = 0
	mino := board.currentMino.CloneMoveRight()
	if mino.ValidLocation(false) {
		board.currentMino = mino
		board.StartLockDelayIfBottom()
	}
}

// MinoRotateRight rotates mino right
func (board *Board) MinoRotateRight() {
	board.dropDistance = 0
	mino := board.currentMino.CloneRotateRight()
	if mino.ValidLocation(false) {
		board.currentMino = mino
		board.StartLockDelayIfBottom()
		return
	}
	mino.MoveLeft()
	if mino.ValidLocation(false) {
		board.currentMino = mino
		board.StartLockDelayIfBottom()
		return
	}
	mino.MoveRight()
	mino.MoveRight()
	if mino.ValidLocation(false) {
		board.currentMino = mino
		board.StartLockDelayIfBottom()
		return
	}
}

// MinoRotateLeft rotates mino right
func (board *Board) MinoRotateLeft() {
	board.dropDistance = 0
	mino := board.currentMino.CloneRotateLeft()
	if mino.ValidLocation(false) {
		board.currentMino = mino
		board.StartLockDelayIfBottom()
		return
	}
	mino.MoveLeft()
	if mino.ValidLocation(false) {
		board.currentMino = mino
		board.StartLockDelayIfBottom()
		return
	}
	mino.MoveRight()
	mino.MoveRight()
	if mino.ValidLocation(false) {
		board.currentMino = mino
		board.StartLockDelayIfBottom()
		return
	}
}

// MinoMoveDown moves mino down
func (board *Board) MinoMoveDown() {
	mino := board.currentMino.CloneMoveDown()
	if mino.ValidLocation(false) {
		board.dropDistance = 0
		board.currentMino = mino
		if !board.StartLockDelayIfBottom() {
			engine.ResetTimer(0)
		}
		return
	}
	if !board.currentMino.ValidLocation(true) {
		engine.GameOver()
		return
	}
	board.nextMino()
}

// MinoDrop dropps mino
func (board *Board) MinoDrop() {
	board.dropDistance = 0
	mino := board.currentMino.CloneMoveDown()
	for mino.ValidLocation(false) {
		board.dropDistance++
		mino.MoveDown()
	}
	for i := 0; i < board.dropDistance; i++ {
		board.currentMino.MoveDown()
	}
	if !board.currentMino.ValidLocation(true) {
		engine.GameOver()
		return
	}
	if board.dropDistance < 1 {
		return
	}
	if !board.StartLockDelayIfBottom() {
		engine.ResetTimer(0)
	}
}

// StartLockDelayIfBottom if at bottom, starts lock delay
func (board *Board) StartLockDelayIfBottom() bool {
	mino := board.currentMino.CloneMoveDown()
	if mino.ValidLocation(false) {
		return false
	}
	engine.ResetTimer(300 * time.Millisecond)
	return true
}

// nextMino gets next mino
func (board *Board) nextMino() {
	engine.AddScore(board.dropDistance)

	board.currentMino.SetOnBoard()

	board.deleteCheck()

	if !board.previewMino.ValidLocation(false) {
		board.previewMino.MoveUp()
		if !board.previewMino.ValidLocation(false) {
			engine.GameOver()
			return
		}
	}

	board.currentMino = board.previewMino
	board.previewMino = NewMino()
	engine.ResetTimer(0)
}

// deleteCheck checks if there are any lines on the board that can be deleted
func (board *Board) deleteCheck() {
	lines := board.fullLines()
	if len(lines) < 1 {
		return
	}

	view.ShowDeleteAnimation(lines)
	for _, line := range lines {
		board.deleteLine(line)
	}

	engine.AddDeleteLines(len(lines))
}

// fullLines returns the line numbers that have full lines
func (board *Board) fullLines() []int {
	fullLines := make([]int, 0, 1)
	for j := 0; j < board.height; j++ {
		if board.isFullLine(j) {
			fullLines = append(fullLines, j)
		}
	}
	return fullLines
}

// isFullLine checks if line is full
func (board *Board) isFullLine(j int) bool {
	for i := 0; i < board.width; i++ {
		if board.colors[i][j] == colorBlank {
			return false
		}
	}
	return true
}

// deleteLine deletes the line
func (board *Board) deleteLine(line int) {
	for i := 0; i < board.width; i++ {
		board.colors[i][line] = colorBlank
	}
	for j := line; j > 0; j-- {
		for i := 0; i < board.width; i++ {
			board.colors[i][j] = board.colors[i][j-1]
			board.rotation[i][j] = board.rotation[i][j-1]
		}
	}
	for i := 0; i < board.width; i++ {
		board.colors[i][0] = colorBlank
	}
}

// SetColor sets the color and rotation of board location
func (board *Board) SetColor(x int, y int, color tcell.Color, rotation int) {
	board.colors[x][y] = color
	if rotation < 0 {
		return
	}
	board.rotation[x][y] = rotation
}

// RotateLeft rotates cell left
func (board *Board) RotateLeft(x int, y int) {
	if board.rotation[x][y] == 0 {
		board.rotation[x][y] = 3
		return
	}
	board.rotation[x][y]--
}

// RotateRight rotates cell right
func (board *Board) RotateRight(x int, y int) {
	if board.rotation[x][y] == 3 {
		board.rotation[x][y] = 0
		return
	}
	board.rotation[x][y]++
}

// ValidBlockLocation checks if block location is vaild
func (board *Board) ValidBlockLocation(x int, y int, mustBeOnBoard bool) bool {
	if x < 0 || x >= board.width || y >= board.height {
		return false
	}
	if mustBeOnBoard {
		if y < 0 {
			return false
		}
	} else {
		if y < -2 {
			return false
		}
	}
	if y > -1 {
		if board.colors[x][y] != colorBlank {
			return false
		}
	}
	return true
}

// ValidDisplayLocation checks if vaild display location
func ValidDisplayLocation(x int, y int) bool {
	return x >= 0 && x < board.width && y >= 0 && y < board.height
}

// DrawBoard draws the board with help from view
func (board *Board) DrawBoard() {
	for i := 0; i < board.width; i++ {
		for j := 0; j < board.height; j++ {
			if board.colors[i][j] != colorBlank {
				view.DrawBlock(i, j, board.colors[i][j], board.rotation[i][j])
			}
		}
	}
}

// DrawPreviewMino draws the preview mino
func (board *Board) DrawPreviewMino() {
	board.previewMino.DrawMino(MinoPreview)
}

// DrawCurrentMino draws the current mino
func (board *Board) DrawCurrentMino() {
	board.currentMino.DrawMino(MinoCurrent)
}

// DrawDropMino draws the drop mino
func (board *Board) DrawDropMino() {
	mino := board.currentMino.CloneMoveDown()
	if !mino.ValidLocation(false) {
		return
	}
	for mino.ValidLocation(false) {
		mino.MoveDown()
	}
	mino.MoveUp()
}

// DrawCursor draws the edit cursor
func (board *Board) DrawCursor(x int, y int) {
	view.DrawCursor(x, y, board.colors[x][y])
}
