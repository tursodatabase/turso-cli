package tetris

import (
	"math/rand"

	"github.com/gdamore/tcell"
)

// NewMinos creates the minos and minoBag
func NewMinos() {
	minoI := MinoBlocks{
		[]tcell.Color{colorBlank, colorCyan, colorBlank, colorBlank},
		[]tcell.Color{colorBlank, colorCyan, colorBlank, colorBlank},
		[]tcell.Color{colorBlank, colorCyan, colorBlank, colorBlank},
		[]tcell.Color{colorBlank, colorCyan, colorBlank, colorBlank},
	}
	minoJ := MinoBlocks{
		[]tcell.Color{colorBlue, colorBlue, colorBlank},
		[]tcell.Color{colorBlank, colorBlue, colorBlank},
		[]tcell.Color{colorBlank, colorBlue, colorBlank},
	}
	minoL := MinoBlocks{
		[]tcell.Color{colorBlank, colorWhite, colorBlank},
		[]tcell.Color{colorBlank, colorWhite, colorBlank},
		[]tcell.Color{colorWhite, colorWhite, colorBlank},
	}
	minoO := MinoBlocks{
		[]tcell.Color{colorYellow, colorYellow},
		[]tcell.Color{colorYellow, colorYellow},
	}
	minoS := MinoBlocks{
		[]tcell.Color{colorBlank, colorGreen, colorBlank},
		[]tcell.Color{colorGreen, colorGreen, colorBlank},
		[]tcell.Color{colorGreen, colorBlank, colorBlank},
	}
	minoT := MinoBlocks{
		[]tcell.Color{colorBlank, colorMagenta, colorBlank},
		[]tcell.Color{colorMagenta, colorMagenta, colorBlank},
		[]tcell.Color{colorBlank, colorMagenta, colorBlank},
	}
	minoZ := MinoBlocks{
		[]tcell.Color{colorRed, colorBlank, colorBlank},
		[]tcell.Color{colorRed, colorRed, colorBlank},
		[]tcell.Color{colorBlank, colorRed, colorBlank},
	}

	var minoRotationI MinoRotation
	minoRotationI[0] = minoI
	for i := 1; i < 4; i++ {
		minoRotationI[i] = minosCloneRotateRight(minoRotationI[i-1])
	}
	var minoRotationJ MinoRotation
	minoRotationJ[0] = minoJ
	for i := 1; i < 4; i++ {
		minoRotationJ[i] = minosCloneRotateRight(minoRotationJ[i-1])
	}
	var minoRotationL MinoRotation
	minoRotationL[0] = minoL
	for i := 1; i < 4; i++ {
		minoRotationL[i] = minosCloneRotateRight(minoRotationL[i-1])
	}
	var minoRotationO MinoRotation
	minoRotationO[0] = minoO
	minoRotationO[1] = minoO
	minoRotationO[2] = minoO
	minoRotationO[3] = minoO
	var minoRotationS MinoRotation
	minoRotationS[0] = minoS
	for i := 1; i < 4; i++ {
		minoRotationS[i] = minosCloneRotateRight(minoRotationS[i-1])
	}
	var minoRotationT MinoRotation
	minoRotationT[0] = minoT
	for i := 1; i < 4; i++ {
		minoRotationT[i] = minosCloneRotateRight(minoRotationT[i-1])
	}
	var minoRotationZ MinoRotation
	minoRotationZ[0] = minoZ
	for i := 1; i < 4; i++ {
		minoRotationZ[i] = minosCloneRotateRight(minoRotationZ[i-1])
	}

	minos = &Minos{
		minoBag: [7]MinoRotation{minoRotationI, minoRotationJ, minoRotationL, minoRotationO, minoRotationS, minoRotationT, minoRotationZ},
		bagRand: rand.Perm(7),
	}
}

// minosCloneRotateRight clones a mino and rotates the mino to the right
func minosCloneRotateRight(minoBlocks MinoBlocks) MinoBlocks {
	length := len(minoBlocks)
	newMinoBlocks := make(MinoBlocks, length)
	for i := 0; i < length; i++ {
		newMinoBlocks[i] = make([]tcell.Color, length)
	}

	for i := 0; i < length; i++ {
		for j := 0; j < length; j++ {
			newMinoBlocks[length-j-1][i] = minoBlocks[i][j]
		}
	}

	return newMinoBlocks
}
