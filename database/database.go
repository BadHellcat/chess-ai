package database

import (
	"chess-ai/game"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Game представляет игру в базе данных
type GameRecord struct {
	ID           int64     `json:"id"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
	Winner       string    `json:"winner"`
	MovesCount   int       `json:"moves_count"`
	WhiteEpsilon float64   `json:"white_epsilon"`
	BlackEpsilon float64   `json:"black_epsilon"`
}

// Database представляет файловое хранилище
type Database struct {
	dbPath      string
	games       []GameRecord
	moves       []MoveRecord
	mu          sync.RWMutex
	nextGameID  int64
	nextMoveID  int64
	gamesFile   string
	movesFile   string
}

// MoveRecord представляет запись о ходе в базе данных
type MoveRecord struct {
	ID           int64     `json:"id"`
	GameID       int64     `json:"game_id"`
	MoveNumber   int       `json:"move_number"`
	FromRow      int       `json:"from_row"`
	FromCol      int       `json:"from_col"`
	ToRow        int       `json:"to_row"`
	ToCol        int       `json:"to_col"`
	Evaluation   float64   `json:"evaluation"`
	Result       string    `json:"result"` // "win", "loss", "draw", "ongoing"
	BoardHash    string    `json:"board_hash"`
	CreatedAt    time.Time `json:"created_at"`
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

	// Определяем пути к файлам
	baseDir := filepath.Dir(dbPath)
	gamesFile := filepath.Join(baseDir, "games.json")
	movesFile := filepath.Join(baseDir, "moves.json")

	database := &Database{
		dbPath:      dbPath,
		games:       []GameRecord{},
		moves:       []MoveRecord{},
		nextGameID:  1,
		nextMoveID:  1,
		gamesFile:   gamesFile,
		movesFile:   movesFile,
	}

	// Загружаем существующие данные
	if err := database.loadData(); err != nil {
		return nil, err
	}

	return database, nil
}

// loadData загружает данные из JSON файлов
func (d *Database) loadData() error {
	// Загружаем игры
	if data, err := os.ReadFile(d.gamesFile); err == nil {
		if err := json.Unmarshal(data, &d.games); err != nil {
			return fmt.Errorf("ошибка при разборе games.json: %v", err)
		}
		// Находим максимальный ID
		for _, game := range d.games {
			if game.ID >= d.nextGameID {
				d.nextGameID = game.ID + 1
			}
		}
	}

	// Загружаем ходы
	if data, err := os.ReadFile(d.movesFile); err == nil {
		if err := json.Unmarshal(data, &d.moves); err != nil {
			return fmt.Errorf("ошибка при разборе moves.json: %v", err)
		}
		// Находим максимальный ID
		for _, move := range d.moves {
			if move.ID >= d.nextMoveID {
				d.nextMoveID = move.ID + 1
			}
		}
	}

	return nil
}

// saveData сохраняет данные в JSON файлы (должна вызываться только с удержанным мьютексом)
func (d *Database) saveData() error {
	// Сохраняем игры
	gamesData, err := json.MarshalIndent(d.games, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка при сериализации games: %v", err)
	}
	if err := os.WriteFile(d.gamesFile, gamesData, 0644); err != nil {
		return fmt.Errorf("ошибка при записи games.json: %v", err)
	}

	// Сохраняем ходы
	movesData, err := json.MarshalIndent(d.moves, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка при сериализации moves: %v", err)
	}
	if err := os.WriteFile(d.movesFile, movesData, 0644); err != nil {
		return fmt.Errorf("ошибка при записи moves.json: %v", err)
	}

	return nil
}

// StartGame создает новую игру в базе данных
func (d *Database) StartGame(whiteEpsilon, blackEpsilon float64) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	gameID := d.nextGameID
	d.nextGameID++

	game := GameRecord{
		ID:           gameID,
		StartedAt:    time.Now(),
		WhiteEpsilon: whiteEpsilon,
		BlackEpsilon: blackEpsilon,
	}

	d.games = append(d.games, game)

	return gameID, d.saveData()
}

// FinishGame обновляет информацию о завершенной игре
func (d *Database) FinishGame(gameID int64, winner string, movesCount int) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for i := range d.games {
		if d.games[i].ID == gameID {
			d.games[i].FinishedAt = time.Now()
			d.games[i].Winner = winner
			d.games[i].MovesCount = movesCount
			return d.saveData()
		}
	}

	return fmt.Errorf("игра с ID %d не найдена", gameID)
}

// RecordMove записывает ход в базу данных
func (d *Database) RecordMove(record MoveRecord) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	record.ID = d.nextMoveID
	d.nextMoveID++
	record.CreatedAt = time.Now()

	d.moves = append(d.moves, record)

	return d.saveData()
}

// GetPositionStats возвращает статистику для данной позиции
func (d *Database) GetPositionStats(boardHash string) (*PositionStats, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := &PositionStats{BoardHash: boardHash}
	
	gameIDs := make(map[int64]bool)
	var totalEval float64
	var evalCount int
	var bestMoveEval float64
	var bestMove *MoveRecord

	// Собираем статистику
	for _, move := range d.moves {
		if move.BoardHash == boardHash {
			gameIDs[move.GameID] = true
			
			if move.Result == "win" {
				stats.Wins++
			} else if move.Result == "loss" {
				stats.Losses++
			} else if move.Result == "draw" {
				stats.Draws++
			}

			totalEval += move.Evaluation
			evalCount++

			// Ищем лучший ход среди победных
			if move.Result == "win" && (bestMove == nil || move.Evaluation > bestMoveEval) {
				moveCopy := move
				bestMove = &moveCopy
				bestMoveEval = move.Evaluation
			}
		}
	}

	stats.TotalGames = len(gameIDs)
	if evalCount > 0 {
		stats.AvgEval = totalEval / float64(evalCount)
	}

	if bestMove != nil {
		stats.BestMove = &game.Move{
			From: game.Position{Row: bestMove.FromRow, Col: bestMove.FromCol},
			To:   game.Position{Row: bestMove.ToRow, Col: bestMove.ToCol},
		}
		stats.BestMoveEval = bestMoveEval
	}

	return stats, nil
}

// GetSimilarMoves возвращает похожие ходы из базы данных
func (d *Database) GetSimilarMoves(boardHash string, limit int) ([]MoveRecord, error) {
	// Валидация параметра limit
	if limit <= 0 || limit > 1000 {
		return nil, fmt.Errorf("limit must be between 1 and 1000, got: %d", limit)
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	var records []MoveRecord
	
	// Собираем все ходы с данным хешем
	for _, move := range d.moves {
		if move.BoardHash == boardHash {
			records = append(records, move)
		}
	}

	// Сортируем по оценке (DESC) и ограничиваем количество
	// Простая сортировка пузырьком для небольших наборов
	for i := 0; i < len(records)-1; i++ {
		for j := 0; j < len(records)-i-1; j++ {
			if records[j].Evaluation < records[j+1].Evaluation {
				records[j], records[j+1] = records[j+1], records[j]
			}
		}
	}

	// Ограничиваем результат
	if len(records) > limit {
		records = records[:limit]
	}

	return records, nil
}

// UpdateMoveResults обновляет результаты всех ходов в игре
func (d *Database) UpdateMoveResults(gameID int64, result string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	updated := false
	for i := range d.moves {
		if d.moves[i].GameID == gameID {
			d.moves[i].Result = result
			updated = true
		}
	}

	if !updated {
		return nil // Нет ходов для обновления
	}

	return d.saveData()
}

// GetTotalGames возвращает общее количество игр в базе
func (d *Database) GetTotalGames() (int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.games), nil
}

// Close закрывает соединение с базой данных
func (d *Database) Close() error {
	// Для JSON-хранилища ничего не нужно закрывать
	return nil
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
