package neural

// TrainBatch обучает сеть на пакете данных
func (n *Network) TrainBatch(inputs [][]float64, targets []float64) {
	for i := range inputs {
		n.Train(inputs[i], targets[i])
	}
}

// Evaluate оценивает точность сети
func (n *Network) Evaluate(inputs [][]float64, targets []float64) float64 {
	if len(inputs) == 0 {
		return 0
	}

	correct := 0
	for i := range inputs {
		output := n.Forward(inputs[i])
		predicted := 0.0
		if output > 0 {
			predicted = 1.0
		}
		if (predicted > 0.5 && targets[i] > 0.5) || (predicted <= 0.5 && targets[i] <= 0.5) {
			correct++
		}
	}

	return float64(correct) / float64(len(inputs))
}
