package database

import (
	"chess-ai/game"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database представляет соединение с базой данных
type Database struct {
	db *sql.DB
}

// MoveRecord представляет запись о ходе в базе данных
type MoveRecord struct {
	ID           int64
	GameID       int64
	MoveNumber   int
	FromRow      int
	FromCol      int
	ToRow        int
	ToCol        int
	Evaluation   float64
	Result       string // "win", "loss", "draw", "ongoing"
	BoardHash    string
	CreatedAt    time.Time
}

// PositionStats представляет статистику позиции
type PositionStats struct {
	BoardHash    string
	TotalGames   int
	Wins         int
	Losses       int
	Draws        int
	AvgEval      float64
	BestMove     *game.Move
	BestMoveEval float64
}

// NewDatabase создает новое подключение к базе данных
func NewDatabase(dbPath string) (*Database, error) {
	// Создаем директорию если не существует
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("не удалось создать директорию: %v", err)
	}

	// Открываем соединение
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть базу данных: %v", err)
	}

	database := &Database{db: db}

	// Создаем таблицы
	if err := database.createTables(); err != nil {
		db.Close()
		return nil, err
	}

	return database, nil
}

// createTables создает необходимые таблицы
func (d *Database) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS games (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		finished_at TIMESTAMP,
		winner TEXT,
		moves_count INTEGER,
		white_epsilon FLOAT,
		black_epsilon FLOAT
	);

	CREATE TABLE IF NOT EXISTS moves (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		game_id INTEGER NOT NULL,
		move_number INTEGER NOT NULL,
		from_row INTEGER NOT NULL,
		from_col INTEGER NOT NULL,
		to_row INTEGER NOT NULL,
		to_col INTEGER NOT NULL,
		evaluation FLOAT,
		board_hash TEXT,
		result TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (game_id) REFERENCES games(id)
	);

	CREATE INDEX IF NOT EXISTS idx_moves_game_id ON moves(game_id);
	CREATE INDEX IF NOT EXISTS idx_moves_board_hash ON moves(board_hash);
	CREATE INDEX IF NOT EXISTS idx_moves_result ON moves(result);
	`

	_, err := d.db.Exec(schema)
	return err
}

// StartGame создает новую игру в базе данных
func (d *Database) StartGame(whiteEpsilon, blackEpsilon float64) (int64, error) {
	result, err := d.db.Exec(
		"INSERT INTO games (white_epsilon, black_epsilon) VALUES (?, ?)",
		whiteEpsilon, blackEpsilon,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// FinishGame обновляет информацию о завершенной игре
func (d *Database) FinishGame(gameID int64, winner string, movesCount int) error {
	_, err := d.db.Exec(
		"UPDATE games SET finished_at = CURRENT_TIMESTAMP, winner = ?, moves_count = ? WHERE id = ?",
		winner, movesCount, gameID,
	)
	return err
}

// RecordMove записывает ход в базу данных
func (d *Database) RecordMove(record MoveRecord) error {
	_, err := d.db.Exec(`
		INSERT INTO moves (game_id, move_number, from_row, from_col, to_row, to_col, evaluation, board_hash, result)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.GameID, record.MoveNumber, record.FromRow, record.FromCol,
		record.ToRow, record.ToCol, record.Evaluation, record.BoardHash, record.Result,
	)
	return err
}

// GetPositionStats возвращает статистику для данной позиции
func (d *Database) GetPositionStats(boardHash string) (*PositionStats, error) {
	stats := &PositionStats{BoardHash: boardHash}

	// Получаем общую статистику
	err := d.db.QueryRow(`
		SELECT 
			COUNT(DISTINCT game_id) as total_games,
			SUM(CASE WHEN result = 'win' THEN 1 ELSE 0 END) as wins,
			SUM(CASE WHEN result = 'loss' THEN 1 ELSE 0 END) as losses,
			SUM(CASE WHEN result = 'draw' THEN 1 ELSE 0 END) as draws,
			AVG(evaluation) as avg_eval
		FROM moves
		WHERE board_hash = ?
	`, boardHash).Scan(&stats.TotalGames, &stats.Wins, &stats.Losses, &stats.Draws, &stats.AvgEval)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Получаем лучший ход для этой позиции
	var fromRow, fromCol, toRow, toCol int
	var bestEval float64
	err = d.db.QueryRow(`
		SELECT from_row, from_col, to_row, to_col, evaluation
		FROM moves
		WHERE board_hash = ? AND result = 'win'
		ORDER BY evaluation DESC
		LIMIT 1
	`, boardHash).Scan(&fromRow, &fromCol, &toRow, &toCol, &bestEval)

	if err == nil {
		stats.BestMove = &game.Move{
			From: game.Position{Row: fromRow, Col: fromCol},
			To:   game.Position{Row: toRow, Col: toCol},
		}
		stats.BestMoveEval = bestEval
	}

	return stats, nil
}

// GetSimilarMoves возвращает похожие ходы из базы данных
func (d *Database) GetSimilarMoves(boardHash string, limit int) ([]MoveRecord, error) {
	rows, err := d.db.Query(`
		SELECT id, game_id, move_number, from_row, from_col, to_row, to_col, evaluation, result, board_hash, created_at
		FROM moves
		WHERE board_hash = ?
		ORDER BY evaluation DESC
		LIMIT ?
	`, boardHash, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []MoveRecord
	for rows.Next() {
		var r MoveRecord
		err := rows.Scan(&r.ID, &r.GameID, &r.MoveNumber, &r.FromRow, &r.FromCol,
			&r.ToRow, &r.ToCol, &r.Evaluation, &r.Result, &r.BoardHash, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}

	return records, nil
}

// UpdateMoveResults обновляет результаты всех ходов в игре
func (d *Database) UpdateMoveResults(gameID int64, result string) error {
	_, err := d.db.Exec(
		"UPDATE moves SET result = ? WHERE game_id = ?",
		result, gameID,
	)
	return err
}

// GetTotalGames возвращает общее количество игр в базе
func (d *Database) GetTotalGames() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM games").Scan(&count)
	return count, err
}

// Close закрывает соединение с базой данных
func (d *Database) Close() error {
	return d.db.Close()
}

// GenerateBoardHash генерирует хеш для позиции на доске
func GenerateBoardHash(board *game.Board) string {
	hash := ""
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			piece := board.Cells[row][col]
			if piece.Type == game.Empty {
				hash += "."
			} else {
				symbol := pieceToChar(piece)
				hash += string(symbol)
			}
		}
	}
	return hash
}

// pieceToChar преобразует фигуру в символ
func pieceToChar(piece game.Piece) rune {
	var char rune
	switch piece.Type {
	case game.Pawn:
		char = 'p'
	case game.Knight:
		char = 'n'
	case game.Bishop:
		char = 'b'
	case game.Rook:
		char = 'r'
	case game.Queen:
		char = 'q'
	case game.King:
		char = 'k'
	default:
		char = '.'
	}

	if piece.Color == game.White {
		return char - 32 // Преобразуем в заглавную букву
	}
	return char
}
