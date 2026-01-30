package agent

import (
	"chess-ai/game"
	"chess-ai/neural"
	"math"
	"math/rand"
)

// Agent представляет RL агента
type Agent struct {
	Network       *neural.Network
	Color         game.Color
	Epsilon       float64 // Вероятность случайного хода
	Gamma         float64 // Коэффициент дисконтирования
	StateHistory  [][]float64
	RewardHistory []float64
}

// NewAgent создает нового агента
func NewAgent(color game.Color) *Agent {
	return &Agent{
		Network: neural.NewNetwork(),
		Color:   color,
		Epsilon: 0.1,
		Gamma:   0.99,
	}
}

// ChooseMove выбирает ход используя epsilon-greedy стратегию
func (a *Agent) ChooseMove(board *game.Board) game.Move {
	moves := board.GetLegalMoves()
	if len(moves) == 0 {
		return game.Move{}
	}

	// Epsilon-greedy: случайный ход с вероятностью epsilon
	if rand.Float64() < a.Epsilon {
		return moves[rand.Intn(len(moves))]
	}

	// Иначе используем minimax с альфа-бета отсечением
	_, bestMove := a.minimax(board, 2, -math.MaxFloat64, math.MaxFloat64, true)
	if bestMove.From.Row == -1 {
		return moves[rand.Intn(len(moves))]
	}

	return bestMove
}

// minimax реализует алгоритм minimax с альфа-бета отсечением
func (a *Agent) minimax(board *game.Board, depth int, alpha, beta float64, maximizing bool) (float64, game.Move) {
	if depth == 0 || board.GameOver {
		return a.evaluatePosition(board), game.Move{From: game.Position{-1, -1}}
	}

	moves := board.GetLegalMoves()
	if len(moves) == 0 {
		return a.evaluatePosition(board), game.Move{From: game.Position{-1, -1}}
	}

	var bestMove game.Move
	bestMove.From = game.Position{-1, -1}

	if maximizing {
		maxEval := -math.MaxFloat64
		for _, move := range moves {
			boardCopy := board.Clone()
			boardCopy.MakeMove(move)
			eval, _ := a.minimax(boardCopy, depth-1, alpha, beta, false)

			if eval > maxEval {
				maxEval = eval
				bestMove = move
			}

			alpha = math.Max(alpha, eval)
			if beta <= alpha {
				break // Альфа-бета отсечение
			}
		}
		return maxEval, bestMove
	} else {
		minEval := math.MaxFloat64
		for _, move := range moves {
			boardCopy := board.Clone()
			boardCopy.MakeMove(move)
			eval, _ := a.minimax(boardCopy, depth-1, alpha, beta, true)

			if eval < minEval {
				minEval = eval
				bestMove = move
			}

			beta = math.Min(beta, eval)
			if beta <= alpha {
				break
			}
		}
		return minEval, bestMove
	}
}

// evaluatePosition оценивает позицию с помощью нейросети
func (a *Agent) evaluatePosition(board *game.Board) float64 {
	input := a.boardToVector(board)
	return a.Network.Forward(input)
}

// boardToVector преобразует доску в вектор (12 битовых плоскостей)
func (a *Agent) boardToVector(board *game.Board) []float64 {
	vector := make([]float64, 768) // 12 плоскостей * 64 клетки

	pieceTypes := []game.PieceType{
		game.Pawn, game.Knight, game.Bishop,
		game.Rook, game.Queen, game.King,
	}

	for planeIdx := 0; planeIdx < 12; planeIdx++ {
		pieceType := pieceTypes[planeIdx%6]
		color := game.White
		if planeIdx >= 6 {
			color = game.Black
		}

		for row := 0; row < 8; row++ {
			for col := 0; col < 8; col++ {
				piece := board.Cells[row][col]
				idx := planeIdx*64 + row*8 + col

				if piece.Type == pieceType && piece.Color == color {
					vector[idx] = 1.0
				} else {
					vector[idx] = 0.0
				}
			}
		}
	}

	return vector
}

// RecordState записывает состояние для последующего обучения
func (a *Agent) RecordState(board *game.Board) {
	state := a.boardToVector(board)
	a.StateHistory = append(a.StateHistory, state)
}

// Learn обучает агента на основе результата игры (TD-Learning)
func (a *Agent) Learn(finalReward float64) {
	if len(a.StateHistory) == 0 {
		return
	}

	// Обратное распространение награды через все состояния
	reward := finalReward
	for i := len(a.StateHistory) - 1; i >= 0; i-- {
		state := a.StateHistory[i]
		a.Network.Train(state, reward)

		// Дисконтирование награды для предыдущих состояний
		reward *= a.Gamma
	}

	// Уменьшаем epsilon (меньше исследования со временем)
	a.Epsilon *= 0.995
	if a.Epsilon < 0.01 {
		a.Epsilon = 0.01
	}
}

// Save сохраняет состояние агента
func (a *Agent) Save() error {
	return a.Network.Save()
}

// Load загружает состояние агента
func (a *Agent) Load() error {
	return a.Network.Load()
}

// GetMovesCount возвращает количество ходов в текущей игре
func (a *Agent) GetMovesCount() int {
	return len(a.StateHistory)
}
