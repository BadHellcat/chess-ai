package stats

import (
	"encoding/json"
	"os"
	"sync"
)

// GameResult представляет результат одной игры
type GameResult struct {
	GameNumber int     `json:"gameNumber"`
	Winner     string  `json:"winner"`
	Epsilon    float64 `json:"epsilon"`
	MovesCount int     `json:"movesCount"`
}

// Statistics хранит статистику игр
type Statistics struct {
	Games []GameResult `json:"games"`
	mu    sync.Mutex
}

// NewStatistics создает новый объект статистики
func NewStatistics() *Statistics {
	stats := &Statistics{
		Games: []GameResult{},
	}
	stats.Load()
	return stats
}

// AddGame добавляет результат игры
func (s *Statistics) AddGame(result GameResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Games = append(s.Games, result)
	s.Save()
}

// GetStats возвращает все результаты игр
func (s *Statistics) GetStats() []GameResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Games
}

// Save сохраняет статистику в файл
func (s *Statistics) Save() error {
	os.MkdirAll("stats", 0755)
	file, err := os.Create("stats/games.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(s)
}

// Load загружает статистику из файла
func (s *Statistics) Load() error {
	file, err := os.Open("stats/games.json")
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(s)
}

// GetWinRate возвращает процент побед AI, игрока и ничьих
func (s *Statistics) GetWinRate() (float64, float64, float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.Games) == 0 {
		return 0, 0, 0
	}

	var aiWins, playerWins, draws float64
	for _, game := range s.Games {
		switch game.Winner {
		case "black":
			aiWins++
		case "white":
			playerWins++
		case "draw":
			draws++
		}
	}

	total := float64(len(s.Games))
	return aiWins / total * 100, playerWins / total * 100, draws / total * 100
}
