package game

import (
	"testing"
)

// TestNewBoard проверяет создание новой доски
func TestNewBoard(t *testing.T) {
	board := NewBoard()

	if board == nil {
		t.Fatal("NewBoard() returned nil")
	}

	if board.CurrentTurn != White {
		t.Errorf("Expected CurrentTurn to be White, got %v", board.CurrentTurn)
	}

	if board.GameOver {
		t.Error("Expected GameOver to be false")
	}
}

// TestInitialPosition проверяет начальную расстановку фигур
func TestInitialPosition(t *testing.T) {
	board := NewBoard()

	// Проверяем белых пешек
	for col := 0; col < 8; col++ {
		piece := board.Cells[6][col]
		if piece.Type != Pawn || piece.Color != White {
			t.Errorf("Expected white pawn at position (6, %d), got %v", col, piece)
		}
	}

	// Проверяем черных пешек
	for col := 0; col < 8; col++ {
		piece := board.Cells[1][col]
		if piece.Type != Pawn || piece.Color != Black {
			t.Errorf("Expected black pawn at position (1, %d), got %v", col, piece)
		}
	}

	// Проверяем белого короля
	king := board.Cells[7][4]
	if king.Type != King || king.Color != White {
		t.Errorf("Expected white king at e1, got %v", king)
	}

	// Проверяем черного короля
	king = board.Cells[0][4]
	if king.Type != King || king.Color != Black {
		t.Errorf("Expected black king at e8, got %v", king)
	}

	// Проверяем пустые клетки
	for row := 2; row < 6; row++ {
		for col := 0; col < 8; col++ {
			piece := board.Cells[row][col]
			if piece.Type != Empty {
				t.Errorf("Expected empty cell at (%d, %d), got %v", row, col, piece)
			}
		}
	}
}

// TestValidMoves проверяет валидные ходы
func TestValidMoves(t *testing.T) {
	board := NewBoard()

	testCases := []struct {
		name  string
		move  Move
		valid bool
	}{
		{
			name:  "Pawn e2-e4",
			move:  Move{From: Position{6, 4}, To: Position{4, 4}},
			valid: true,
		},
		{
			name:  "Pawn e2-e3",
			move:  Move{From: Position{6, 4}, To: Position{5, 4}},
			valid: true,
		},
		{
			name:  "Knight b1-c3",
			move:  Move{From: Position{7, 1}, To: Position{5, 2}},
			valid: true,
		},
		{
			name:  "Invalid move - pawn backwards",
			move:  Move{From: Position{6, 4}, To: Position{7, 4}},
			valid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := board.IsValidMove(tc.move)
			if result != tc.valid {
				t.Errorf("IsValidMove(%v) = %v, expected %v", tc.move, result, tc.valid)
			}
		})
	}
}

// TestMakeMove проверяет выполнение хода
func TestMakeMove(t *testing.T) {
	board := NewBoard()

	// Ход белой пешкой e2-e4
	move := Move{From: Position{6, 4}, To: Position{4, 4}}
	board.MakeMove(move)

	// Проверяем, что пешка переместилась
	if board.Cells[4][4].Type != Pawn || board.Cells[4][4].Color != White {
		t.Error("Pawn should be at e4")
	}

	// Проверяем, что старая позиция пуста
	if board.Cells[6][4].Type != Empty {
		t.Error("e2 should be empty")
	}

	// Проверяем, что очередь хода изменилась
	if board.CurrentTurn != Black {
		t.Error("Turn should be Black's")
	}

	// Проверяем счетчик ходов
	if board.MovesCount != 1 {
		t.Errorf("MovesCount should be 1, got %d", board.MovesCount)
	}
}

// TestGetLegalMoves проверяет получение всех возможных ходов
func TestGetLegalMoves(t *testing.T) {
	board := NewBoard()

	// Получаем все возможные ходы для белых
	moves := board.GetLegalMoves()

	if len(moves) == 0 {
		t.Error("Expected at least some valid moves for white at start")
	}

	// В начальной позиции белые имеют 20 возможных ходов:
	// 8 пешек * 2 хода (на 1 или 2 клетки) = 16 ходов
	// 2 коня * 2 хода = 4 хода
	expectedMoves := 20
	if len(moves) != expectedMoves {
		t.Errorf("Expected %d moves for white at start, got %d", expectedMoves, len(moves))
	}
}

// TestClone проверяет клонирование доски
func TestClone(t *testing.T) {
	board := NewBoard()

	// Делаем ход на оригинальной доске
	move := Move{From: Position{6, 4}, To: Position{4, 4}}
	board.MakeMove(move)

	// Клонируем доску
	clone := board.Clone()

	// Проверяем, что клон имеет те же данные
	if clone.CurrentTurn != board.CurrentTurn {
		t.Error("Clone should have same CurrentTurn")
	}

	if clone.MovesCount != board.MovesCount {
		t.Error("Clone should have same MovesCount")
	}

	// Проверяем, что изменение клона не влияет на оригинал
	cloneMove := Move{From: Position{1, 4}, To: Position{3, 4}}
	clone.MakeMove(cloneMove)

	if board.MovesCount == clone.MovesCount {
		t.Error("Clone should be independent of original")
	}
}

// TestPosition проверяет структуру Position
func TestPosition(t *testing.T) {
	pos := Position{Row: 4, Col: 3}

	if pos.Row != 4 {
		t.Errorf("Expected Row to be 4, got %d", pos.Row)
	}

	if pos.Col != 3 {
		t.Errorf("Expected Col to be 3, got %d", pos.Col)
	}
}

// TestPiece проверяет структуру Piece
func TestPiece(t *testing.T) {
	piece := Piece{Type: Queen, Color: White}

	if piece.Type != Queen {
		t.Errorf("Expected Type to be Queen, got %v", piece.Type)
	}

	if piece.Color != White {
		t.Errorf("Expected Color to be White, got %v", piece.Color)
	}
}
