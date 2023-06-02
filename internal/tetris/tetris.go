// Modified and adapted from github.com/MichaelS11/go-tetris.git
// Under MIT.
package tetris

import (
	"log"
	"os"
)

func Start() error {
	logger = log.New(os.Stderr, "", log.Ldate|log.Ltime|log.LUTC|log.Lshortfile)
	logFile, err := os.OpenFile(".go-tetris.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		logger.Fatal("opening log file error:", err)
	}
	defer logFile.Close()
	logger.SetOutput(logFile)

	err = loadBoards()
	if err != nil {
		return err
	}

	NewView()
	NewMinos()
	NewBoard()
	NewEngine()

	engine.Start()

	view.Stop()
	return nil
}
