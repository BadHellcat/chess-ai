package neural

import (
	"encoding/gob"
	"math"
	"math/rand"
	"os"
)

// Network представляет нейронную сеть
type Network struct {
	Weights1 [][]float64 // 768 -> 256
	Bias1    []float64   // 256
	Weights2 [][]float64 // 256 -> 128
	Bias2    []float64   // 128
	Weights3 [][]float64 // 128 -> 1
	Bias3    []float64   // 1

	// Для momentum
	VWeights1 [][]float64
	VBias1    []float64
	VWeights2 [][]float64
	VBias2    []float64
	VWeights3 [][]float64
	VBias3    []float64

	LearningRate float64
	Momentum     float64
}

// NewNetwork создает новую нейронную сеть
func NewNetwork() *Network {
	n := &Network{
		LearningRate: 0.001,
		Momentum:     0.9,
	}

	// Инициализация весов (Xavier initialization)
	n.Weights1 = make([][]float64, 768)
	n.VWeights1 = make([][]float64, 768)
	for i := range n.Weights1 {
		n.Weights1[i] = make([]float64, 256)
		n.VWeights1[i] = make([]float64, 256)
		scale := math.Sqrt(2.0 / 768.0)
		for j := range n.Weights1[i] {
			n.Weights1[i][j] = (rand.Float64()*2 - 1) * scale
		}
	}

	n.Bias1 = make([]float64, 256)
	n.VBias1 = make([]float64, 256)

	n.Weights2 = make([][]float64, 256)
	n.VWeights2 = make([][]float64, 256)
	for i := range n.Weights2 {
		n.Weights2[i] = make([]float64, 128)
		n.VWeights2[i] = make([]float64, 128)
		scale := math.Sqrt(2.0 / 256.0)
		for j := range n.Weights2[i] {
			n.Weights2[i][j] = (rand.Float64()*2 - 1) * scale
		}
	}

	n.Bias2 = make([]float64, 128)
	n.VBias2 = make([]float64, 128)

	n.Weights3 = make([][]float64, 128)
	n.VWeights3 = make([][]float64, 128)
	for i := range n.Weights3 {
		n.Weights3[i] = make([]float64, 1)
		n.VWeights3[i] = make([]float64, 1)
		scale := math.Sqrt(2.0 / 128.0)
		for j := range n.Weights3[i] {
			n.Weights3[i][j] = (rand.Float64()*2 - 1) * scale
		}
	}

	n.Bias3 = make([]float64, 1)
	n.VBias3 = make([]float64, 1)

	// Попытка загрузить сохраненные веса
	// Сохраняем начальные значения LearningRate и Momentum
	initialLR := n.LearningRate
	initialMomentum := n.Momentum
	
	if err := n.Load(); err == nil {
		// Если загрузка успешна, проверяем и восстанавливаем LearningRate и Momentum,
		// если они были обнулены или имеют неразумные значения
		if n.LearningRate <= 0 || n.LearningRate > 1.0 {
			n.LearningRate = initialLR
		}
		if n.Momentum <= 0 || n.Momentum > 1.0 {
			n.Momentum = initialMomentum
		}
	}

	return n
}

// Forward выполняет прямое распространение
func (n *Network) Forward(input []float64) float64 {
	// Слой 1: 768 -> 256 с ReLU
	hidden1 := make([]float64, 256)
	for i := 0; i < 256; i++ {
		sum := n.Bias1[i]
		for j := 0; j < 768; j++ {
			sum += input[j] * n.Weights1[j][i]
		}
		hidden1[i] = relu(sum)
	}

	// Слой 2: 256 -> 128 с ReLU
	hidden2 := make([]float64, 128)
	for i := 0; i < 128; i++ {
		sum := n.Bias2[i]
		for j := 0; j < 256; j++ {
			sum += hidden1[j] * n.Weights2[j][i]
		}
		hidden2[i] = relu(sum)
	}

	// Выходной слой: 128 -> 1 с tanh
	sum := n.Bias3[0]
	for j := 0; j < 128; j++ {
		sum += hidden2[j] * n.Weights3[j][0]
	}

	return tanh(sum)
}

// Train обучает сеть на одном примере
func (n *Network) Train(input []float64, target float64) {
	// Forward pass с сохранением активаций
	hidden1 := make([]float64, 256)
	for i := 0; i < 256; i++ {
		sum := n.Bias1[i]
		for j := 0; j < 768; j++ {
			sum += input[j] * n.Weights1[j][i]
		}
		hidden1[i] = relu(sum)
	}

	hidden2 := make([]float64, 128)
	for i := 0; i < 128; i++ {
		sum := n.Bias2[i]
		for j := 0; j < 256; j++ {
			sum += hidden1[j] * n.Weights2[j][i]
		}
		hidden2[i] = relu(sum)
	}

	sum := n.Bias3[0]
	for j := 0; j < 128; j++ {
		sum += hidden2[j] * n.Weights3[j][0]
	}
	output := tanh(sum)

	// Backward pass
	// Ошибка выходного слоя
	outputError := target - output
	outputDelta := outputError * tanhDerivative(sum)

	// Обновление весов выходного слоя с momentum
	for j := 0; j < 128; j++ {
		grad := outputDelta * hidden2[j]
		n.VWeights3[j][0] = n.Momentum*n.VWeights3[j][0] + n.LearningRate*grad
		n.Weights3[j][0] += n.VWeights3[j][0]
	}
	n.VBias3[0] = n.Momentum*n.VBias3[0] + n.LearningRate*outputDelta
	n.Bias3[0] += n.VBias3[0]

	// Ошибка второго скрытого слоя
	hidden2Error := make([]float64, 128)
	for i := 0; i < 128; i++ {
		hidden2Error[i] = outputDelta * n.Weights3[i][0]
		if hidden2[i] <= 0 {
			hidden2Error[i] = 0 // ReLU derivative
		}
	}

	// Обновление весов второго слоя
	for i := 0; i < 256; i++ {
		for j := 0; j < 128; j++ {
			grad := hidden2Error[j] * hidden1[i]
			n.VWeights2[i][j] = n.Momentum*n.VWeights2[i][j] + n.LearningRate*grad
			n.Weights2[i][j] += n.VWeights2[i][j]
		}
	}
	for j := 0; j < 128; j++ {
		n.VBias2[j] = n.Momentum*n.VBias2[j] + n.LearningRate*hidden2Error[j]
		n.Bias2[j] += n.VBias2[j]
	}

	// Ошибка первого скрытого слоя
	hidden1Error := make([]float64, 256)
	for i := 0; i < 256; i++ {
		for j := 0; j < 128; j++ {
			hidden1Error[i] += hidden2Error[j] * n.Weights2[i][j]
		}
		if hidden1[i] <= 0 {
			hidden1Error[i] = 0 // ReLU derivative
		}
	}

	// Обновление весов первого слоя
	for i := 0; i < 768; i++ {
		for j := 0; j < 256; j++ {
			grad := hidden1Error[j] * input[i]
			n.VWeights1[i][j] = n.Momentum*n.VWeights1[i][j] + n.LearningRate*grad
			n.Weights1[i][j] += n.VWeights1[i][j]
		}
	}
	for j := 0; j < 256; j++ {
		n.VBias1[j] = n.Momentum*n.VBias1[j] + n.LearningRate*hidden1Error[j]
		n.Bias1[j] += n.VBias1[j]
	}
}

// Save сохраняет веса сети
func (n *Network) Save() error {
	os.MkdirAll("neural", 0755)
	file, err := os.Create("neural/weights.gob")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	return encoder.Encode(n)
}

// Load загружает веса сети
func (n *Network) Load() error {
	file, err := os.Open("neural/weights.gob")
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	return decoder.Decode(n)
}

// Активационные функции
func relu(x float64) float64 {
	if x > 0 {
		return x
	}
	return 0
}

func tanh(x float64) float64 {
	return math.Tanh(x)
}

func tanhDerivative(x float64) float64 {
	t := math.Tanh(x)
	return 1 - t*t
}
