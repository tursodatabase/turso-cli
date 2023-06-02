package tetris

import (
	"math/rand"
)

// NewMino creates a new Mino
func NewMino() *Mino {
	minoRotation := minos.minoBag[minos.bagRand[minos.bagIndex]]
	minos.bagIndex++
	if minos.bagIndex > 6 {
		minos.bagIndex = 0
		minos.bagRand = rand.Perm(7)
	}
	mino := &Mino{
		minoRotation: minoRotation,
		length:       len(minoRotation[0]),
	}
	mino.x = board.width/2 - (mino.length+1)/2
	mino.y = -1
	return mino
}

// CloneMoveLeft creates copy of the mino and moves it left
func (mino *Mino) CloneMoveLeft() *Mino {
	newMino := *mino
	newMino.MoveLeft()
	return &newMino
}

// MoveLeft moves the mino left
func (mino *Mino) MoveLeft() {
	mino.x--
}

// CloneMoveRight creates copy of the mino and moves it right
func (mino *Mino) CloneMoveRight() *Mino {
	newMino := *mino
	newMino.MoveRight()
	return &newMino
}

// MoveRight moves the mino right
func (mino *Mino) MoveRight() {
	mino.x++
}

// CloneRotateRight creates copy of the mino and rotates it right
func (mino *Mino) CloneRotateRight() *Mino {
	newMino := *mino
	newMino.RotateRight()
	return &newMino
}

// RotateRight rotates the mino right
func (mino *Mino) RotateRight() {
	mino.rotation++
	if mino.rotation > 3 {
		mino.rotation = 0
	}
}

// CloneRotateLeft creates copy of the mino and rotates it left
func (mino *Mino) CloneRotateLeft() *Mino {
	newMino := *mino
	newMino.RotateLeft()
	return &newMino
}

// RotateLeft rotates the mino left
func (mino *Mino) RotateLeft() {
	if mino.rotation < 1 {
		mino.rotation = 3
		return
	}
	mino.rotation--
}

// CloneMoveDown creates copy of the mino and moves it down
func (mino *Mino) CloneMoveDown() *Mino {
	newMino := *mino
	newMino.MoveDown()
	return &newMino
}

// MoveDown moves the mino down
func (mino *Mino) MoveDown() {
	mino.y++
}

// MoveUp moves the mino up
func (mino *Mino) MoveUp() {
	mino.y--
}

// ValidLocation check if the mino is in a valid location
func (mino *Mino) ValidLocation(mustBeOnBoard bool) bool {
	minoBlocks := mino.minoRotation[mino.rotation]
	for i := 0; i < mino.length; i++ {
		for j := 0; j < mino.length; j++ {
			if minoBlocks[i][j] == colorBlank {
				continue
			}
			if !board.ValidBlockLocation(mino.x+i, mino.y+j, mustBeOnBoard) {
				return false
			}
		}
	}
	return true
}

// SetOnBoard attaches mino to the board
func (mino *Mino) SetOnBoard() {
	minoBlocks := mino.minoRotation[mino.rotation]
	for i := 0; i < mino.length; i++ {
		for j := 0; j < mino.length; j++ {
			if minoBlocks[i][j] != colorBlank {
				board.SetColor(mino.x+i, mino.y+j, minoBlocks[i][j], mino.rotation)
			}
		}
	}
}

// DrawMino draws the mino on the board
func (mino *Mino) DrawMino(minoType MinoType) {
	minoBlocks := mino.minoRotation[mino.rotation]
	for i := 0; i < mino.length; i++ {
		for j := 0; j < mino.length; j++ {
			if minoBlocks[i][j] != colorBlank {
				switch minoType {
				case MinoPreview:
					view.DrawPreviewMinoBlock(i, j, minoBlocks[i][j], mino.rotation, mino.length)
				case MinoCurrent:
					if ValidDisplayLocation(mino.x+i, mino.y+j) {
						view.DrawBlock(mino.x+i, mino.y+j, minoBlocks[i][j], mino.rotation)
					}
				}
			}
		}
	}
}
