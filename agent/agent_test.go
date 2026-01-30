package agent

import (
	"chess-ai/game"
	"testing"
)

// TestNewAgent проверяет создание нового агента
func TestNewAgent(t *testing.T) {
	agent := NewAgent(game.Black)

	if agent == nil {
		t.Fatal("NewAgent() returned nil")
	}

	if agent.Network == nil {
		t.Error("Agent's Network should not be nil")
	}

	if agent.Color != game.Black {
		t.Errorf("Expected Color to be Black, got %v", agent.Color)
	}

	if agent.Epsilon != 0.1 {
		t.Errorf("Expected Epsilon to be 0.1, got %f", agent.Epsilon)
	}

	if agent.Gamma != 0.99 {
		t.Errorf("Expected Gamma to be 0.99, got %f", agent.Gamma)
	}
}

// TestChooseMove проверяет выбор хода агентом
func TestChooseMove(t *testing.T) {
	agent := NewAgent(game.Black)
	board := game.NewBoard()

	// Делаем первый ход белыми, чтобы настала очередь черных
	board.MakeMove(game.Move{From: game.Position{6, 4}, To: game.Position{4, 4}})

	move := agent.ChooseMove(board)

	// Проверяем, что ход валиден
	if !board.IsValidMove(move) {
		t.Error("Agent should choose a valid move")
	}
}

// TestRecordState проверяет запись состояния
func TestRecordState(t *testing.T) {
	agent := NewAgent(game.Black)
	board := game.NewBoard()

	initialCount := len(agent.StateHistory)

	agent.RecordState(board)

	if len(agent.StateHistory) != initialCount+1 {
		t.Error("StateHistory should increase after RecordState")
	}

	// Проверяем размер вектора состояния
	state := agent.StateHistory[len(agent.StateHistory)-1]
	if len(state) != 768 {
		t.Errorf("Expected state vector of length 768, got %d", len(state))
	}
}

// TestBoardToVector проверяет преобразование доски в вектор
func TestBoardToVector(t *testing.T) {
	agent := NewAgent(game.Black)
	board := game.NewBoard()

	vector := agent.boardToVector(board)

	// Проверяем размер вектора
	if len(vector) != 768 {
		t.Errorf("Expected vector length 768, got %d", len(vector))
	}

	// Проверяем, что вектор содержит только 0 и 1
	for i, val := range vector {
		if val != 0.0 && val != 1.0 {
			t.Errorf("Vector value at index %d should be 0 or 1, got %f", i, val)
		}
	}

	// Подсчитываем количество единиц (должно быть 32 фигуры на доске)
	count := 0
	for _, val := range vector {
		if val == 1.0 {
			count++
		}
	}

	if count != 32 {
		t.Errorf("Expected 32 pieces on board, got %d", count)
	}
}

// TestLearn проверяет обучение агента
func TestLearn(t *testing.T) {
	agent := NewAgent(game.Black)
	board := game.NewBoard()

	// Записываем несколько состояний
	agent.RecordState(board)
	board.MakeMove(game.Move{From: game.Position{6, 4}, To: game.Position{4, 4}})
	agent.RecordState(board)

	initialEpsilon := agent.Epsilon

	// Обучаем на победе
	agent.Learn(1.0)

	// Epsilon должен уменьшиться
	if agent.Epsilon >= initialEpsilon {
		t.Error("Epsilon should decrease after learning")
	}
}

// TestLearnWithDifferentRewards проверяет обучение с разными наградами
func TestLearnWithDifferentRewards(t *testing.T) {
	testCases := []struct {
		name   string
		reward float64
	}{
		{"Win", 1.0},
		{"Loss", 0.0},
		{"Draw", 0.5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			agent := NewAgent(game.Black)
			board := game.NewBoard()

			agent.RecordState(board)
			agent.Learn(tc.reward)

			// Проверяем, что история очищена не сразу, а epsilon изменился
			if agent.Epsilon >= 0.1 {
				t.Error("Epsilon should change after learning")
			}
		})
	}
}

// TestEvaluatePosition проверяет оценку позиции
func TestEvaluatePosition(t *testing.T) {
	agent := NewAgent(game.Black)
	board := game.NewBoard()

	eval := agent.evaluatePosition(board)

	// Оценка должна быть в диапазоне [-1, 1] из-за tanh
	if eval < -1.0 || eval > 1.0 {
		t.Errorf("Expected evaluation in range [-1, 1], got %f", eval)
	}
}

// TestGetMovesCount проверяет получение количества ходов
func TestGetMovesCount(t *testing.T) {
	agent := NewAgent(game.Black)

	if agent.GetMovesCount() != 0 {
		t.Error("New agent should have 0 moves")
	}

	board := game.NewBoard()
	agent.RecordState(board)
	agent.RecordState(board)

	if agent.GetMovesCount() != 2 {
		t.Errorf("Expected 2 moves, got %d", agent.GetMovesCount())
	}
}

// TestEpsilonDecay проверяет уменьшение epsilon
func TestEpsilonDecay(t *testing.T) {
	agent := NewAgent(game.Black)
	board := game.NewBoard()

	// Обучаем агента 100 раз
	for i := 0; i < 100; i++ {
		agent.RecordState(board)
		agent.Learn(1.0)
		agent.StateHistory = nil // Очищаем историю для следующей итерации
	}

	// Epsilon должен быть меньше начального значения
	if agent.Epsilon >= 0.1 {
		t.Errorf("Expected epsilon < 0.1 after 100 iterations, got %f", agent.Epsilon)
	}

	// Но не меньше минимума 0.01
	if agent.Epsilon < 0.01 {
		t.Errorf("Epsilon should not go below 0.01, got %f", agent.Epsilon)
	}
}
