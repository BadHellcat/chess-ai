package stats

import (
	"os"
	"testing"
)

// TestNewStatistics проверяет создание новой статистики
func TestNewStatistics(t *testing.T) {
	// Удаляем файл статистики, если он существует
	os.Remove("stats/games.json")

	stats := NewStatistics()

	if stats == nil {
		t.Fatal("NewStatistics() returned nil")
	}

	if stats.Games == nil {
		t.Error("Games slice should not be nil")
	}
}

// TestAddGame проверяет добавление результата игры
func TestAddGame(t *testing.T) {
	os.Remove("stats/games.json")
	stats := NewStatistics()

	result := GameResult{
		GameNumber: 1,
		Winner:     "black",
		Epsilon:    0.1,
		MovesCount: 45,
	}

	initialCount := len(stats.Games)
	stats.AddGame(result)

	if len(stats.Games) != initialCount+1 {
		t.Error("Games count should increase after AddGame")
	}

	lastGame := stats.Games[len(stats.Games)-1]
	if lastGame.GameNumber != 1 {
		t.Errorf("Expected GameNumber to be 1, got %d", lastGame.GameNumber)
	}

	if lastGame.Winner != "black" {
		t.Errorf("Expected Winner to be 'black', got %s", lastGame.Winner)
	}
}

// TestGetStats проверяет получение статистики
func TestGetStats(t *testing.T) {
	os.Remove("stats/games.json")
	stats := NewStatistics()

	result1 := GameResult{GameNumber: 1, Winner: "black", Epsilon: 0.1, MovesCount: 30}
	result2 := GameResult{GameNumber: 2, Winner: "white", Epsilon: 0.09, MovesCount: 40}

	stats.AddGame(result1)
	stats.AddGame(result2)

	allStats := stats.GetStats()

	if len(allStats) != 2 {
		t.Errorf("Expected 2 games, got %d", len(allStats))
	}
}

// TestSaveLoad проверяет сохранение и загрузку статистики
func TestSaveLoad(t *testing.T) {
	os.Remove("stats/games.json")
	stats := NewStatistics()

	result := GameResult{
		GameNumber: 42,
		Winner:     "draw",
		Epsilon:    0.05,
		MovesCount: 60,
	}

	stats.AddGame(result)

	// Создаем новую статистику и загружаем
	newStats := NewStatistics()

	if len(newStats.Games) == 0 {
		t.Error("Loaded statistics should not be empty")
	}

	lastGame := newStats.Games[len(newStats.Games)-1]
	if lastGame.GameNumber != 42 {
		t.Errorf("Expected GameNumber to be 42, got %d", lastGame.GameNumber)
	}

	if lastGame.Winner != "draw" {
		t.Errorf("Expected Winner to be 'draw', got %s", lastGame.Winner)
	}
}

// TestGetWinRate проверяет подсчет процента побед
func TestGetWinRate(t *testing.T) {
	os.Remove("stats/games.json")
	stats := NewStatistics()

	// Добавляем 10 игр: 5 побед AI, 3 победы игрока, 2 ничьи
	for i := 0; i < 5; i++ {
		stats.AddGame(GameResult{GameNumber: i + 1, Winner: "black", Epsilon: 0.1, MovesCount: 30})
	}

	for i := 5; i < 8; i++ {
		stats.AddGame(GameResult{GameNumber: i + 1, Winner: "white", Epsilon: 0.1, MovesCount: 30})
	}

	for i := 8; i < 10; i++ {
		stats.AddGame(GameResult{GameNumber: i + 1, Winner: "draw", Epsilon: 0.1, MovesCount: 30})
	}

	aiWinRate, playerWinRate, drawRate := stats.GetWinRate()

	if aiWinRate != 50.0 {
		t.Errorf("Expected AI win rate 50%%, got %.1f%%", aiWinRate)
	}

	if playerWinRate != 30.0 {
		t.Errorf("Expected player win rate 30%%, got %.1f%%", playerWinRate)
	}

	if drawRate != 20.0 {
		t.Errorf("Expected draw rate 20%%, got %.1f%%", drawRate)
	}
}

// TestGetWinRateEmpty проверяет процент побед для пустой статистики
func TestGetWinRateEmpty(t *testing.T) {
	os.Remove("stats/games.json")
	stats := NewStatistics()

	aiWinRate, playerWinRate, drawRate := stats.GetWinRate()

	if aiWinRate != 0.0 || playerWinRate != 0.0 || drawRate != 0.0 {
		t.Error("Empty statistics should return 0% for all rates")
	}
}

// TestGameResult проверяет структуру GameResult
func TestGameResult(t *testing.T) {
	result := GameResult{
		GameNumber: 1,
		Winner:     "black",
		Epsilon:    0.1,
		MovesCount: 50,
	}

	if result.GameNumber != 1 {
		t.Errorf("Expected GameNumber to be 1, got %d", result.GameNumber)
	}

	if result.Winner != "black" {
		t.Errorf("Expected Winner to be 'black', got %s", result.Winner)
	}

	if result.Epsilon != 0.1 {
		t.Errorf("Expected Epsilon to be 0.1, got %f", result.Epsilon)
	}

	if result.MovesCount != 50 {
		t.Errorf("Expected MovesCount to be 50, got %d", result.MovesCount)
	}
}

// TestConcurrentAccess проверяет потокобезопасность
func TestConcurrentAccess(t *testing.T) {
	os.Remove("stats/games.json")
	stats := NewStatistics()

	// Запускаем несколько горутин для одновременного добавления игр
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			result := GameResult{
				GameNumber: n,
				Winner:     "black",
				Epsilon:    0.1,
				MovesCount: 30,
			}
			stats.AddGame(result)
			done <- true
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < 10; i++ {
		<-done
	}

	if len(stats.Games) != 10 {
		t.Errorf("Expected 10 games after concurrent adds, got %d", len(stats.Games))
	}
}
