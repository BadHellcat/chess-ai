package ui

import (
	"chess-ai/agent"
	"chess-ai/game"
	"chess-ai/stats"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// WebUI представляет веб-интерфейс для игры в шахматы
type WebUI struct {
	board      *game.Board
	agent      *agent.Agent
	statistics *stats.Statistics
	mutex      sync.Mutex
	
	// Для режима самообучения
	selfPlayRunning bool
	selfPlayStop    chan bool
	whiteAgent      *agent.Agent
	blackAgent      *agent.Agent
}

// NewWebUI создает новый веб-интерфейс
func NewWebUI(board *game.Board, agentAI *agent.Agent, statistics *stats.Statistics) *WebUI {
	return &WebUI{
		board:           board,
		agent:           agentAI,
		statistics:      statistics,
		selfPlayRunning: false,
		selfPlayStop:    make(chan bool),
		whiteAgent:      agent.NewAgent(game.White),
		blackAgent:      agent.NewAgent(game.Black),
	}
}

// Start запускает веб-сервер
func (w *WebUI) Start(port int) error {
	http.HandleFunc("/", w.handleIndex)
	http.HandleFunc("/api/state", w.handleState)
	http.HandleFunc("/api/move", w.handleMove)
	http.HandleFunc("/api/reset", w.handleReset)
	http.HandleFunc("/api/stats", w.handleStats)
	http.HandleFunc("/api/selfplay/start", w.handleSelfPlayStart)
	http.HandleFunc("/api/selfplay/stop", w.handleSelfPlayStop)
	http.HandleFunc("/api/selfplay/status", w.handleSelfPlayStatus)

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Starting web server on http://localhost%s\n", addr)
	
	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return server.ListenAndServe()
}

// handleIndex возвращает HTML страницу
func (w *WebUI) handleIndex(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.Write([]byte(htmlPage))
}

// BoardState представляет состояние доски
type BoardState struct {
	Cells       [8][8]CellState `json:"cells"`
	CurrentTurn string          `json:"currentTurn"`
	GameOver    bool            `json:"gameOver"`
	Winner      string          `json:"winner"`
	IsCheck     bool            `json:"isCheck"`
	Epsilon     float64         `json:"epsilon"`
	MovesCount  int             `json:"movesCount"`
}

// CellState представляет состояние клетки
type CellState struct {
	Piece string `json:"piece"`
	Color string `json:"color"`
}

// handleState возвращает текущее состояние доски
func (w *WebUI) handleState(rw http.ResponseWriter, r *http.Request) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.writeState(rw)
}

// writeState writes the current board state to the response (must be called with mutex held)
func (w *WebUI) writeState(rw http.ResponseWriter) {
	state := BoardState{
		CurrentTurn: colorToString(w.board.CurrentTurn),
		GameOver:    w.board.GameOver,
		Winner:      colorToString(w.board.Winner),
		IsCheck:     w.board.IsCheck,
		Epsilon:     w.agent.Epsilon,
		MovesCount:  w.board.MovesCount,
	}

	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			piece := w.board.Cells[row][col]
			state.Cells[row][col] = CellState{
				Piece: pieceTypeToString(piece.Type),
				Color: colorToString(piece.Color),
			}
		}
	}

	rw.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(rw).Encode(state); err != nil {
		http.Error(rw, "Failed to encode state", http.StatusInternalServerError)
	}
}

// MoveRequest представляет запрос на ход
type MoveRequest struct {
	FromRow int `json:"fromRow"`
	FromCol int `json:"fromCol"`
	ToRow   int `json:"toRow"`
	ToCol   int `json:"toCol"`
}

// handleMove обрабатывает ход игрока
func (w *WebUI) handleMove(rw http.ResponseWriter, r *http.Request) {
	w.mutex.Lock()

	var req MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.mutex.Unlock()
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	if req.FromRow < 0 || req.FromRow > 7 || req.FromCol < 0 || req.FromCol > 7 ||
		req.ToRow < 0 || req.ToRow > 7 || req.ToCol < 0 || req.ToCol > 7 {
		w.mutex.Unlock()
		http.Error(rw, "Invalid coordinates", http.StatusBadRequest)
		return
	}

	move := game.Move{
		From: game.Position{Row: req.FromRow, Col: req.FromCol},
		To:   game.Position{Row: req.ToRow, Col: req.ToCol},
	}

	if !w.board.IsValidMove(move) {
		w.mutex.Unlock()
		http.Error(rw, "Invalid move", http.StatusBadRequest)
		return
	}

	w.board.MakeMove(move)

	// Capture state before releasing mutex
	gameOver := w.board.GameOver
	currentTurn := w.board.CurrentTurn
	aiColor := w.agent.Color

	// Send response immediately after player's move
	w.writeState(rw)
	
	// Handle game over for player's winning move synchronously (with mutex held)
	if gameOver {
		w.handleGameEnd()
		w.mutex.Unlock()
		return
	}
	
	w.mutex.Unlock()

	// Process AI move asynchronously if it's AI's turn
	if !gameOver && currentTurn == aiColor {
		go func() {
			// Clone board for AI computation
			w.mutex.Lock()
			boardClone := w.board.Clone()
			
			// Record state before making move (with mutex held)
			w.agent.RecordState(boardClone)
			w.mutex.Unlock()

			// Compute AI move without holding mutex (can take 10+ seconds)
			aiMove := w.agent.ChooseMove(boardClone)

			// Re-acquire mutex to apply the move
			w.mutex.Lock()
			defer w.mutex.Unlock()

			// Verify game state is still valid (game not reset, still AI's turn)
			if !w.board.GameOver && w.board.CurrentTurn == aiColor {
				w.board.MakeMove(aiMove)

				if w.board.GameOver {
					w.handleGameEnd()
				}
			}
		}()
	}
}

// handleGameEnd handles the end of game, learning and statistics (must be called with mutex held)
func (w *WebUI) handleGameEnd() {
	var reward float64
	
	// Determine if it's a draw: game is over and no checkmate occurred
	// In board.go, draws set Winner to White (stalemate or move limit)
	isDraw := w.board.GameOver && !w.board.IsCheck
	
	if isDraw {
		reward = 0.5 // Draw
	} else if w.board.Winner == w.agent.Color {
		reward = 1.0 // AI won
	} else {
		reward = 0.0 // AI lost
	}
	
	// Train the AI
	w.agent.Learn(reward)
	w.agent.Save()
	
	gameNumber := len(w.statistics.GetStats()) + 1
	
	// Determine winner string for stats
	winnerStr := "draw"
	if !isDraw {
		winnerStr = colorToString(w.board.Winner)
	}
	
	result := stats.GameResult{
		GameNumber: gameNumber,
		Winner:     winnerStr,
		Epsilon:    w.agent.Epsilon,
		MovesCount: w.board.MovesCount,
	}
	w.statistics.AddGame(result)
	
	// Reset state history for next game
	w.agent.StateHistory = nil
	w.agent.RewardHistory = nil
}

// handleReset сбрасывает игру
func (w *WebUI) handleReset(rw http.ResponseWriter, r *http.Request) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	w.board.CurrentTurn = game.White
	w.board.GameOver = false
	w.board.Winner = game.White
	w.board.EnPassantTarget = nil
	w.board.WhiteKingMoved = false
	w.board.WhiteRookAMoved = false
	w.board.WhiteRookHMoved = false
	w.board.BlackKingMoved = false
	w.board.BlackRookAMoved = false
	w.board.BlackRookHMoved = false
	w.board.MovesCount = 0
	
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			w.board.Cells[i][j] = game.Piece{Type: game.Empty, Color: game.White}
		}
	}
	
	w.board.Cells[0] = [8]game.Piece{
		{game.Rook, game.Black}, {game.Knight, game.Black}, {game.Bishop, game.Black}, {game.Queen, game.Black},
		{game.King, game.Black}, {game.Bishop, game.Black}, {game.Knight, game.Black}, {game.Rook, game.Black},
	}
	for i := 0; i < 8; i++ {
		w.board.Cells[1][i] = game.Piece{game.Pawn, game.Black}
	}
	w.board.Cells[7] = [8]game.Piece{
		{game.Rook, game.White}, {game.Knight, game.White}, {game.Bishop, game.White}, {game.Queen, game.White},
		{game.King, game.White}, {game.Bishop, game.White}, {game.Knight, game.White}, {game.Rook, game.White},
	}
	for i := 0; i < 8; i++ {
		w.board.Cells[6][i] = game.Piece{game.Pawn, game.White}
	}
	
	rw.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(rw).Encode(map[string]string{"status": "ok"}); err != nil {
		http.Error(rw, "Failed to encode response", http.StatusInternalServerError)
	}
}

// handleStats возвращает статистику
func (w *WebUI) handleStats(rw http.ResponseWriter, r *http.Request) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	rw.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(rw).Encode(w.statistics.GetStats()); err != nil {
		http.Error(rw, "Failed to encode stats", http.StatusInternalServerError)
	}
}

// handleSelfPlayStart запускает режим самообучения
func (w *WebUI) handleSelfPlayStart(rw http.ResponseWriter, r *http.Request) {
	w.mutex.Lock()
	
	if w.selfPlayRunning {
		w.mutex.Unlock()
		http.Error(rw, "Self-play is already running", http.StatusBadRequest)
		return
	}
	
	w.selfPlayRunning = true
	w.selfPlayStop = make(chan bool)
	w.mutex.Unlock()
	
	// Запускаем самообучение в отдельной горутине
	go w.runSelfPlay()
	
	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]bool{"success": true})
}

// handleSelfPlayStop останавливает режим самообучения
func (w *WebUI) handleSelfPlayStop(rw http.ResponseWriter, r *http.Request) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	if !w.selfPlayRunning {
		http.Error(rw, "Self-play is not running", http.StatusBadRequest)
		return
	}
	
	w.selfPlayStop <- true
	w.selfPlayRunning = false
	
	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]bool{"success": true})
}

// handleSelfPlayStatus возвращает статус самообучения
func (w *WebUI) handleSelfPlayStatus(rw http.ResponseWriter, r *http.Request) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	rw.Header().Set("Content-Type", "application/json")
	json.NewEncoder(rw).Encode(map[string]bool{"running": w.selfPlayRunning})
}

// runSelfPlay запускает процесс самообучения
func (w *WebUI) runSelfPlay() {
	for {
		select {
		case <-w.selfPlayStop:
			return
		default:
			w.mutex.Lock()
			// Сбрасываем доску для новой игры
			w.board = game.NewBoard()
			w.whiteAgent.StateHistory = nil
			w.blackAgent.StateHistory = nil
			w.mutex.Unlock()
			
			// Играем одну игру
			for {
				select {
				case <-w.selfPlayStop:
					return
				default:
					w.mutex.Lock()
					
					if w.board.GameOver {
						// Обучаем агентов
						var whiteReward, blackReward float64
						if w.board.Winner == game.White {
							whiteReward = 1.0
							blackReward = 0.0
						} else if w.board.Winner == game.Black {
							whiteReward = 0.0
							blackReward = 1.0
						} else {
							whiteReward = 0.5
							blackReward = 0.5
						}
						
						w.whiteAgent.Learn(whiteReward)
						w.blackAgent.Learn(blackReward)
						w.whiteAgent.Save()
						w.blackAgent.Save()
						
						w.mutex.Unlock()
						break
					}
					
					// Выбираем текущего агента
					var currentAgent *agent.Agent
					if w.board.CurrentTurn == game.White {
						currentAgent = w.whiteAgent
					} else {
						currentAgent = w.blackAgent
					}
					
					// Записываем состояние
					currentAgent.RecordState(w.board)
					
					// Выбираем ход
					move := currentAgent.ChooseMove(w.board)
					if move.From.Row == -1 {
						w.mutex.Unlock()
						break
					}
					
					// Делаем ход
					w.board.MakeMove(move)
					w.mutex.Unlock()
					
					// Небольшая пауза для визуализации
					time.Sleep(500 * time.Millisecond)
				}
			}
			
			// Пауза между играми
			time.Sleep(1 * time.Second)
		}
	}
}

// Вспомогательные функции
func colorToString(c game.Color) string {
	if c == game.White {
		return "white"
	}
	return "black"
}

func pieceTypeToString(pt game.PieceType) string {
	switch pt {
	case game.Pawn:
		return "pawn"
	case game.Knight:
		return "knight"
	case game.Bishop:
		return "bishop"
	case game.Rook:
		return "rook"
	case game.Queen:
		return "queen"
	case game.King:
		return "king"
	default:
		return ""
	}
}

const htmlPage = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Chess AI</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
            color: #333;
        }
        
        .container {
            max-width: 1400px;
            margin: 0 auto;
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            padding: 30px;
        }
        
        h1 {
            text-align: center;
            color: #667eea;
            margin-bottom: 30px;
            font-size: 2.5em;
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.1);
        }
        
        .game-area {
            display: grid;
            grid-template-columns: 1fr 600px 1fr;
            gap: 30px;
            align-items: start;
        }
        
        .left-panel, .right-panel {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 10px;
        }
        
        .status-bar {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 15px;
            border-radius: 10px;
            margin-bottom: 20px;
            text-align: center;
            font-size: 1.2em;
            font-weight: bold;
        }
        
        .chess-board {
            width: 600px;
            height: 600px;
            display: grid;
            grid-template-columns: repeat(8, 1fr);
            grid-template-rows: repeat(8, 1fr);
            border: 4px solid #333;
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
            border-radius: 10px;
            overflow: hidden;
        }
        
        .cell {
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 60px;
            cursor: pointer;
            transition: all 0.2s;
            user-select: none;
        }
        
        .cell:hover {
            filter: brightness(0.9);
        }
        
        .cell.light {
            background-color: #f0d9b5;
        }
        
        .cell.dark {
            background-color: #b58863;
        }
        
        .cell.selected {
            background-color: #7cb342 !important;
            box-shadow: inset 0 0 20px rgba(0, 0, 0, 0.3);
        }
        
        .cell.valid-move {
            background-color: #ffeb3b !important;
        }
        
        .cell.dragging {
            opacity: 0.5;
        }
        
        .controls {
            margin-top: 20px;
        }
        
        button {
            width: 100%;
            padding: 15px;
            margin: 10px 0;
            font-size: 1.1em;
            font-weight: bold;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            transition: all 0.3s;
            text-transform: uppercase;
        }
        
        button.primary {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        
        button.secondary {
            background: #f44336;
            color: white;
        }
        
        button:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(0, 0, 0, 0.3);
        }
        
        button:active {
            transform: translateY(0);
        }
        
        .stats-section {
            margin-top: 20px;
        }
        
        .stats-section h3 {
            color: #667eea;
            margin-bottom: 15px;
            font-size: 1.3em;
        }
        
        .stat-item {
            background: white;
            padding: 12px;
            margin: 8px 0;
            border-radius: 8px;
            display: flex;
            justify-content: space-between;
            box-shadow: 0 2px 5px rgba(0, 0, 0, 0.1);
        }
        
        .stat-label {
            font-weight: bold;
            color: #555;
        }
        
        .stat-value {
            color: #667eea;
            font-weight: bold;
        }
        
        #progressChart {
            margin-top: 20px;
            border: 2px solid #ddd;
            border-radius: 10px;
            background: white;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        }
        
        .chart-title {
            text-align: center;
            margin: 20px 0 10px 0;
            color: #667eea;
            font-size: 1.2em;
            font-weight: bold;
        }
        
        @media (max-width: 1200px) {
            .game-area {
                grid-template-columns: 1fr;
            }
            
            .chess-board {
                margin: 0 auto;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>♔ Chess AI Game ♚</h1>
        
        <div class="game-area">
            <div class="left-panel">
                <div class="status-bar" id="statusBar">White's Turn</div>
                
                <div class="controls">
                    <button class="primary" onclick="resetGame()">🔄 New Game</button>
                    <button class="secondary" onclick="resetGame()">♻️ Reset</button>
                </div>
                
                <div class="stats-section">
                    <h3>🤖 Self-Play Mode</h3>
                    <div class="controls">
                        <button class="primary" id="startSelfPlay" onclick="startSelfPlay()">▶️ Start Training</button>
                        <button class="secondary" id="stopSelfPlay" onclick="stopSelfPlay()" style="display: none;">⏸️ Stop Training</button>
                    </div>
                    <div class="stat-item" style="margin-top: 10px;">
                        <span class="stat-label">Training Status:</span>
                        <span class="stat-value" id="selfPlayStatus">Stopped</span>
                    </div>
                </div>
                
                <div class="stats-section">
                    <h3>📊 Game Statistics</h3>
                    <div class="stat-item">
                        <span class="stat-label">Total Games:</span>
                        <span class="stat-value" id="totalGames">0</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-label">AI Wins:</span>
                        <span class="stat-value" id="aiWins">0</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-label">Player Wins:</span>
                        <span class="stat-value" id="playerWins">0</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-label">Draws:</span>
                        <span class="stat-value" id="draws">0</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-label">Win Rate:</span>
                        <span class="stat-value" id="winRate">0%</span>
                    </div>
                </div>
                
                <div class="stats-section">
                    <h3>🧠 AI Training Info</h3>
                    <div class="stat-item">
                        <span class="stat-label">Current Epsilon:</span>
                        <span class="stat-value" id="currentEpsilon">-</span>
                    </div>
                    <div class="stat-item">
                        <span class="stat-label">Moves This Game:</span>
                        <span class="stat-value" id="movesCount">0</span>
                    </div>
                </div>
            </div>
            
            <div class="center-panel">
                <div class="chess-board" id="chessBoard"></div>
            </div>
            
            <div class="right-panel">
                <div class="chart-title">📈 Progress Chart</div>
                <canvas id="progressChart" width="400" height="400"></canvas>
            </div>
        </div>
    </div>
    
    <script>
        const pieceSymbols = {
            white: {
                king: '♔',
                queen: '♕',
                rook: '♖',
                bishop: '♗',
                knight: '♘',
                pawn: '♙'
            },
            black: {
                king: '♚',
                queen: '♛',
                rook: '♜',
                bishop: '♝',
                knight: '♞',
                pawn: '♟'
            }
        };
        
        let selectedCell = null;
        let boardState = null;
        let statsData = [];
        
        function createBoard() {
            const board = document.getElementById('chessBoard');
            board.innerHTML = '';
            
            for (let row = 0; row < 8; row++) {
                for (let col = 0; col < 8; col++) {
                    const cell = document.createElement('div');
                    cell.className = 'cell ' + ((row + col) % 2 === 0 ? 'light' : 'dark');
                    cell.dataset.row = row;
                    cell.dataset.col = col;
                    cell.onclick = () => handleCellClick(row, col);
                    cell.ondragstart = (e) => handleDragStart(e, row, col);
                    cell.ondragover = (e) => e.preventDefault();
                    cell.ondrop = (e) => handleDrop(e, row, col);
                    board.appendChild(cell);
                }
            }
        }
        
        function handleDragStart(e, row, col) {
            if (boardState && boardState.cells[row][col].piece) {
                selectedCell = { row, col };
                e.target.classList.add('dragging');
            } else {
                e.preventDefault();
            }
        }
        
        function handleDrop(e, row, col) {
            e.preventDefault();
            document.querySelectorAll('.cell').forEach(c => c.classList.remove('dragging'));
            
            if (selectedCell) {
                makeMove(selectedCell.row, selectedCell.col, row, col);
                selectedCell = null;
                updateBoard();
            }
        }
        
        function handleCellClick(row, col) {
            if (selectedCell) {
                makeMove(selectedCell.row, selectedCell.col, row, col);
                selectedCell = null;
                updateBoard();
            } else {
                if (boardState && boardState.cells[row][col].piece) {
                    selectedCell = { row, col };
                    updateBoard();
                }
            }
        }
        
        async function makeMove(fromRow, fromCol, toRow, toCol) {
            try {
                const response = await fetch('/api/move', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ fromRow, fromCol, toRow, toCol })
                });
                
                if (response.ok) {
                    await loadState();
                    await loadStats();
                } else {
                    console.error('Invalid move');
                }
            } catch (error) {
                console.error('Error making move:', error);
            }
        }
        
        async function loadState() {
            try {
                const response = await fetch('/api/state');
                boardState = await response.json();
                updateBoard();
                updateStatus();
                updateStats(); // Update stats to show current epsilon and move count
            } catch (error) {
                console.error('Error loading state:', error);
            }
        }
        
        function updateBoard() {
            if (!boardState) return;
            
            const cells = document.querySelectorAll('.cell');
            cells.forEach(cell => {
                const row = parseInt(cell.dataset.row);
                const col = parseInt(cell.dataset.col);
                const piece = boardState.cells[row][col];
                
                cell.classList.remove('selected', 'valid-move');
                
                if (piece.piece && piece.color) {
                    cell.textContent = pieceSymbols[piece.color][piece.piece];
                    cell.draggable = true;
                } else {
                    cell.textContent = '';
                    cell.draggable = false;
                }
                
                if (selectedCell && selectedCell.row === row && selectedCell.col === col) {
                    cell.classList.add('selected');
                }
            });
        }
        
        function updateStatus() {
            if (!boardState) return;
            
            const statusBar = document.getElementById('statusBar');
            
            if (boardState.gameOver) {
                if (boardState.winner === 'white') {
                    statusBar.textContent = '🏆 White Wins!';
                } else if (boardState.winner === 'black') {
                    statusBar.textContent = '🏆 Black Wins!';
                } else {
                    statusBar.textContent = '🤝 Draw!';
                }
            } else {
                let turnText = (boardState.currentTurn === 'white' ? 'White' : 'Black') + "'s Turn";
                if (boardState.isCheck) {
                    turnText += ' - ♔ Check!';
                }
                // Add AI thinking indicator
                if (boardState.currentTurn === 'black') {
                    turnText += ' 🤔 AI is thinking...';
                }
                statusBar.textContent = turnText;
            }
        }
        
        async function resetGame() {
            try {
                await fetch('/api/reset', { method: 'POST' });
                selectedCell = null;
                await loadState();
            } catch (error) {
                console.error('Error resetting game:', error);
            }
        }
        
        async function loadStats() {
            try {
                const response = await fetch('/api/stats');
                statsData = await response.json();
                updateStats();
                drawChart();
            } catch (error) {
                console.error('Error loading stats:', error);
            }
        }
        
        function updateStats() {
            const totalGames = statsData.length;
            let aiWins = 0;
            let playerWins = 0;
            let draws = 0;
            
            statsData.forEach(game => {
                if (game.winner === 'black') aiWins++;
                else if (game.winner === 'white') playerWins++;
                else draws++;
            });
            
            const winRate = totalGames > 0 ? ((playerWins / totalGames) * 100).toFixed(1) : 0;
            
            document.getElementById('totalGames').textContent = totalGames;
            document.getElementById('aiWins').textContent = aiWins;
            document.getElementById('playerWins').textContent = playerWins;
            document.getElementById('draws').textContent = draws;
            document.getElementById('winRate').textContent = winRate + '%';
            
            // Update current epsilon and moves count from board state
            if (boardState) {
                const epsilon = boardState.epsilon !== undefined ? boardState.epsilon.toFixed(4) : '-';
                const moves = boardState.movesCount !== undefined ? boardState.movesCount : 0;
                document.getElementById('currentEpsilon').textContent = epsilon;
                document.getElementById('movesCount').textContent = moves;
            }
        }
        
        async function startSelfPlay() {
            try {
                const response = await fetch('/api/selfplay/start', { method: 'POST' });
                if (response.ok) {
                    document.getElementById('startSelfPlay').style.display = 'none';
                    document.getElementById('stopSelfPlay').style.display = 'block';
                    document.getElementById('selfPlayStatus').textContent = 'Running';
                    document.getElementById('selfPlayStatus').style.color = '#4CAF50';
                }
            } catch (error) {
                console.error('Error starting self-play:', error);
            }
        }
        
        async function stopSelfPlay() {
            try {
                const response = await fetch('/api/selfplay/stop', { method: 'POST' });
                if (response.ok) {
                    document.getElementById('startSelfPlay').style.display = 'block';
                    document.getElementById('stopSelfPlay').style.display = 'none';
                    document.getElementById('selfPlayStatus').textContent = 'Stopped';
                    document.getElementById('selfPlayStatus').style.color = '#F44336';
                }
            } catch (error) {
                console.error('Error stopping self-play:', error);
            }
        }
        
        async function checkSelfPlayStatus() {
            try {
                const response = await fetch('/api/selfplay/status');
                const status = await response.json();
                if (status.running) {
                    document.getElementById('startSelfPlay').style.display = 'none';
                    document.getElementById('stopSelfPlay').style.display = 'block';
                    document.getElementById('selfPlayStatus').textContent = 'Running';
                    document.getElementById('selfPlayStatus').style.color = '#4CAF50';
                } else {
                    document.getElementById('startSelfPlay').style.display = 'block';
                    document.getElementById('stopSelfPlay').style.display = 'none';
                    document.getElementById('selfPlayStatus').textContent = 'Stopped';
                    document.getElementById('selfPlayStatus').style.color = '#F44336';
                }
            } catch (error) {
                console.error('Error checking self-play status:', error);
            }
        }
        
        function drawChart() {
            const canvas = document.getElementById('progressChart');
            const ctx = canvas.getContext('2d');
            const width = canvas.width;
            const height = canvas.height;
            
            // Clear canvas
            ctx.fillStyle = 'white';
            ctx.fillRect(0, 0, width, height);
            
            if (statsData.length === 0) {
                ctx.fillStyle = '#999';
                ctx.font = '16px Arial';
                ctx.textAlign = 'center';
                ctx.fillText('No data yet', width / 2, height / 2);
                return;
            }
            
            const padding = 50;
            const chartWidth = width - 2 * padding;
            const chartHeight = height - 2 * padding;
            
            // Draw axes
            ctx.strokeStyle = '#333';
            ctx.lineWidth = 2;
            ctx.beginPath();
            ctx.moveTo(padding, padding);
            ctx.lineTo(padding, height - padding);
            ctx.lineTo(width - padding, height - padding);
            ctx.stroke();
            
            // Draw labels
            ctx.fillStyle = '#333';
            ctx.font = '12px Arial';
            ctx.textAlign = 'center';
            ctx.fillText('Game Number', width / 2, height - 10);
            
            ctx.save();
            ctx.translate(15, height / 2);
            ctx.rotate(-Math.PI / 2);
            ctx.fillText('Epsilon / Result', 0, 0);
            ctx.restore();
            
            // Draw epsilon line
            if (statsData.length > 1) {
                ctx.strokeStyle = '#2196F3';
                ctx.lineWidth = 2;
                ctx.beginPath();
                
                statsData.forEach((game, i) => {
                    const x = padding + (i / (statsData.length - 1)) * chartWidth;
                    const y = height - padding - (game.epsilon * chartHeight);
                    
                    if (i === 0) {
                        ctx.moveTo(x, y);
                    } else {
                        ctx.lineTo(x, y);
                    }
                });
                
                ctx.stroke();
            }
            
            // Draw game results as dots
            statsData.forEach((game, i) => {
                const x = padding + (i / Math.max(statsData.length - 1, 1)) * chartWidth;
                const y = height - padding - (game.epsilon * chartHeight);
                
                ctx.beginPath();
                ctx.arc(x, y, 4, 0, 2 * Math.PI);
                
                if (game.winner === 'black') {
                    ctx.fillStyle = '#4CAF50'; // Green for AI win
                } else if (game.winner === 'white') {
                    ctx.fillStyle = '#F44336'; // Red for player win
                } else {
                    ctx.fillStyle = '#FF9800'; // Orange for draw
                }
                
                ctx.fill();
                ctx.strokeStyle = '#333';
                ctx.lineWidth = 1;
                ctx.stroke();
            });
            
            // Draw Y-axis labels
            ctx.fillStyle = '#333';
            ctx.font = '10px Arial';
            ctx.textAlign = 'right';
            for (let i = 0; i <= 10; i++) {
                const y = height - padding - (i / 10) * chartHeight;
                ctx.fillText((i / 10).toFixed(1), padding - 5, y + 3);
                
                // Grid lines
                ctx.strokeStyle = '#eee';
                ctx.lineWidth = 1;
                ctx.beginPath();
                ctx.moveTo(padding, y);
                ctx.lineTo(width - padding, y);
                ctx.stroke();
            }
            
            // Draw X-axis labels
            ctx.textAlign = 'center';
            const step = Math.max(1, Math.floor(statsData.length / 10));
            for (let i = 0; i < statsData.length; i += step) {
                const x = padding + (i / Math.max(statsData.length - 1, 1)) * chartWidth;
                ctx.fillText(String(i + 1), x, height - padding + 15);
            }
            
            // Draw legend
            const legendX = width - padding - 100;
            const legendY = padding + 20;
            
            ctx.font = 'bold 12px Arial';
            ctx.fillStyle = '#333';
            ctx.textAlign = 'left';
            ctx.fillText('Legend:', legendX, legendY);
            
            // AI Win
            ctx.beginPath();
            ctx.arc(legendX + 10, legendY + 20, 5, 0, 2 * Math.PI);
            ctx.fillStyle = '#4CAF50';
            ctx.fill();
            ctx.fillStyle = '#333';
            ctx.font = '11px Arial';
            ctx.fillText('AI Win', legendX + 20, legendY + 24);
            
            // Player Win
            ctx.beginPath();
            ctx.arc(legendX + 10, legendY + 40, 5, 0, 2 * Math.PI);
            ctx.fillStyle = '#F44336';
            ctx.fill();
            ctx.fillStyle = '#333';
            ctx.fillText('Player Win', legendX + 20, legendY + 44);
            
            // Draw
            ctx.beginPath();
            ctx.arc(legendX + 10, legendY + 60, 5, 0, 2 * Math.PI);
            ctx.fillStyle = '#FF9800';
            ctx.fill();
            ctx.fillStyle = '#333';
            ctx.fillText('Draw', legendX + 20, legendY + 64);
            
            // Epsilon line
            ctx.strokeStyle = '#2196F3';
            ctx.lineWidth = 2;
            ctx.beginPath();
            ctx.moveTo(legendX, legendY + 80);
            ctx.lineTo(legendX + 15, legendY + 80);
            ctx.stroke();
            ctx.fillStyle = '#333';
            ctx.fillText('Epsilon', legendX + 20, legendY + 84);
        }
        
        // Initialize
        createBoard();
        loadState();
        loadStats();
        checkSelfPlayStatus();
        
        // Poll for state changes (AI moves and game updates)
        setInterval(async () => {
            try {
                if (boardState) {
                    const prevTurn = boardState.currentTurn;
                    const prevGameOver = boardState.gameOver;
                    
                    await loadState();
                    
                    // If turn changed or game ended, reload stats
                    if (boardState && (boardState.currentTurn !== prevTurn || boardState.gameOver !== prevGameOver)) {
                        await loadStats();
                    }
                }
                
                // Check self-play status periodically
                await checkSelfPlayStatus();
            } catch (error) {
                console.error('Error polling state:', error);
            }
        }, 500); // Poll every 500ms for responsive UI
    </script>
</body>
</html>`
