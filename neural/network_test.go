package neural

import (
	"math"
	"testing"
)

// TestNewNetwork проверяет создание новой нейронной сети
func TestNewNetwork(t *testing.T) {
	network := NewNetwork()

	if network == nil {
		t.Fatal("NewNetwork() returned nil")
	}

	// Проверяем размерности слоев
	if len(network.Weights1) != 768 {
		t.Errorf("Expected Weights1 to have 768 rows, got %d", len(network.Weights1))
	}

	if len(network.Weights1[0]) != 256 {
		t.Errorf("Expected Weights1 to have 256 columns, got %d", len(network.Weights1[0]))
	}

	if len(network.Bias1) != 256 {
		t.Errorf("Expected Bias1 to have 256 elements, got %d", len(network.Bias1))
	}

	if len(network.Weights2) != 256 {
		t.Errorf("Expected Weights2 to have 256 rows, got %d", len(network.Weights2))
	}

	if len(network.Weights2[0]) != 128 {
		t.Errorf("Expected Weights2 to have 128 columns, got %d", len(network.Weights2[0]))
	}

	if len(network.Bias2) != 128 {
		t.Errorf("Expected Bias2 to have 128 elements, got %d", len(network.Bias2))
	}

	if len(network.Weights3) != 128 {
		t.Errorf("Expected Weights3 to have 128 rows, got %d", len(network.Weights3))
	}

	if len(network.Weights3[0]) != 1 {
		t.Errorf("Expected Weights3 to have 1 column, got %d", len(network.Weights3[0]))
	}

	if len(network.Bias3) != 1 {
		t.Errorf("Expected Bias3 to have 1 element, got %d", len(network.Bias3))
	}

	// Проверяем параметры обучения
	if network.LearningRate != 0.001 {
		t.Errorf("Expected LearningRate to be 0.001, got %f", network.LearningRate)
	}

	if network.Momentum != 0.9 {
		t.Errorf("Expected Momentum to be 0.9, got %f", network.Momentum)
	}
}

// TestForward проверяет прямое распространение
func TestForward(t *testing.T) {
	network := NewNetwork()

	// Создаем входной вектор размера 768
	input := make([]float64, 768)
	for i := range input {
		input[i] = float64(i%2) * 0.5 // Простой паттерн
	}

	output := network.Forward(input)

	// Проверяем, что вывод находится в диапазоне [-1, 1] (tanh)
	if output < -1.0 || output > 1.0 {
		t.Errorf("Expected output in range [-1, 1], got %f", output)
	}
}

// TestTrain проверяет обучение сети
func TestTrain(t *testing.T) {
	network := NewNetwork()

	input := make([]float64, 768)
	for i := range input {
		input[i] = 0.1
	}

	// Запоминаем исходные веса
	originalWeight := network.Weights1[0][0]

	// Прямое распространение и обучение
	network.Train(input, 1.0)

	// Проверяем, что веса изменились
	if network.Weights1[0][0] == originalWeight {
		t.Error("Weights should change after training")
	}
}

// TestSaveLoad проверяет сохранение и загрузку сети
func TestSaveLoad(t *testing.T) {
	network := NewNetwork()

	// Устанавливаем известные значения
	network.Weights1[0][0] = 0.12345
	network.Bias1[0] = 0.67890

	// Сохраняем
	err := network.Save()
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Создаем новую сеть и загружаем
	newNetwork := NewNetwork()
	err = newNetwork.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Проверяем, что значения совпадают
	if math.Abs(newNetwork.Weights1[0][0]-0.12345) > 0.00001 {
		t.Errorf("Expected Weights1[0][0] to be 0.12345, got %f", newNetwork.Weights1[0][0])
	}

	if math.Abs(newNetwork.Bias1[0]-0.67890) > 0.00001 {
		t.Errorf("Expected Bias1[0] to be 0.67890, got %f", newNetwork.Bias1[0])
	}
}

// TestReLU проверяет функцию активации ReLU
func TestReLU(t *testing.T) {
	testCases := []struct {
		input    float64
		expected float64
	}{
		{input: -1.0, expected: 0.0},
		{input: 0.0, expected: 0.0},
		{input: 1.0, expected: 1.0},
		{input: 0.5, expected: 0.5},
		{input: -0.5, expected: 0.0},
	}

	for _, tc := range testCases {
		result := relu(tc.input)
		if result != tc.expected {
			t.Errorf("relu(%f) = %f, expected %f", tc.input, result, tc.expected)
		}
	}
}

// TestTanh проверяет функцию активации tanh
func TestTanh(t *testing.T) {
	testCases := []struct {
		input float64
		min   float64
		max   float64
	}{
		{input: -10.0, min: -1.0, max: -0.99},
		{input: 0.0, min: -0.01, max: 0.01},
		{input: 10.0, min: 0.99, max: 1.0},
	}

	for _, tc := range testCases {
		result := tanh(tc.input)
		if result < tc.min || result > tc.max {
			t.Errorf("tanh(%f) = %f, expected in range [%f, %f]", tc.input, result, tc.min, tc.max)
		}
	}
}
