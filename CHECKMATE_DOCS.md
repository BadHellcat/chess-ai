# Проверка шаха и мата - Документация

## Обзор

Реализована полная система проверки шаха и мата, соответствующая правилам FIDE.

## Новые методы

### `findKing(color Color) Position`
Находит позицию короля указанного цвета на доске.

```go
kingPos := board.findKing(game.White)
// Возвращает Position{Row: 7, Col: 4} для белого короля в начальной позиции
```

### `isSquareUnderAttack(pos Position, byColor Color) bool`
Проверяет, атакована ли указанная клетка фигурами определённого цвета.

```go
// Проверить, атакована ли клетка e4 чёрными фигурами
underAttack := board.isSquareUnderAttack(Position{Row: 4, Col: 4}, game.Black)
```

Особенности:
- Для пешек проверяются только диагональные атаки
- Для остальных фигур используется базовая валидация движения
- Не вызывает рекурсию при проверке короля

### `isInCheck(color Color) bool`
Определяет, находится ли король указанного цвета под шахом.

```go
if board.isInCheck(game.White) {
    fmt.Println("Белый король под шахом!")
}
```

### `wouldBeInCheck(move Move, color Color) bool`
Симулирует ход и проверяет, приведёт ли он к шаху своему королю.

```go
move := Move{From: Position{7, 4}, To: Position{6, 5}}
if board.wouldBeInCheck(move, game.White) {
    fmt.Println("Этот ход оставит короля под шахом")
}
```

## Обновлённые методы

### `IsValidMove(move Move) bool`
Теперь проверяет:
1. Базовые правила движения фигуры
2. Не оставляет ли ход короля под шахом

```go
// Этот ход будет заблокирован если оставляет короля под шахом
if board.IsValidMove(move) {
    board.MakeMove(move)
}
```

### `canCastle(move Move, piece Piece) bool`
Проверяет все условия рокировки:
1. Король не двигался
2. Ладья не двигалась
3. Путь свободен
4. **Король не под шахом**
5. **Король не проходит через шах**
6. **Король не попадает под шах**

### `checkGameOver()`
Правильно определяет:
- **Мат**: Нет легальных ходов И король под шахом → Противник выигрывает
- **Пат**: Нет легальных ходов И король НЕ под шахом → Ничья

```go
if len(moves) == 0 {
    if board.isInCheck(board.CurrentTurn) {
        // МАТ - противник выигрывает
        board.Winner = opponent
    } else {
        // ПАТ - ничья
        board.Winner = White // используется как индикатор ничьей
    }
}
```

### `MakeMove(move Move)`
После каждого хода обновляет флаг `IsCheck`:

```go
board.IsCheck = board.isInCheck(board.CurrentTurn)
```

### `Clone() *Board`
Копирует поле `IsCheck` для корректной симуляции ходов.

## Поле IsCheck

Новое поле в структуре `Board`:

```go
type Board struct {
    // ...
    IsCheck bool // Находится ли текущий игрок под шахом
    // ...
}
```

## Примеры использования

### Пример 1: Fool's Mate

```go
board := game.NewBoard()

board.MakeMove(Move{From: Position{6, 5}, To: Position{5, 5}}) // f2-f3
board.MakeMove(Move{From: Position{1, 4}, To: Position{2, 4}}) // e7-e6
board.MakeMove(Move{From: Position{6, 6}, To: Position{4, 6}}) // g2-g4
board.MakeMove(Move{From: Position{0, 3}, To: Position{4, 7}}) // Qd8-h4#

// board.IsCheck == true
// board.GameOver == true
// board.Winner == game.Black
```

### Пример 2: Заблокированный ход

```go
board := game.NewBoard()
// ... настройка позиции где f2 атакована Bc4

move := Move{From: Position{7, 4}, To: Position{6, 5}} // Ke1-f2

if !board.IsValidMove(move) {
    fmt.Println("Нельзя ходить под шах!")
}
```

### Пример 3: Закреплённая фигура

```go
// Если фигура закреплена (её движение откроет короля для шаха),
// такой ход автоматически заблокируется

pinnedMove := Move{From: knightPos, To: newPos}

if !board.IsValidMove(pinnedMove) {
    fmt.Println("Фигура закреплена и не может двигаться")
}
```

## Веб API

Поле `isCheck` добавлено в JSON ответ:

```json
{
  "cells": [...],
  "currentTurn": "white",
  "gameOver": false,
  "winner": "",
  "isCheck": true
}
```

В веб-интерфейсе при шахе показывается:
```
White's Turn - ♔ Check!
```

## Производительность

- `findKing()`: O(64) = O(1) - константное время
- `isSquareUnderAttack()`: O(64 × типов_ходов) ≈ O(1)
- `isInCheck()`: O(1)
- `wouldBeInCheck()`: O(1) + время клонирования доски
- `IsValidMove()`: Добавляет одну проверку O(1)

## Соответствие FIDE

✅ Шах должен быть устранён следующим ходом  
✅ Нельзя ходить под шах  
✅ Нельзя делать ход, открывающий своего короля для шаха  
✅ Рокировка запрещена при/через/в шах  
✅ Мат = шах + нет легальных ходов  
✅ Пат = не шах + нет легальных ходов (ничья)

## Тестирование

Запустите демонстрацию:

```bash
cd /home/runner/work/chess-ai/chess-ai
go run demo_check.go
```

Или используйте веб-интерфейс:

```bash
./chess-ai
# Откройте http://localhost:8080
```

Или терминальный режим:

```bash
./chess-ai --terminal
```
