package tetris

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/gdamore/tcell"
)

// NewView creates a new view
func NewView() {
	var err error

	screen, err = tcell.NewScreen()
	if err != nil {
		logger.Fatal("NewScreen error:", err)
	}
	err = screen.Init()
	if err != nil {
		logger.Fatal("screen Init error:", err)
	}

	screen.Clear()

	view = &View{}
}

// Stop stops the view
func (view *View) Stop() {
	logger.Println("View Stop start")

	screen.Fini()

	logger.Println("View Stop end")
}

// RefreshScreen refreshes the updated view to the screen
func (view *View) RefreshScreen() {

	switch engine.mode {

	case engineModeRun:
		view.drawBoardBoarder()
		view.drawPreviewBoarder()
		view.drawTexts()
		board.DrawBoard()
		board.DrawPreviewMino()
		board.DrawDropMino()
		board.DrawCurrentMino()
		screen.Show()

	case engineModePaused:
		screen.Fill(' ', tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorBlack))
		view.drawBoardBoarder()
		view.drawPreviewBoarder()
		view.drawTexts()
		view.drawPaused()
		screen.Show()

	case engineModeGameOver:
		view.drawBoardBoarder()
		view.drawPreviewBoarder()
		view.drawTexts()
		view.drawGameOver()
		view.drawRankingScores()
		screen.Show()

	case engineModePreview:
		screen.Fill(' ', tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorBlack))
		view.drawBoardBoarder()
		view.drawPreviewBoarder()
		view.drawTexts()
		board.DrawBoard()
		view.drawGameOver()
		screen.Show()
	}

}

// drawBoardBoarder draws the board boarder
func (view *View) drawBoardBoarder() {
	xOffset := boardXOffset
	yOffset := boardYOffset
	xEnd := boardXOffset + board.width*2 + 4
	yEnd := boardYOffset + board.height + 2
	styleBoarder := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorLightGray)
	styleBoard := tcell.StyleDefault.Foreground(tcell.ColorLightGray).Background(tcell.ColorBlack)
	for x := xOffset; x < xEnd; x++ {
		for y := yOffset; y < yEnd; y++ {
			if x == xOffset || x == xOffset+1 || x == xEnd-1 || x == xEnd-2 || y == yOffset || y == yEnd-1 {
				screen.SetContent(x, y, ' ', nil, styleBoarder)
			} else {
				screen.SetContent(x, y, ' ', nil, styleBoard)
			}
		}
	}
}

// drawPreviewBoarder draws the preview boarder
func (view *View) drawPreviewBoarder() {
	xOffset := boardXOffset + board.width*2 + 8
	yOffset := boardYOffset
	xEnd := xOffset + 14
	yEnd := yOffset + 6
	styleBoarder := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorLightGray)
	styleBoard := tcell.StyleDefault.Foreground(tcell.ColorLightGray).Background(tcell.ColorBlack)
	for x := xOffset; x < xEnd; x++ {
		for y := yOffset; y < yEnd; y++ {
			if x == xOffset || x == xOffset+1 || x == xEnd-1 || x == xEnd-2 || y == yOffset || y == yEnd-1 {
				screen.SetContent(x, y, ' ', nil, styleBoarder)
			} else {
				screen.SetContent(x, y, ' ', nil, styleBoard)
			}
		}
	}

}

// drawTexts draws the text
func (view *View) drawTexts() {
	xOffset := boardXOffset + board.width*2 + 8
	yOffset := boardYOffset + 7

	view.drawText(xOffset, yOffset, "SCORE:", tcell.ColorLightGray, tcell.ColorDarkBlue)
	view.drawText(xOffset+7, yOffset, fmt.Sprintf("%7d", engine.score), tcell.ColorBlack, tcell.ColorLightGray)

	yOffset += 2

	view.drawText(xOffset, yOffset, "LINES:", tcell.ColorLightGray, tcell.ColorDarkBlue)
	view.drawText(xOffset+7, yOffset, fmt.Sprintf("%7d", engine.deleteLines), tcell.ColorBlack, tcell.ColorLightGray)

	yOffset += 2

	view.drawText(xOffset, yOffset, "LEVEL:", tcell.ColorLightGray, tcell.ColorDarkBlue)
	view.drawText(xOffset+7, yOffset, fmt.Sprintf("%4d", engine.level), tcell.ColorBlack, tcell.ColorLightGray)

	yOffset += 2

	// ascii arrow characters add extra two spaces
	view.drawText(xOffset, yOffset, "←  - left", tcell.ColorLightGray, tcell.ColorBlack)
	yOffset++
	view.drawText(xOffset, yOffset, "→  - right", tcell.ColorLightGray, tcell.ColorBlack)
	yOffset++
	view.drawText(xOffset, yOffset, "↓  - soft drop", tcell.ColorLightGray, tcell.ColorBlack)
	yOffset++
	view.drawText(xOffset, yOffset, "↑  - hard drop", tcell.ColorLightGray, tcell.ColorBlack)
	yOffset++
	view.drawText(xOffset, yOffset, "sbar - hard drop", tcell.ColorLightGray, tcell.ColorBlack)
	yOffset++
	view.drawText(xOffset, yOffset, "z    - rotate left", tcell.ColorLightGray, tcell.ColorBlack)
	yOffset++
	view.drawText(xOffset, yOffset, "x    - rotate right", tcell.ColorLightGray, tcell.ColorBlack)
	yOffset++
	view.drawText(xOffset, yOffset, "p    - pause", tcell.ColorLightGray, tcell.ColorBlack)
	yOffset++
	view.drawText(xOffset, yOffset, "q    - quit", tcell.ColorLightGray, tcell.ColorBlack)
}

// DrawPreviewMinoBlock draws the preview mino
func (view *View) DrawPreviewMinoBlock(x int, y int, color tcell.Color, rotation int, length int) {
	char1 := '█'
	char2 := '█'
	switch rotation {
	case 0:
		char1 = '▄'
		char2 = '▄'
	case 1:
		char2 = ' '
	case 2:
		char1 = '▀'
		char2 = '▀'
	case 3:
		char1 = ' '
	}
	xOffset := 2*x + 2*board.width + boardXOffset + 11 + (4 - length)
	style := tcell.StyleDefault.Foreground(color).Background(color).Dim(true)
	screen.SetContent(xOffset, y+boardYOffset+2, char1, nil, style)
	screen.SetContent(xOffset+1, y+boardYOffset+2, char2, nil, style)
}

// DrawBlock draws a block
func (view *View) DrawBlock(x int, y int, color tcell.Color, rotation int) {
	char1 := '█'
	char2 := '█'
	switch rotation {
	case 0:
		char1 = '▄'
		char2 = '▄'
	case 1:
		char2 = ' '
	case 2:
		char1 = '▀'
		char2 = '▀'
	case 3:
		char1 = ' '
	}
	if color == colorBlank {
		// colorBlank means drop Mino
		style := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorSilver).Bold(true)
		screen.SetContent(2*x+boardXOffset+2, y+boardYOffset+1, char1, nil, style)
		screen.SetContent(2*x+boardXOffset+3, y+boardYOffset+1, char2, nil, style)
	} else {
		style := tcell.StyleDefault.Foreground(color).Background(color).Dim(true)
		screen.SetContent(2*x+boardXOffset+2, y+boardYOffset+1, char1, nil, style)
		screen.SetContent(2*x+boardXOffset+3, y+boardYOffset+1, char2, nil, style)
	}
}

// drawPaused draws Paused
func (view *View) drawPaused() {
	yOffset := (board.height+1)/2 + boardYOffset
	view.drawTextCenter(yOffset, "Paused", tcell.ColorWhite, tcell.ColorBlack)
}

// drawGameOver draws GAME OVER
func (view *View) drawGameOver() {
	yOffset := boardYOffset + 2
	view.drawTextCenter(yOffset, " GAME OVER", tcell.ColorWhite, tcell.ColorBlack)
	yOffset += 2
	view.drawTextCenter(yOffset, "sbar for new game", tcell.ColorWhite, tcell.ColorBlack)

	if engine.mode == engineModePreview {
		return
	}
}

// drawRankingScores draws the ranking scores
func (view *View) drawRankingScores() {
	yOffset := boardYOffset + 10
	for index, line := range engine.ranking.scores {
		view.drawTextCenter(yOffset+index, fmt.Sprintf("%1d: %6d", index+1, line), tcell.ColorWhite, tcell.ColorBlack)
	}
}

// drawText draws the provided text
func (view *View) drawText(x int, y int, text string, fg tcell.Color, bg tcell.Color) {
	style := tcell.StyleDefault.Foreground(fg).Background(bg)
	for index, char := range text {
		screen.SetContent(x+index, y, rune(char), nil, style)
	}
}

// drawTextCenter draws text in the center of the board
func (view *View) drawTextCenter(y int, text string, fg tcell.Color, bg tcell.Color) {
	xOffset := board.width - (len(text)+1)/2 + boardXOffset + 2
	style := tcell.StyleDefault.Foreground(fg).Background(bg)
	for index, char := range text {
		screen.SetContent(index+xOffset, y, rune(char), nil, style)
	}
}

// ShowDeleteAnimation draws the delete animation
func (view *View) ShowDeleteAnimation(lines []int) {
	view.RefreshScreen()

	for times := 0; times < 3; times++ {
		for _, y := range lines {
			view.colorizeLine(y, tcell.ColorLightGray)
		}
		screen.Show()
		time.Sleep(140 * time.Millisecond)

		view.RefreshScreen()
		time.Sleep(140 * time.Millisecond)
	}
}

// ShowGameOverAnimation draws one randomily picked gave over animation
func (view *View) ShowGameOverAnimation() {
	logger.Println("View ShowGameOverAnimation start")

	switch rand.Intn(3) {
	case 0:
		logger.Println("View ShowGameOverAnimation case 0")
		for y := board.height - 1; y >= 0; y-- {
			view.colorizeLine(y, tcell.ColorLightGray)
			screen.Show()
			time.Sleep(60 * time.Millisecond)
		}

	case 1:
		logger.Println("View ShowGameOverAnimation case 1")
		for y := 0; y < board.height; y++ {
			view.colorizeLine(y, tcell.ColorLightGray)
			screen.Show()
			time.Sleep(60 * time.Millisecond)
		}

	case 2:
		logger.Println("View ShowGameOverAnimation case 2")
		sleepTime := 50 * time.Millisecond
		topStartX := boardXOffset + 3
		topEndX := board.width*2 + boardXOffset + 1
		topY := boardYOffset + 1
		rightStartY := boardYOffset + 1
		rightEndY := board.height + boardYOffset + 1
		rightX := board.width*2 + boardXOffset + 1
		bottomStartX := topEndX - 1
		bottomEndX := topStartX - 1
		bottomY := board.height + boardYOffset
		leftStartY := rightEndY - 1
		leftEndY := rightStartY - 1
		leftX := boardXOffset + 2
		style := tcell.StyleDefault.Foreground(tcell.ColorLightGray).Background(tcell.ColorLightGray)

		for topStartX <= topEndX && rightStartY <= rightEndY {
			for x := topStartX; x < topEndX; x++ {
				screen.SetContent(x, topY, ' ', nil, style)
			}
			topStartX++
			topEndX--
			topY++
			for y := rightStartY; y < rightEndY; y++ {
				screen.SetContent(rightX, y, ' ', nil, style)
			}
			rightStartY++
			rightEndY--
			rightX--
			for x := bottomStartX; x > bottomEndX; x-- {
				screen.SetContent(x, bottomY, ' ', nil, style)
			}
			bottomStartX--
			bottomEndX++
			bottomY--
			for y := leftStartY; y > leftEndY; y-- {
				screen.SetContent(leftX, y, ' ', nil, style)
			}
			leftStartY--
			leftEndY++
			leftX++
			screen.Show()
			time.Sleep(sleepTime)
			sleepTime += 4 * time.Millisecond
		}
	}

	logger.Println("View ShowGameOverAnimation end")
}

// colorizeLine changes the color of a line
func (view *View) colorizeLine(y int, color tcell.Color) {
	style := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(color)
	for x := 0; x < board.width; x++ {
		screen.SetContent(x*2+boardXOffset+2, y+boardYOffset+1, ' ', nil, style)
		screen.SetContent(x*2+boardXOffset+3, y+boardYOffset+1, ' ', nil, style)
	}
}

// DrawCursor draws current cursor location
func (view *View) DrawCursor(x int, y int, color tcell.Color) {
	style := tcell.StyleDefault.Foreground(color).Background(tcell.ColorBlack)
	if color == colorBlank {
		style = tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorLightGrey)
	}
	screen.SetContent(x*2+boardXOffset+2, y+boardYOffset+1, '◄', nil, style)
	screen.SetContent(x*2+boardXOffset+3, y+boardYOffset+1, '►', nil, style)
	screen.Show()
}
