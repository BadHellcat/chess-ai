package selfplay

import (
	"chess-ai/agent"
	"chess-ai/database"
	"chess-ai/game"
	"fmt"
	"time"
)

// SelfPlayManager управляет процессом самообучения
type SelfPlayManager struct {
	whiteAgent *agent.Agent
	blackAgent *agent.Agent
	db         *database.Database
	gamesCount int
}

// NewSelfPlayManager создает новый менеджер самообучения
func NewSelfPlayManager(db *database.Database) *SelfPlayManager {
	whiteAgent := agent.NewAgent(game.White)
	blackAgent := agent.NewAgent(game.Black)

	// Оба агента должны использовать одну и ту же нейросеть
	// чтобы обучаться на опыте друг друга
	blackAgent.Network = whiteAgent.Network

	// Настраиваем базу данных для агентов
	whiteAgent.SetDatabase(db, true)
	blackAgent.SetDatabase(db, true)

	return &SelfPlayManager{
		whiteAgent: whiteAgent,
		blackAgent: blackAgent,
		db:         db,
		gamesCount: 0,
	}
}

// PlayGame запускает одну игру между двумя агентами
func (m *SelfPlayManager) PlayGame(verbose bool) error {
	board := game.NewBoard()
	m.gamesCount++

	// Записываем начало игры в базу данных
	gameID, err := m.db.StartGame(m.whiteAgent.Epsilon, m.blackAgent.Epsilon)
	if err != nil {
		return fmt.Errorf("ошибка при создании игры в БД: %v", err)
	}

	if verbose {
		fmt.Printf("\n=== Игра #%d начата (ID: %d) ===\n", m.gamesCount, gameID)
	}

	moveNumber := 0
	var moves []struct {
		move       game.Move
		evaluation float64
		boardHash  string
	}

	// Игровой цикл
	for !board.GameOver && moveNumber < 200 {
		var currentAgent *agent.Agent
		if board.CurrentTurn == game.White {
			currentAgent = m.whiteAgent
		} else {
			currentAgent = m.blackAgent
		}

		// Записываем состояние
		currentAgent.RecordState(board)

		// Получаем хеш позиции перед ходом
		boardHash := database.GenerateBoardHash(board)

		// Выбираем ход
		move := currentAgent.ChooseMove(board)
		if move.From.Row == -1 {
			break
		}

		// Оцениваем позицию
		evaluation := currentAgent.Network.Forward(currentAgent.StateHistory[len(currentAgent.StateHistory)-1])

		// Сохраняем информацию о ходе
		moves = append(moves, struct {
			move       game.Move
			evaluation float64
			boardHash  string
		}{move, evaluation, boardHash})

		// Делаем ход
		board.MakeMove(move)
		moveNumber++

		if verbose && moveNumber%10 == 0 {
			fmt.Printf("  Ход %d выполнен\n", moveNumber)
		}
	}

	// Определяем результат
	var winner string
	var whiteReward, blackReward float64

	if board.GameOver {
		if board.Winner == game.White {
			winner = "white"
			whiteReward = 1.0
			blackReward = 0.0
		} else if board.Winner == game.Black {
			winner = "black"
			whiteReward = 0.0
			blackReward = 1.0
		} else {
			winner = "draw"
			whiteReward = 0.5
			blackReward = 0.5
		}
	} else {
		winner = "draw"
		whiteReward = 0.5
		blackReward = 0.5
	}

	// Записываем все ходы в базу данных
	for i, moveInfo := range moves {
		var result string
		// Определяем результат для каждого хода в зависимости от того, кто его сделал
		if i%2 == 0 { // Ход белых
			if winner == "white" {
				result = "win"
			} else if winner == "black" {
				result = "loss"
			} else {
				result = "draw"
			}
		} else { // Ход черных
			if winner == "black" {
				result = "win"
			} else if winner == "white" {
				result = "loss"
			} else {
				result = "draw"
			}
		}

		err := m.db.RecordMove(database.MoveRecord{
			GameID:     gameID,
			MoveNumber: i + 1,
			FromRow:    moveInfo.move.From.Row,
			FromCol:    moveInfo.move.From.Col,
			ToRow:      moveInfo.move.To.Row,
			ToCol:      moveInfo.move.To.Col,
			Evaluation: moveInfo.evaluation,
			BoardHash:  moveInfo.boardHash,
			Result:     result,
		})
		if err != nil {
			return fmt.Errorf("ошибка при записи хода: %v", err)
		}
	}

	// Завершаем игру в базе данных
	err = m.db.FinishGame(gameID, winner, moveNumber)
	if err != nil {
		return fmt.Errorf("ошибка при завершении игры: %v", err)
	}

	// Обучаем агентов
	m.whiteAgent.Learn(whiteReward)
	m.blackAgent.Learn(blackReward)

	// Очищаем историю состояний после обучения
	m.whiteAgent.StateHistory = nil
	m.blackAgent.StateHistory = nil

	if verbose {
		fmt.Printf("=== Игра #%d завершена: %s, ходов: %d ===\n", m.gamesCount, winner, moveNumber)
		fmt.Printf("  Epsilon белых: %.4f, черных: %.4f\n", m.whiteAgent.Epsilon, m.blackAgent.Epsilon)
	}

	return nil
}

// Train запускает обучение на заданное количество игр
func (m *SelfPlayManager) Train(numGames int, verbose bool) error {
	startTime := time.Now()

	if verbose {
		fmt.Printf("\n╔═══════════════════════════════════════════════╗\n")
		fmt.Printf("║   РЕЖИМ САМООБУЧЕНИЯ                         ║\n")
		fmt.Printf("╚═══════════════════════════════════════════════╝\n")
		fmt.Printf("Начинается обучение на %d играх...\n\n", numGames)
	}

	for i := 0; i < numGames; i++ {
		err := m.PlayGame(verbose && (i%10 == 0 || i < 5))
		if err != nil {
			return fmt.Errorf("ошибка в игре %d: %v", i+1, err)
		}

		// Сохраняем веса каждые 10 игр
		if (i+1)%10 == 0 {
			m.whiteAgent.Save()
			m.blackAgent.Save()

			if verbose {
				elapsed := time.Since(startTime)
				fmt.Printf("\n--- Прогресс: %d/%d игр завершено (%.1f%%) ---\n",
					i+1, numGames, float64(i+1)/float64(numGames)*100)
				fmt.Printf("    Время: %s\n", elapsed.Round(time.Second))
				fmt.Printf("    Скорость: %.1f игр/сек\n\n", float64(i+1)/elapsed.Seconds())
			}
		}
	}

	// Финальное сохранение
	m.whiteAgent.Save()
	m.blackAgent.Save()

	if verbose {
		totalTime := time.Since(startTime)
		fmt.Printf("\n╔═══════════════════════════════════════════════╗\n")
		fmt.Printf("║   ОБУЧЕНИЕ ЗАВЕРШЕНО                         ║\n")
		fmt.Printf("╚═══════════════════════════════════════════════╝\n")
		fmt.Printf("Всего игр: %d\n", numGames)
		fmt.Printf("Общее время: %s\n", totalTime.Round(time.Second))
		fmt.Printf("Средняя скорость: %.1f игр/сек\n", float64(numGames)/totalTime.Seconds())

		// Показываем статистику из базы данных
		totalGames, err := m.db.GetTotalGames()
		if err == nil {
			fmt.Printf("\nВсего игр в базе данных: %d\n", totalGames)
		}
	}

	return nil
}

// GetGamesCount возвращает количество сыгранных игр
func (m *SelfPlayManager) GetGamesCount() int {
	return m.gamesCount
}
