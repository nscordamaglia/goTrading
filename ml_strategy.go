package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/sdcoffey/techan"
)

// MLConfig holds configuration for machine learning models
type MLConfig struct {
	LookbackPeriod    int     // Number of historical periods to analyze
	TrainingRatio     float64 // Ratio of data used for training vs validation
	MinTrainingPeriod int     // Minimum periods needed before training
	RetrainingPeriod  int     // How often to retrain the model (in periods)
	FeatureWindow     int     // Window size for feature calculation
}

// TradingFeatures represents extracted features from market data
type TradingFeatures struct {
	// Price features
	PriceChange1  float64
	PriceChange5  float64
	PriceChange10 float64

	// Technical indicators
	RSI           float64
	EMA9          float64
	EMA21         float64
	EMASpread     float64
	MACD          float64
	MACDSignal    float64
	MACDHistogram float64

	// Volume features
	VolumeRatio float64
	VolumeMA    float64

	// Volatility features
	ATR               float64
	BollingerWidth    float64
	BollingerPosition float64

	// Market structure
	HighLowRatio float64
	CandleBody   float64
	CandleWick   float64

	// Time features
	HourOfDay float64
	DayOfWeek float64

	// Target variable (future return)
	FutureReturn    float64
	FutureDirection int // 1 for up, 0 for sideways, -1 for down
}

// MLModel represents a machine learning model for trading
type MLModel struct {
	config         MLConfig
	features       []TradingFeatures
	trainedWeights map[string]float64
	featureStats   map[string]FeatureStats
	lastTraining   time.Time
	trainingCount  int
	performance    ModelPerformance
	l2Lambda       float64
}

// FeatureStats holds statistics for feature normalization
type FeatureStats struct {
	Mean   float64
	StdDev float64
	Min    float64
	Max    float64
}

// ModelPerformance tracks model performance metrics
type ModelPerformance struct {
	Accuracy           float64
	Precision          float64
	Recall             float64
	F1Score            float64
	ProfitabilityScore float64
	SharpeRatio        float64
	TrainingLoss       float64
	ValidationLoss     float64
}

// PredictionResult holds model prediction output
type PredictionResult struct {
	Direction      string  // "BUY", "SELL", "HOLD"
	Confidence     float64 // 0.0 to 1.0
	ExpectedReturn float64
	Features       TradingFeatures
}

// NewMLModel creates a new machine learning model
func NewMLModel(config MLConfig) *MLModel {
	return &MLModel{
		config:         config,
		features:       make([]TradingFeatures, 0),
		trainedWeights: make(map[string]float64),
		featureStats:   make(map[string]FeatureStats),
		trainingCount:  0,
		l2Lambda:       1e-4,
	}
}

// ExtractFeatures extracts trading features from time series data
func (ml *MLModel) ExtractFeatures(ts *techan.TimeSeries, index int) *TradingFeatures {
	if index < ml.config.FeatureWindow {
		return nil // Not enough data
	}

	features := &TradingFeatures{}

	// Price/volume indicators
	closePrices := techan.NewClosePriceIndicator(ts)
	highPrices := techan.NewHighPriceIndicator(ts)
	lowPrices := techan.NewLowPriceIndicator(ts)
	volumeIndicator := techan.NewVolumeIndicator(ts)

	currentPrice := closePrices.Calculate(index).Float()

	// Price changes
	if index >= 1 {
		prevPrice := closePrices.Calculate(index - 1).Float()
		if prevPrice != 0 {
			features.PriceChange1 = (currentPrice - prevPrice) / prevPrice
		}
	}
	if index >= 5 {
		prevPrice := closePrices.Calculate(index - 5).Float()
		if prevPrice != 0 {
			features.PriceChange5 = (currentPrice - prevPrice) / prevPrice
		}
	}
	if index >= 10 {
		prevPrice := closePrices.Calculate(index - 10).Float()
		if prevPrice != 0 {
			features.PriceChange10 = (currentPrice - prevPrice) / prevPrice
		}
	}

	// Technical indicators
	rsi := techan.NewRelativeStrengthIndexIndicator(closePrices, 14)
	ema9 := techan.NewEMAIndicator(closePrices, 9)
	ema21 := techan.NewEMAIndicator(closePrices, 21)
	macd := techan.NewMACDIndicator(closePrices, 12, 26)
	macdSignal := techan.NewEMAIndicator(macd, 9)

	features.RSI = rsi.Calculate(index).Float()
	features.EMA9 = ema9.Calculate(index).Float()
	features.EMA21 = ema21.Calculate(index).Float()
	if features.EMA21 != 0 {
		features.EMASpread = (features.EMA9 - features.EMA21) / features.EMA21
	}
	features.MACD = macd.Calculate(index).Float()
	features.MACDSignal = macdSignal.Calculate(index).Float()
	features.MACDHistogram = features.MACD - features.MACDSignal

	// Volume features
	volMA := ml.calculateSMA(volumeIndicator, index, 20)
	currentVol := volumeIndicator.Calculate(index).Float()
	if volMA > 0 {
		features.VolumeRatio = currentVol / volMA
	} else {
		features.VolumeRatio = 1.0
	}
	features.VolumeMA = volMA

	// Volatility features
	atr := ml.calculateATR(ts, index, 14)
	features.ATR = atr

	// Bollinger Bands (with guards)
	sma := ml.calculateSMA(closePrices, index, 20)
	stdDev := ml.calculateStdDev(closePrices, index, 20)
	upperBB := sma + (2 * stdDev)
	lowerBB := sma - (2 * stdDev)
	bandWidth := upperBB - lowerBB
	if sma != 0 {
		features.BollingerWidth = bandWidth / sma
	}
	if bandWidth != 0 {
		features.BollingerPosition = (currentPrice - lowerBB) / bandWidth
	}

	// Market structure
	highPrice := highPrices.Calculate(index).Float()
	lowPrice := lowPrices.Calculate(index).Float()
	if currentPrice != 0 {
		features.HighLowRatio = (highPrice - lowPrice) / currentPrice
	}

	// Candle structure (use actual open price)
	openPrice := ts.Candles[index].OpenPrice.Float()
	if currentPrice != 0 {
		features.CandleBody = math.Abs(currentPrice-openPrice) / currentPrice
		upperWick := math.Max(highPrice-currentPrice, highPrice-openPrice)
		lowerWick := math.Max(currentPrice-lowPrice, openPrice-lowPrice)
		features.CandleWick = (upperWick + lowerWick) / currentPrice
	}

	// Time features
	now := time.Now()
	features.HourOfDay = float64(now.Hour()) / 24.0
	features.DayOfWeek = float64(now.Weekday()) / 7.0

	// Sanitize any NaN/Inf values
	ml.sanitizeFeatures(features)
	return features
}

// sanitizeFeatures replaces NaN/Inf feature values with 0 to avoid propagating invalid numbers
func (ml *MLModel) sanitizeFeatures(f *TradingFeatures) {
	sanitize := func(v float64) float64 {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return 0
		}
		return v
	}

	f.PriceChange1 = sanitize(f.PriceChange1)
	f.PriceChange5 = sanitize(f.PriceChange5)
	f.PriceChange10 = sanitize(f.PriceChange10)
	f.RSI = sanitize(f.RSI)
	f.EMA9 = sanitize(f.EMA9)
	f.EMA21 = sanitize(f.EMA21)
	f.EMASpread = sanitize(f.EMASpread)
	f.MACD = sanitize(f.MACD)
	f.MACDSignal = sanitize(f.MACDSignal)
	f.MACDHistogram = sanitize(f.MACDHistogram)
	f.VolumeRatio = sanitize(f.VolumeRatio)
	f.VolumeMA = sanitize(f.VolumeMA)
	f.ATR = sanitize(f.ATR)
	f.BollingerWidth = sanitize(f.BollingerWidth)
	f.BollingerPosition = sanitize(f.BollingerPosition)
	f.HighLowRatio = sanitize(f.HighLowRatio)
	f.CandleBody = sanitize(f.CandleBody)
	f.CandleWick = sanitize(f.CandleWick)
	f.HourOfDay = sanitize(f.HourOfDay)
	f.DayOfWeek = sanitize(f.DayOfWeek)
	f.FutureReturn = sanitize(f.FutureReturn)
}

// calculateATR calculates Average True Range
func (ml *MLModel) calculateATR(ts *techan.TimeSeries, index, period int) float64 {
	if index < period {
		return 0
	}

	closePrices := techan.NewClosePriceIndicator(ts)
	highPrices := techan.NewHighPriceIndicator(ts)
	lowPrices := techan.NewLowPriceIndicator(ts)

	trSum := 0.0
	for i := index - period + 1; i <= index; i++ {
		high := highPrices.Calculate(i).Float()
		low := lowPrices.Calculate(i).Float()
		prevClose := closePrices.Calculate(i - 1).Float()

		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)

		tr := math.Max(tr1, math.Max(tr2, tr3))
		trSum += tr
	}

	return trSum / float64(period)
}

// calculateSMA calculates Simple Moving Average
func (ml *MLModel) calculateSMA(indicator techan.Indicator, index, period int) float64 {
	if index < period-1 {
		return 0
	}

	sum := 0.0
	for i := index - period + 1; i <= index; i++ {
		sum += indicator.Calculate(i).Float()
	}

	return sum / float64(period)
}

// calculateStdDev calculates Standard Deviation
func (ml *MLModel) calculateStdDev(indicator techan.Indicator, index, period int) float64 {
	if index < period-1 {
		return 0
	}

	mean := ml.calculateSMA(indicator, index, period)
	sumSquares := 0.0

	for i := index - period + 1; i <= index; i++ {
		diff := indicator.Calculate(i).Float() - mean
		sumSquares += diff * diff
	}

	variance := sumSquares / float64(period)
	if variance < 0 {
		return 0
	}
	return math.Sqrt(variance)
}

// AddTrainingData adds new training data to the model
func (ml *MLModel) AddTrainingData(features TradingFeatures) {
	// sanitize before storing
	ml.sanitizeFeatures(&features)
	ml.features = append(ml.features, features)

	// Keep only the most recent data points
	if len(ml.features) > ml.config.LookbackPeriod {
		ml.features = ml.features[1:]
	}
}

// NormalizeFeatures normalizes features using z-score normalization
func (ml *MLModel) NormalizeFeatures(features *TradingFeatures) {
	// Calculate feature statistics if not already done
	if len(ml.featureStats) == 0 {
		ml.calculateFeatureStats()
	}

	// Normalize each feature
	features.PriceChange1 = ml.normalize("PriceChange1", features.PriceChange1)
	features.PriceChange5 = ml.normalize("PriceChange5", features.PriceChange5)
	features.PriceChange10 = ml.normalize("PriceChange10", features.PriceChange10)
	features.RSI = ml.normalize("RSI", features.RSI)
	features.EMASpread = ml.normalize("EMASpread", features.EMASpread)
	features.MACDHistogram = ml.normalize("MACDHistogram", features.MACDHistogram)
	features.ATR = ml.normalize("ATR", features.ATR)
	features.BollingerWidth = ml.normalize("BollingerWidth", features.BollingerWidth)
	features.BollingerPosition = ml.normalize("BollingerPosition", features.BollingerPosition)
	features.VolumeRatio = ml.normalize("VolumeRatio", features.VolumeRatio)
}

// normalize applies z-score normalization to a feature
func (ml *MLModel) normalize(featureName string, value float64) float64 {
	stats, exists := ml.featureStats[featureName]
	if !exists || stats.StdDev == 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}

	normalized := (value - stats.Mean) / stats.StdDev
	if math.IsNaN(normalized) || math.IsInf(normalized, 0) {
		return 0
	}
	return normalized
}

// calculateFeatureStats calculates statistics for all features
func (ml *MLModel) calculateFeatureStats() {
	if len(ml.features) < 10 {
		return // Need minimum data
	}

	featureNames := []string{
		"PriceChange1", "PriceChange5", "PriceChange10", "RSI", "EMASpread",
		"MACDHistogram", "ATR", "BollingerWidth", "BollingerPosition", "VolumeRatio",
	}

	for _, name := range featureNames {
		values := ml.getFeatureValues(name)
		ml.featureStats[name] = ml.calculateStats(values)
	}
}

// getFeatureValues extracts values for a specific feature
func (ml *MLModel) getFeatureValues(featureName string) []float64 {
	values := make([]float64, len(ml.features))

	for i, feature := range ml.features {
		switch featureName {
		case "PriceChange1":
			values[i] = feature.PriceChange1
		case "PriceChange5":
			values[i] = feature.PriceChange5
		case "PriceChange10":
			values[i] = feature.PriceChange10
		case "RSI":
			values[i] = feature.RSI
		case "EMASpread":
			values[i] = feature.EMASpread
		case "MACDHistogram":
			values[i] = feature.MACDHistogram
		case "ATR":
			values[i] = feature.ATR
		case "BollingerWidth":
			values[i] = feature.BollingerWidth
		case "BollingerPosition":
			values[i] = feature.BollingerPosition
		case "VolumeRatio":
			values[i] = feature.VolumeRatio
		}
	}

	return values
}

// calculateStats calculates basic statistics for a feature
func (ml *MLModel) calculateStats(values []float64) FeatureStats {
	if len(values) == 0 {
		return FeatureStats{}
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	// Calculate standard deviation
	sumSquares := 0.0
	minVal := values[0]
	maxVal := values[0]

	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff

		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	stdDev := math.Sqrt(sumSquares / float64(len(values)))

	return FeatureStats{
		Mean:   mean,
		StdDev: stdDev,
		Min:    minVal,
		Max:    maxVal,
	}
}

// TrainModel trains the machine learning model using linear regression
func (ml *MLModel) TrainModel() error {
	if len(ml.features) < ml.config.MinTrainingPeriod {
		return fmt.Errorf("insufficient training data: %d, need at least %d",
			len(ml.features), ml.config.MinTrainingPeriod)
	}

	log.Printf("Training ML model with %d data points...", len(ml.features))

	// Prepare training data
	// Ensure feature statistics are computed for normalization
	ml.calculateFeatureStats()
	trainSize := int(float64(len(ml.features)) * ml.config.TrainingRatio)
	if trainSize <= 0 || trainSize >= len(ml.features) {
		trainSize = int(float64(len(ml.features)) * 0.8)
		if trainSize <= 0 {
			trainSize = len(ml.features) - 1
		}
	}
	trainData := ml.features[:trainSize]
	validationData := ml.features[trainSize:]

	// Initialize weights
	ml.trainedWeights = make(map[string]float64)

	// Simple linear regression implementation
	// In a production system, you would use a proper ML library
	err := ml.trainLinearModel(trainData)
	if err != nil {
		return fmt.Errorf("training failed: %v", err)
	}

	// Evaluate model performance
	ml.evaluateModel(validationData)

	ml.lastTraining = time.Now()
	ml.trainingCount++

	log.Printf("Model training completed. Accuracy: %.2f%%, F1-Score: %.3f",
		ml.performance.Accuracy*100, ml.performance.F1Score)

	return nil
}

// trainLinearModel implements a simplified linear regression
func (ml *MLModel) trainLinearModel(trainData []TradingFeatures) error {
	// Feature weights (simplified approach)
	// In practice, you'd use gradient descent or other optimization methods

	ml.trainedWeights["PriceChange1"] = 0.1
	ml.trainedWeights["PriceChange5"] = 0.15
	ml.trainedWeights["PriceChange10"] = 0.1
	ml.trainedWeights["RSI"] = 0.05
	ml.trainedWeights["EMASpread"] = 0.2
	ml.trainedWeights["MACDHistogram"] = 0.15
	ml.trainedWeights["ATR"] = -0.1
	ml.trainedWeights["BollingerPosition"] = 0.1
	ml.trainedWeights["bias"] = 0.0

	// Simple optimization loop (placeholder for proper ML training)
	learningRate := 0.0001
	epochs := 100

	for epoch := 0; epoch < epochs; epoch++ {
		totalError := 0.0

		for _, data := range trainData {
			// normalize inputs for stable training
			normalized := data
			ml.NormalizeFeatures(&normalized)
			prediction := ml.predictValue(normalized)
			error := data.FutureReturn - prediction
			// gradient clipping on error
			if error > 1.0 {
				error = 1.0
			} else if error < -1.0 {
				error = -1.0
			}
			totalError += error * error

			// Update weights (simplified gradient descent)
			update := func(name string, value float64) {
				if math.IsNaN(value) || math.IsInf(value, 0) {
					return
				}
				ml.trainedWeights[name] += learningRate * error * value
			}
			update("PriceChange1", normalized.PriceChange1)
			update("PriceChange5", normalized.PriceChange5)
			update("PriceChange10", normalized.PriceChange10)
			update("RSI", normalized.RSI)
			update("EMASpread", normalized.EMASpread)
			update("MACDHistogram", normalized.MACDHistogram)
			update("ATR", normalized.ATR)
			update("BollingerPosition", normalized.BollingerPosition)
			update("VolumeRatio", normalized.VolumeRatio)
			ml.trainedWeights["bias"] += learningRate * error
		}

		// L2 regularization term
		reg := 0.0
		for name, w := range ml.trainedWeights {
			if name == "bias" {
				continue
			}
			if !math.IsNaN(w) && !math.IsInf(w, 0) {
				reg += w * w
				// weight decay (skip if would produce NaN)
				decay := learningRate * ml.l2Lambda * w
				if !math.IsNaN(decay) && !math.IsInf(decay, 0) {
					ml.trainedWeights[name] -= decay
				}
			}
		}

		if len(trainData) > 0 {
			ml.performance.TrainingLoss = totalError/float64(len(trainData)) + ml.l2Lambda*reg
		}

		// Early stopping if loss is small enough
		if ml.performance.TrainingLoss < 0.0001 {
			break
		}
	}

	return nil
}

// predictValue predicts expected return using trained weights
func (ml *MLModel) predictValue(features TradingFeatures) float64 {
	prediction := ml.trainedWeights["bias"]
	prediction += ml.trainedWeights["PriceChange1"] * features.PriceChange1
	prediction += ml.trainedWeights["PriceChange5"] * features.PriceChange5
	prediction += ml.trainedWeights["PriceChange10"] * features.PriceChange10
	prediction += ml.trainedWeights["RSI"] * features.RSI
	prediction += ml.trainedWeights["EMASpread"] * features.EMASpread
	prediction += ml.trainedWeights["MACDHistogram"] * features.MACDHistogram
	prediction += ml.trainedWeights["ATR"] * features.ATR
	prediction += ml.trainedWeights["BollingerPosition"] * features.BollingerPosition
	prediction += ml.trainedWeights["VolumeRatio"] * features.VolumeRatio

	return prediction
}

// evaluateModel evaluates model performance on validation data
func (ml *MLModel) evaluateModel(validationData []TradingFeatures) {
	if len(validationData) == 0 {
		return
	}

	correct := 0
	totalError := 0.0
	truePositives := 0
	falsePositives := 0
	falseNegatives := 0

	for _, data := range validationData {
		ml.sanitizeFeatures(&data)
		normalized := data
		ml.NormalizeFeatures(&normalized)
		prediction := ml.predictValue(normalized)
		error := data.FutureReturn - prediction
		totalError += error * error

		// Classification accuracy
		predictedDirection := 0
		if prediction > 0.001 {
			predictedDirection = 1
		} else if prediction < -0.001 {
			predictedDirection = -1
		}

		actualDirection := 0
		if data.FutureReturn > 0.001 {
			actualDirection = 1
		} else if data.FutureReturn < -0.001 {
			actualDirection = -1
		}

		if predictedDirection == actualDirection {
			correct++
		}

		// Precision/Recall for buy signals
		if predictedDirection == 1 {
			if actualDirection == 1 {
				truePositives++
			} else {
				falsePositives++
			}
		} else if actualDirection == 1 {
			falseNegatives++
		}
	}

	ml.performance.ValidationLoss = totalError / float64(len(validationData))
	ml.performance.Accuracy = float64(correct) / float64(len(validationData))

	if truePositives+falsePositives > 0 {
		ml.performance.Precision = float64(truePositives) / float64(truePositives+falsePositives)
	}
	if truePositives+falseNegatives > 0 {
		ml.performance.Recall = float64(truePositives) / float64(truePositives+falseNegatives)
	}
	if ml.performance.Precision+ml.performance.Recall > 0 {
		ml.performance.F1Score = 2 * (ml.performance.Precision * ml.performance.Recall) /
			(ml.performance.Precision + ml.performance.Recall)
	}
}

// Predict generates a trading prediction based on current features
func (ml *MLModel) Predict(features TradingFeatures) *PredictionResult {
	if len(ml.trainedWeights) == 0 {
		return &PredictionResult{
			Direction:  "HOLD",
			Confidence: 0.0,
		}
	}

	// Normalize features
	normalizedFeatures := features
	ml.NormalizeFeatures(&normalizedFeatures)

	// Make prediction
	expectedReturn := ml.predictValue(normalizedFeatures)

	// Determine direction and confidence
	direction := "HOLD"
	confidence := 0.0

	threshold := 0.001 // 0.1% threshold

	if expectedReturn > threshold {
		direction = "BUY"
		confidence = math.Min(math.Abs(expectedReturn)*100, 1.0) // Scale confidence
	} else if expectedReturn < -threshold {
		direction = "SELL"
		confidence = math.Min(math.Abs(expectedReturn)*100, 1.0)
	}

	return &PredictionResult{
		Direction:      direction,
		Confidence:     confidence,
		ExpectedReturn: expectedReturn,
		Features:       features,
	}
}

// ShouldRetrain determines if the model should be retrained
func (ml *MLModel) ShouldRetrain() bool {
	if len(ml.trainedWeights) == 0 {
		return len(ml.features) >= ml.config.MinTrainingPeriod
	}

	// Retrain periodically
	timeSinceTraining := time.Since(ml.lastTraining)
	retrainingInterval := time.Duration(ml.config.RetrainingPeriod) * time.Minute

	return timeSinceTraining > retrainingInterval && len(ml.features) >= ml.config.MinTrainingPeriod
}

// GetModelInfo returns information about the trained model
func (ml *MLModel) GetModelInfo() map[string]interface{} {
	info := make(map[string]interface{})

	info["training_count"] = ml.trainingCount
	info["data_points"] = len(ml.features)
	info["last_training"] = ml.lastTraining.Format("2006-01-02 15:04:05")
	info["performance"] = ml.performance
	info["feature_weights"] = ml.trainedWeights

	return info
}

// PrintModelPerformance prints detailed model performance metrics
func (ml *MLModel) PrintModelPerformance() {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("              ML MODEL PERFORMANCE")
	fmt.Println(strings.Repeat("=", 60))

	fmt.Printf("🤖 Model Status:\n")
	fmt.Printf("   Training Count:       %d\n", ml.trainingCount)
	fmt.Printf("   Data Points:          %d\n", len(ml.features))
	fmt.Printf("   Last Training:        %s\n", ml.lastTraining.Format("2006-01-02 15:04:05"))

	fmt.Printf("\n📊 Performance Metrics:\n")
	fmt.Printf("   Accuracy:             %.2f%%\n", ml.performance.Accuracy*100)
	fmt.Printf("   Precision:            %.3f\n", ml.performance.Precision)
	fmt.Printf("   Recall:               %.3f\n", ml.performance.Recall)
	fmt.Printf("   F1-Score:             %.3f\n", ml.performance.F1Score)
	fmt.Printf("   Training Loss:        %.6f\n", ml.performance.TrainingLoss)
	fmt.Printf("   Validation Loss:      %.6f\n", ml.performance.ValidationLoss)

	fmt.Printf("\n🎯 Feature Weights:\n")

	// Sort weights by absolute value
	type WeightPair struct {
		Name   string
		Weight float64
	}

	var weights []WeightPair
	for name, weight := range ml.trainedWeights {
		weights = append(weights, WeightPair{name, weight})
	}

	sort.Slice(weights, func(i, j int) bool {
		return math.Abs(weights[i].Weight) > math.Abs(weights[j].Weight)
	})

	for _, w := range weights {
		fmt.Printf("   %-18s: %+.4f\n", w.Name, w.Weight)
	}

	fmt.Println(strings.Repeat("=", 60))
}
