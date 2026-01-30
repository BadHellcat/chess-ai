package database

import (
	"chess-ai/game"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDatabaseOperations(t *testing.T) {
	// Создаем временную директорию
	tmpDir := filepath.Join(os.TempDir(), "chess-test-"+time.Now().Format("20060102150405"))
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Создаем базу данных
	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Не удалось создать базу данных: %v", err)
	}
	defer db.Close()

	// Тест 1: Создание игры
	gameID, err := db.StartGame(0.1, 0.1)
	if err != nil {
		t.Fatalf("Ошибка при создании игры: %v", err)
	}
	if gameID != 1 {
		t.Errorf("Ожидался gameID = 1, получен %d", gameID)
	}

	// Тест 2: Запись хода
	moveRecord := MoveRecord{
		GameID:     gameID,
		MoveNumber: 1,
		FromRow:    6,
		FromCol:    4,
		ToRow:      4,
		ToCol:      4,
		Evaluation: 0.5,
		BoardHash:  "test-hash-1",
		Result:     "ongoing",
	}
	err = db.RecordMove(moveRecord)
	if err != nil {
		t.Fatalf("Ошибка при записи хода: %v", err)
	}

	// Тест 3: Получение статистики позиции
	stats, err := db.GetPositionStats("test-hash-1")
	if err != nil {
		t.Fatalf("Ошибка при получении статистики: %v", err)
	}
	if stats.TotalGames != 1 {
		t.Errorf("Ожидалось 1 игра, получено %d", stats.TotalGames)
	}

	// Тест 4: Завершение игры
	err = db.FinishGame(gameID, "white", 10)
	if err != nil {
		t.Fatalf("Ошибка при завершении игры: %v", err)
	}

	// Тест 5: Получение количества игр
	totalGames, err := db.GetTotalGames()
	if err != nil {
		t.Fatalf("Ошибка при получении количества игр: %v", err)
	}
	if totalGames != 1 {
		t.Errorf("Ожидалось 1 игра, получено %d", totalGames)
	}

	// Тест 6: Обновление результатов ходов
	err = db.UpdateMoveResults(gameID, "win")
	if err != nil {
		t.Fatalf("Ошибка при обновлении результатов: %v", err)
	}

	// Тест 7: Получение похожих ходов
	similarMoves, err := db.GetSimilarMoves("test-hash-1", 10)
	if err != nil {
		t.Fatalf("Ошибка при получении похожих ходов: %v", err)
	}
	if len(similarMoves) != 1 {
		t.Errorf("Ожидался 1 ход, получено %d", len(similarMoves))
	}
	if similarMoves[0].Result != "win" {
		t.Errorf("Ожидался результат 'win', получено '%s'", similarMoves[0].Result)
	}
}

func TestBoardHashGeneration(t *testing.T) {
	board := game.NewBoard()
	hash := GenerateBoardHash(board)
	
	if hash == "" {
		t.Error("Хеш доски не должен быть пустым")
	}
	
	// Проверяем, что хеш имеет правильную длину (64 клетки)
	if len(hash) != 64 {
		t.Errorf("Ожидалась длина хеша 64, получено %d", len(hash))
	}
}

func TestDatabasePersistence(t *testing.T) {
	// Создаем временную директорию
	tmpDir := filepath.Join(os.TempDir(), "chess-test-persist-"+time.Now().Format("20060102150405"))
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")

	// Создаем базу данных и добавляем данные
	db1, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Не удалось создать базу данных: %v", err)
	}

	gameID, _ := db1.StartGame(0.2, 0.3)
	db1.RecordMove(MoveRecord{
		GameID:     gameID,
		MoveNumber: 1,
		FromRow:    6,
		FromCol:    4,
		ToRow:      4,
		ToCol:      4,
		Evaluation: 0.7,
		BoardHash:  "persist-hash",
		Result:     "win",
	})
	db1.Close()

	// Открываем базу данных снова
	db2, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("Не удалось открыть существующую базу данных: %v", err)
	}
	defer db2.Close()

	// Проверяем, что данные сохранились
	totalGames, _ := db2.GetTotalGames()
	if totalGames != 1 {
		t.Errorf("Ожидалась 1 сохраненная игра, получено %d", totalGames)
	}

	stats, _ := db2.GetPositionStats("persist-hash")
	if stats.TotalGames != 1 {
		t.Errorf("Ожидалась статистика для 1 игры, получено %d", stats.TotalGames)
	}
	if stats.Wins != 1 {
		t.Errorf("Ожидалась 1 победа, получено %d", stats.Wins)
	}
}
