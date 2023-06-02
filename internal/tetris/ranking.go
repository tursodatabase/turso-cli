package tetris

// NewRanking create a new ranking
func NewRanking() *Ranking {
	ranking := &Ranking{
		scores: make([]uint64, 9),
	}

	return ranking
}

// Save saves the rankings to a file
func (ranking *Ranking) Save() {
}

// InsertScore inserts a score into the rankings
func (ranking *Ranking) InsertScore(newScore uint64) {
	for index, score := range ranking.scores {
		if newScore > score {
			ranking.slideScores(index)
			ranking.scores[index] = newScore
			return
		}
	}
}

// slideScores slides the scores down to make room for a new score
func (ranking *Ranking) slideScores(index int) {
	for i := len(ranking.scores) - 1; i > index; i-- {
		ranking.scores[i] = ranking.scores[i-1]
	}
}
