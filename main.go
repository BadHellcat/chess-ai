package main

import (
	"bufio"
	"chess-ai/agent"
	"chess-ai/database"
	"chess-ai/game"
	"chess-ai/selfplay"
	"chess-ai/stats"
	"chess-ai/ui"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Определяем флаги командной строки
	terminalMode := flag.Bool("terminal", false, "Запустить в терминальном режиме")
	selfPlayMode := flag.Bool("self-play", false, "Режим самообучения (AI играет сам с собой)")
	numGames := flag.Int("games", 100, "Количество игр для самообучения")
	dbPath := flag.String("db", "data/chess.db", "Путь к директории базы данных")
	flag.Parse()

	if *selfPlayMode {
		runSelfPlay(*numGames, *dbPath)
	} else if *terminalMode {
		runTerminal(*dbPath)
	} else {
		runWeb(*dbPath)
	}
}

func runSelfPlay(numGames int, dbPath string) {
	fmt.Println("=== Режим самообучения шахматной нейросети ===")

	// Валидация параметров
	if numGames <= 0 {
		fmt.Printf("Ошибка: количество игр должно быть больше нуля (указано: %d)\n", numGames)
		os.Exit(1)
	}

	// Создаем соединение с базой данных
	db, err := database.NewDatabase(dbPath)
	if err != nil {
		fmt.Printf("Ошибка при создании базы данных: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Создаем менеджер самообучения
	manager := selfplay.NewSelfPlayManager(db)

	// Запускаем обучение
	err = manager.Train(numGames, true)
	if err != nil {
		fmt.Printf("Ошибка во время обучения: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nОбучение успешно завершено!")
}

func runWeb(dbPath string) {
	fmt.Println("=== Шахматы с обучающейся нейросетью ===")
	fmt.Println("Запуск веб-сервера...")
	fmt.Println("Откройте браузер на http://localhost:8080")

	board := game.NewBoard()
	ai := agent.NewAgent(game.Black)
	statistics := stats.NewStatistics()

	// Подключаем базу данных
	db, err := database.NewDatabase(dbPath)
	if err != nil {
		fmt.Printf("Предупреждение: не удалось подключиться к базе данных: %v\n", err)
	} else {
		ai.SetDatabase(db, true)
		defer db.Close()
		fmt.Println("База данных подключена для анализа ходов")
	}

	webUI := ui.NewWebUI(board, ai, statistics)
	webUI.Start(8080)
}

func runTerminal(dbPath string) {
	fmt.Println("=== Шахматы с обучающейся нейросетью ===")
	fmt.Println("Вы играете белыми (заглавные буквы)")
	fmt.Println("Введите ход в формате: e2 e4")
	fmt.Println("Для рокировки: e1 g1 (короткая) или e1 c1 (длинная)")
	fmt.Println()

	board := game.NewBoard()
	ai := agent.NewAgent(game.Black)

	// Подключаем базу данных
	db, err := database.NewDatabase(dbPath)
	if err != nil {
		fmt.Printf("Предупреждение: не удалось подключиться к базе данных: %v\n", err)
	} else {
		ai.SetDatabase(db, true)
		defer db.Close()
		fmt.Println("База данных подключена для анализа ходов")
		fmt.Println()
	}

	scanner := bufio.NewScanner(os.Stdin)
	gamesPlayed := 0

	for {
		fmt.Print(board.String())

		if board.GameOver {
			handleGameOver(board, ai, &gamesPlayed)
			board = game.NewBoard()
			ai.StateHistory = nil
			ai.RewardHistory = nil
			continue
		}

		if board.CurrentTurn == game.White {
			fmt.Print("Ваш ход: ")
			if !scanner.Scan() {
				break
			}

			input := scanner.Text()
			if input == "quit" || input == "exit" {
				ai.Save()
				fmt.Println("Игра сохранена. До свидания!")
				break
			}

			move := parseMove(input)
			if !board.IsValidMove(move) {
				fmt.Println("Некорректный ход! Попробуйте еще раз.")
				continue
			}

			board.MakeMove(move)

		} else {
			fmt.Println("AI думает...")
			ai.RecordState(board)
			move := ai.ChooseMove(board)
			board.MakeMove(move)
			fmt.Printf("AI ходит: %s -> %s\n",
				posToString(move.From),
				posToString(move.To))
		}
	}
}

func handleGameOver(board *game.Board, ai *agent.Agent, gamesPlayed *int) {
	*gamesPlayed++

	fmt.Println("\n=== ИГРА ОКОНЧЕНА ===")

	var reward float64
	if board.Winner == game.Black {
		fmt.Println("AI победил!")
		reward = 1.0
	} else if board.Winner == game.White {
		fmt.Println("Вы победили!")
		reward = 0.0
	} else {
		fmt.Println("Ничья!")
		reward = 0.5
	}

	fmt.Println("AI обучается на результатах игры...")
	ai.Learn(reward)

	fmt.Printf("Игр сыграно: %d\n", *gamesPlayed)
	fmt.Printf("Текущий уровень исследования (epsilon): %.4f\n\n", ai.Epsilon)
}

func parseMove(input string) game.Move {
	parts := strings.Fields(input)
	if len(parts) != 2 {
		return game.Move{}
	}

	from := parsePosition(parts[0])
	to := parsePosition(parts[1])

	return game.Move{From: from, To: to}
}

func parsePosition(pos string) game.Position {
	if len(pos) != 2 {
		return game.Position{-1, -1}
	}

	col := int(pos[0] - 'a')
	row := 8 - int(pos[1]-'0')

	return game.Position{Row: row, Col: col}
}

func posToString(pos game.Position) string {
	col := string(rune('a' + pos.Col))
	row := fmt.Sprintf("%d", 8-pos.Row)
	return col + row
}
