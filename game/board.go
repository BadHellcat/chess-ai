package game

import (
	"fmt"
	"strings"
)

// Цвет фигуры
type Color int

const (
	White Color = iota
	Black
)

// Тип фигуры
type PieceType int

const (
	Empty PieceType = iota
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King
)

// Позиция на доске
type Position struct {
	Row, Col int
}

// Фигура
type Piece struct {
	Type  PieceType
	Color Color
}

// Ход
type Move struct {
	From Position
	To   Position
}

// Доска
type Board struct {
	Cells           [8][8]Piece
	CurrentTurn     Color
	GameOver        bool
	Winner          Color
	EnPassantTarget *Position // Позиция для взятия на проходе
	WhiteKingMoved  bool
	BlackKingMoved  bool
	WhiteRookAMoved bool
	WhiteRookHMoved bool
	BlackRookAMoved bool
	BlackRookHMoved bool
	MovesCount      int
}

// NewBoard создает новую доску с начальной позицией
func NewBoard() *Board {
	board := &Board{
		CurrentTurn: White,
	}
	board.setupInitialPosition()
	return board
}

// setupInitialPosition устанавливает начальную позицию фигур
func (b *Board) setupInitialPosition() {
	// Черные фигуры
	b.Cells[0] = [8]Piece{
		{Rook, Black}, {Knight, Black}, {Bishop, Black}, {Queen, Black},
		{King, Black}, {Bishop, Black}, {Knight, Black}, {Rook, Black},
	}
	for i := 0; i < 8; i++ {
		b.Cells[1][i] = Piece{Pawn, Black}
	}

	// Пустые клетки
	for i := 2; i < 6; i++ {
		for j := 0; j < 8; j++ {
			b.Cells[i][j] = Piece{Empty, White}
		}
	}

	// Белые пешки
	for i := 0; i < 8; i++ {
		b.Cells[6][i] = Piece{Pawn, White}
	}

	// Белые фигуры
	b.Cells[7] = [8]Piece{
		{Rook, White}, {Knight, White}, {Bishop, White}, {Queen, White},
		{King, White}, {Bishop, White}, {Knight, White}, {Rook, White},
	}
}

// IsValidMove проверяет, является ли ход корректным
func (b *Board) IsValidMove(move Move) bool {
	if move.From.Row < 0 || move.From.Row > 7 || move.From.Col < 0 || move.From.Col > 7 {
		return false
	}
	if move.To.Row < 0 || move.To.Row > 7 || move.To.Col < 0 || move.To.Col > 7 {
		return false
	}

	piece := b.Cells[move.From.Row][move.From.Col]
	if piece.Type == Empty {
		return false
	}

	if piece.Color != b.CurrentTurn {
		return false
	}

	target := b.Cells[move.To.Row][move.To.Col]
	if target.Type != Empty && target.Color == piece.Color {
		return false
	}

	// Проверка правил движения для каждой фигуры
	switch piece.Type {
	case Pawn:
		return b.isValidPawnMove(move, piece)
	case Knight:
		return b.isValidKnightMove(move)
	case Bishop:
		return b.isValidBishopMove(move)
	case Rook:
		return b.isValidRookMove(move)
	case Queen:
		return b.isValidQueenMove(move)
	case King:
		return b.isValidKingMove(move, piece)
	}

	return false
}

// isValidPawnMove проверяет ход пешки (включая взятие на проходе)
func (b *Board) isValidPawnMove(move Move, piece Piece) bool {
	direction := -1
	startRow := 6
	if piece.Color == Black {
		direction = 1
		startRow = 1
	}

	rowDiff := move.To.Row - move.From.Row
	colDiff := move.To.Col - move.From.Col

	// Движение вперед на одну клетку
	if colDiff == 0 && rowDiff == direction {
		return b.Cells[move.To.Row][move.To.Col].Type == Empty
	}

	// Движение вперед на две клетки с начальной позиции
	if colDiff == 0 && rowDiff == 2*direction && move.From.Row == startRow {
		middleRow := move.From.Row + direction
		return b.Cells[middleRow][move.From.Col].Type == Empty &&
			b.Cells[move.To.Row][move.To.Col].Type == Empty
	}

	// Взятие по диагонали
	if abs(colDiff) == 1 && rowDiff == direction {
		target := b.Cells[move.To.Row][move.To.Col]
		if target.Type != Empty && target.Color != piece.Color {
			return true
		}

		// Взятие на проходе
		if b.EnPassantTarget != nil &&
			move.To.Row == b.EnPassantTarget.Row &&
			move.To.Col == b.EnPassantTarget.Col {
			return true
		}
	}

	return false
}

// isValidKnightMove проверяет ход коня
func (b *Board) isValidKnightMove(move Move) bool {
	rowDiff := abs(move.To.Row - move.From.Row)
	colDiff := abs(move.To.Col - move.From.Col)
	return (rowDiff == 2 && colDiff == 1) || (rowDiff == 1 && colDiff == 2)
}

// isValidBishopMove проверяет ход слона
func (b *Board) isValidBishopMove(move Move) bool {
	rowDiff := abs(move.To.Row - move.From.Row)
	colDiff := abs(move.To.Col - move.From.Col)

	if rowDiff != colDiff {
		return false
	}

	return b.isPathClear(move)
}

// isValidRookMove проверяет ход ладьи
func (b *Board) isValidRookMove(move Move) bool {
	if move.From.Row != move.To.Row && move.From.Col != move.To.Col {
		return false
	}

	return b.isPathClear(move)
}

// isValidQueenMove проверяет ход ферзя
func (b *Board) isValidQueenMove(move Move) bool {
	return b.isValidBishopMove(move) || b.isValidRookMove(move)
}

// isValidKingMove проверяет ход короля (включая рокировку)
func (b *Board) isValidKingMove(move Move, piece Piece) bool {
	rowDiff := abs(move.To.Row - move.From.Row)
	colDiff := abs(move.To.Col - move.From.Col)

	// Обычный ход короля
	if rowDiff <= 1 && colDiff <= 1 {
		return true
	}

	// Рокировка
	if rowDiff == 0 && colDiff == 2 {
		return b.canCastle(move, piece)
	}

	return false
}

// canCastle проверяет возможность рокировки
func (b *Board) canCastle(move Move, piece Piece) bool {
	// Король не должен был двигаться
	if piece.Color == White && b.WhiteKingMoved {
		return false
	}
	if piece.Color == Black && b.BlackKingMoved {
		return false
	}

	// Проверяем направление рокировки
	isKingSide := move.To.Col > move.From.Col
	
	var rookCol int
	var rookMoved bool
	
	if piece.Color == White {
		if isKingSide {
			rookCol = 7
			rookMoved = b.WhiteRookHMoved
		} else {
			rookCol = 0
			rookMoved = b.WhiteRookAMoved
		}
	} else {
		if isKingSide {
			rookCol = 7
			rookMoved = b.BlackRookHMoved
		} else {
			rookCol = 0
			rookMoved = b.BlackRookAMoved
		}
	}

	// Ладья не должна была двигаться
	if rookMoved {
		return false
	}

	// Проверяем, что ладья на месте
	rook := b.Cells[move.From.Row][rookCol]
	if rook.Type != Rook || rook.Color != piece.Color {
		return false
	}

	// Путь должен быть свободен
	minCol := min(move.From.Col, rookCol)
	maxCol := max(move.From.Col, rookCol)
	for col := minCol + 1; col < maxCol; col++ {
		if b.Cells[move.From.Row][col].Type != Empty {
			return false
		}
	}

	// Король не должен быть под шахом (упрощенная проверка)
	// В полной реализации нужно проверить все три клетки
	return true
}

// isPathClear проверяет, что путь между клетками свободен
func (b *Board) isPathClear(move Move) bool {
	rowStep := 0
	colStep := 0

	if move.To.Row > move.From.Row {
		rowStep = 1
	} else if move.To.Row < move.From.Row {
		rowStep = -1
	}

	if move.To.Col > move.From.Col {
		colStep = 1
	} else if move.To.Col < move.From.Col {
		colStep = -1
	}

	row := move.From.Row + rowStep
	col := move.From.Col + colStep

	for row != move.To.Row || col != move.To.Col {
		if b.Cells[row][col].Type != Empty {
			return false
		}
		row += rowStep
		col += colStep
	}

	return true
}

// MakeMove выполняет ход
func (b *Board) MakeMove(move Move) {
	piece := b.Cells[move.From.Row][move.From.Col]
	
	// Взятие на проходе
	_ = false
	if piece.Type == Pawn && b.EnPassantTarget != nil &&
		move.To.Row == b.EnPassantTarget.Row && move.To.Col == b.EnPassantTarget.Col {
		_ = true
		// Удаляем взятую пешку
		if piece.Color == White {
			b.Cells[move.To.Row+1][move.To.Col] = Piece{Empty, White}
		} else {
			b.Cells[move.To.Row-1][move.To.Col] = Piece{Empty, White}
		}
	}

	// Рокировка
	if piece.Type == King && abs(move.To.Col-move.From.Col) == 2 {
		// Перемещаем ладью
		isKingSide := move.To.Col > move.From.Col
		if isKingSide {
			// Короткая рокировка
			b.Cells[move.From.Row][5] = b.Cells[move.From.Row][7]
			b.Cells[move.From.Row][7] = Piece{Empty, White}
		} else {
			// Длинная рокировка
			b.Cells[move.From.Row][3] = b.Cells[move.From.Row][0]
			b.Cells[move.From.Row][0] = Piece{Empty, White}
		}
	}

	// Обычное перемещение фигуры
	b.Cells[move.To.Row][move.To.Col] = piece
	b.Cells[move.From.Row][move.From.Col] = Piece{Empty, White}

	// Превращение пешки
	if piece.Type == Pawn {
		if (piece.Color == White && move.To.Row == 0) ||
			(piece.Color == Black && move.To.Row == 7) {
			b.Cells[move.To.Row][move.To.Col] = Piece{Queen, piece.Color}
		}
	}

	// Обновляем флаг взятия на проходе
	b.EnPassantTarget = nil
	if piece.Type == Pawn && abs(move.To.Row-move.From.Row) == 2 {
		epRow := (move.From.Row + move.To.Row) / 2
		b.EnPassantTarget = &Position{Row: epRow, Col: move.To.Col}
	}

	// Обновляем флаги движения короля и ладей
	if piece.Type == King {
		if piece.Color == White {
			b.WhiteKingMoved = true
		} else {
			b.BlackKingMoved = true
		}
	}
	if piece.Type == Rook {
		if piece.Color == White {
			if move.From.Col == 0 {
				b.WhiteRookAMoved = true
			} else if move.From.Col == 7 {
				b.WhiteRookHMoved = true
			}
		} else {
			if move.From.Col == 0 {
				b.BlackRookAMoved = true
			} else if move.From.Col == 7 {
				b.BlackRookHMoved = true
			}
		}
	}

	// Меняем ход
	if b.CurrentTurn == White {
		b.CurrentTurn = Black
	} else {
		b.CurrentTurn = White
	}

	b.MovesCount++

	// Проверяем окончание игры
	b.checkGameOver()
}

// GetLegalMoves возвращает список всех легальных ходов для текущего игрока
func (b *Board) GetLegalMoves() []Move {
	moves := []Move{}

	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			piece := b.Cells[row][col]
			if piece.Type == Empty || piece.Color != b.CurrentTurn {
				continue
			}

			from := Position{Row: row, Col: col}

			// Перебираем все возможные позиции
			for toRow := 0; toRow < 8; toRow++ {
				for toCol := 0; toCol < 8; toCol++ {
					to := Position{Row: toRow, Col: toCol}
					move := Move{From: from, To: to}

					if b.IsValidMove(move) {
						moves = append(moves, move)
					}
				}
			}
		}
	}

	return moves
}

// checkGameOver проверяет окончание игры
func (b *Board) checkGameOver() {
	moves := b.GetLegalMoves()
	
	if len(moves) == 0 {
		b.GameOver = true
		// Упрощенная логика: если нет ходов - ничья
		// В полной реализации нужно проверить шах
		b.Winner = White // Ничья обозначается как отсутствие победителя
	}

	// Проверка на ничью по количеству ходов (упрощенная)
	if b.MovesCount > 200 {
		b.GameOver = true
		b.Winner = White // Ничья
	}
}

// Clone создает копию доски
func (b *Board) Clone() *Board {
	clone := &Board{
		Cells:           b.Cells,
		CurrentTurn:     b.CurrentTurn,
		GameOver:        b.GameOver,
		Winner:          b.Winner,
		WhiteKingMoved:  b.WhiteKingMoved,
		BlackKingMoved:  b.BlackKingMoved,
		WhiteRookAMoved: b.WhiteRookAMoved,
		WhiteRookHMoved: b.WhiteRookHMoved,
		BlackRookAMoved: b.BlackRookAMoved,
		BlackRookHMoved: b.BlackRookHMoved,
		MovesCount:      b.MovesCount,
	}

	if b.EnPassantTarget != nil {
		clone.EnPassantTarget = &Position{
			Row: b.EnPassantTarget.Row,
			Col: b.EnPassantTarget.Col,
		}
	}

	return clone
}

// String возвращает строковое представление доски
func (b *Board) String() string {
	var sb strings.Builder

	sb.WriteString("\n  a b c d e f g h\n")
	for row := 0; row < 8; row++ {
		sb.WriteString(fmt.Sprintf("%d ", 8-row))
		for col := 0; col < 8; col++ {
			piece := b.Cells[row][col]
			sb.WriteString(pieceToString(piece))
			sb.WriteString(" ")
		}
		sb.WriteString(fmt.Sprintf("%d\n", 8-row))
	}
	sb.WriteString("  a b c d e f g h\n\n")

	return sb.String()
}

// pieceToString конвертирует фигуру в строку
func pieceToString(piece Piece) string {
	if piece.Type == Empty {
		return "."
	}

	symbols := map[PieceType]string{
		Pawn:   "P",
		Knight: "N",
		Bishop: "B",
		Rook:   "R",
		Queen:  "Q",
		King:   "K",
	}

	symbol := symbols[piece.Type]
	if piece.Color == Black {
		return strings.ToLower(symbol)
	}
	return symbol
}

// Вспомогательные функции
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
