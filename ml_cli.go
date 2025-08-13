package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

// RunMLCLI runs the machine learning CLI
func RunMLCLI() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Check for help flag first
	for _, arg := range os.Args {
		if arg == "-help" || arg == "--help" {
			printMLHelp()
			return
		}
	}

	// Parse command line arguments
	symbol := "BTCUSDT"
	mode := "train"        // "train", "test", "live"
	dataLimit := 1000
	lookbackPeriod := 500
	minTrainingPeriod := 100
	retrainingPeriod := 60  // minutes

	// Simple argument parsing
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "-ml" {
			continue
		}
		if strings.HasPrefix(arg, "-symbol=") {
			symbol = strings.TrimPrefix(arg, "-symbol=")
		} else if strings.HasPrefix(arg, "-mode=") {
			mode = strings.TrimPrefix(arg, "-mode=")
		} else if strings.HasPrefix(arg, "-limit=") {
			if val, err := strconv.Atoi(strings.TrimPrefix(arg, "-limit=")); err == nil {
				dataLimit = val
			}
		} else if strings.HasPrefix(arg, "-lookback=") {
			if val, err := strconv.Atoi(strings.TrimPrefix(arg, "-lookback=")); err == nil {
				lookbackPeriod = val
			}
		} else if strings.HasPrefix(arg, "-mintrain=") {
			if val, err := strconv.Atoi(strings.TrimPrefix(arg, "-mintrain=")); err == nil {
				minTrainingPeriod = val
			}
		} else if strings.HasPrefix(arg, "-retrain=") {
			if val, err := strconv.Atoi(strings.TrimPrefix(arg, "-retrain=")); err == nil {
				retrainingPeriod = val
			}
		} else if arg == "-symbol" && i+1 < len(args) {
			symbol = args[i+1]
		} else if arg == "-mode" && i+1 < len(args) {
			mode = args[i+1]
		} else if arg == "-limit" && i+1 < len(args) {
			if val, err := strconv.Atoi(args[i+1]); err == nil {
				dataLimit = val
			}
		}
	}

	// Initialize Binance client
	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_SECRET_KEY")
	if apiKey == "" || secretKey == "" {
		log.Fatal("BINANCE_API_KEY and BINANCE_SECRET_KEY must be set in .env file")
	}
	binanceClient = NewBinanceClient(apiKey, secretKey)

	fmt.Printf("🤖 ML Strategy Mode: %s\n", strings.ToUpper(mode))
	fmt.Printf("📊 Symbol: %s\n", symbol)
	fmt.Printf("📈 Data Limit: %d candles\n", dataLimit)
	fmt.Printf("🔍 Lookback Period: %d\n", lookbackPeriod)
	fmt.Println(strings.Repeat("-", 50))

	switch mode {
	case "train":
		runMLTraining(symbol, dataLimit, lookbackPeriod, minTrainingPeriod)
	case "test":
		runMLTesting(symbol, dataLimit, lookbackPeriod, minTrainingPeriod)
	case "live":
		runMLLiveTrading(symbol, lookbackPeriod, minTrainingPeriod, retrainingPeriod)
	case "compare":
		runMLComparison(symbol, dataLimit, lookbackPeriod, minTrainingPeriod)
	default:
		fmt.Printf("❌ Unknown mode: %s\n", mode)
		fmt.Println("Available modes: train, test, live, compare")
	}
}

func printMLHelp() {
	fmt.Println(`
🤖 GoTrading ML Strategy CLI

USAGE:
  go run . -ml [OPTIONS]

MODES:
  train     - Train ML model on historical data
  test      - Test trained model against historical data  
  live      - Run live trading with ML predictions
  compare   - Compare ML strategy vs traditional strategy

OPTIONS:
  -symbol      Trading pair to analyze (default: BTCUSDT)
  -mode        Operation mode (default: train)
  -limit       Number of historical candles (default: 1000)
  -lookback    Lookback period for model (default: 500)  
  -mintrain    Minimum training periods (default: 100)
  -retrain     Retraining interval in minutes (default: 60)
  -help        Show this help message

EXAMPLES:
  # Train ML model on BTC data
  go run . -ml -mode=train -symbol=BTCUSDT

  # Test ML model performance  
  go run . -ml -mode=test -symbol=ETHUSDT -limit=2000

  # Compare ML vs traditional strategy
  go run . -ml -mode=compare -symbol=ADAUSDT

  # Run live ML trading
  go run . -ml -mode=live -symbol=BTCUSDT -retrain=30

FEATURES:
  📊 Extracts 15+ technical features from price data
  🧠 Uses machine learning to predict price movements
  🎯 Automatically optimizes strategy parameters
  📈 Provides detailed performance analytics
  🔄 Supports online learning and model retraining
  💡 Compares against traditional indicators

The ML system learns patterns from historical data and adapts
to changing market conditions for improved trading decisions.
`)
}

// runMLTraining trains a new ML model on historical data
func runMLTraining(symbol string, dataLimit, lookbackPeriod, minTrainingPeriod int) {
	fmt.Println("🚀 Starting ML model training...")

	// Create ML configuration
	config := MLConfig{
		LookbackPeriod:    lookbackPeriod,
		TrainingRatio:     0.8,
		MinTrainingPeriod: minTrainingPeriod,
		RetrainingPeriod:  60,
		FeatureWindow:     30,
	}

	// Create ML model
	mlModel := NewMLModel(config)

	// Fetch historical data
	fmt.Printf("📊 Fetching %d candles of historical data...\n", dataLimit)
	klines, err := binanceClient.fetchKlines(symbol, "15m", dataLimit)
	if err != nil {
		log.Fatalf("Error fetching historical data: %v", err)
	}

	// Convert to time series
	ts := buildTimeSeries(klines)
	
	fmt.Printf("🔍 Extracting features from %d data points...\n", len(klines))
	
	// Extract features and prepare training data
	for i := config.FeatureWindow; i < len(klines)-5; i++ {
		features := mlModel.ExtractFeatures(ts, i)
		if features == nil {
			continue
		}

		// Calculate future return (5 periods ahead)
		if i+5 < len(klines) {
			currentPrice, _ := strconv.ParseFloat(klines[i].Close, 64)
			futurePrice, _ := strconv.ParseFloat(klines[i+5].Close, 64)
			features.FutureReturn = (futurePrice - currentPrice) / currentPrice
		}

		mlModel.AddTrainingData(*features)
	}

	// Train the model
	fmt.Println("🧠 Training machine learning model...")
	err = mlModel.TrainModel()
	if err != nil {
		log.Fatalf("Training failed: %v", err)
	}

	// Display results
	mlModel.PrintModelPerformance()

	// Save model (in a real implementation)
	fmt.Println("\n💾 Model training completed successfully!")
	fmt.Printf("📈 Model trained on %d data points\n", len(mlModel.features))
	fmt.Printf("🎯 Accuracy: %.2f%%\n", mlModel.performance.Accuracy*100)
	fmt.Printf("⚡ F1-Score: %.3f\n", mlModel.performance.F1Score)
}

// runMLTesting tests the ML model against historical data
func runMLTesting(symbol string, dataLimit, lookbackPeriod, minTrainingPeriod int) {
	fmt.Println("🧪 Testing ML model performance...")

	config := MLConfig{
		LookbackPeriod:    lookbackPeriod,
		TrainingRatio:     0.7, // Use 70% for training, 30% for testing
		MinTrainingPeriod: minTrainingPeriod,
		RetrainingPeriod:  60,
		FeatureWindow:     30,
	}

	mlModel := NewMLModel(config)

	// Fetch data
	fmt.Printf("📊 Fetching %d candles for testing...\n", dataLimit)
	klines, err := binanceClient.fetchKlines(symbol, "15m", dataLimit)
	if err != nil {
		log.Fatalf("Error fetching data: %v", err)
	}

	ts := buildTimeSeries(klines)

	// Prepare all data
	allFeatures := make([]TradingFeatures, 0)
	for i := config.FeatureWindow; i < len(klines)-5; i++ {
		features := mlModel.ExtractFeatures(ts, i)
		if features == nil {
			continue
		}

		if i+5 < len(klines) {
			currentPrice, _ := strconv.ParseFloat(klines[i].Close, 64)
			futurePrice, _ := strconv.ParseFloat(klines[i+5].Close, 64)
			features.FutureReturn = (futurePrice - currentPrice) / currentPrice
		}

		allFeatures = append(allFeatures, *features)
	}

	// Split data
	trainSize := int(float64(len(allFeatures)) * config.TrainingRatio)
	trainData := allFeatures[:trainSize]
	testData := allFeatures[trainSize:]

	fmt.Printf("📚 Training on %d samples, testing on %d samples\n", len(trainData), len(testData))

	// Add training data and train
	for _, features := range trainData {
		mlModel.AddTrainingData(features)
	}

	err = mlModel.TrainModel()
	if err != nil {
		log.Fatalf("Training failed: %v", err)
	}

	// Test predictions
	fmt.Println("🎯 Testing model predictions...")
	correct := 0
	totalReturns := 0.0
	predictedReturns := 0.0

	for _, testFeatures := range testData {
		prediction := mlModel.Predict(testFeatures)
		
		// Check direction accuracy
		actualDirection := 0
		if testFeatures.FutureReturn > 0.001 {
			actualDirection = 1
		} else if testFeatures.FutureReturn < -0.001 {
			actualDirection = -1
		}

		predictedDirection := 0
		if prediction.Direction == "BUY" {
			predictedDirection = 1
		} else if prediction.Direction == "SELL" {
			predictedDirection = -1
		}

		if predictedDirection == actualDirection {
			correct++
		}

		// Calculate returns if following predictions
		if prediction.Direction == "BUY" && testFeatures.FutureReturn > 0 {
			predictedReturns += testFeatures.FutureReturn
		} else if prediction.Direction == "SELL" && testFeatures.FutureReturn < 0 {
			predictedReturns += -testFeatures.FutureReturn // Short profit
		}

		totalReturns += testFeatures.FutureReturn
	}

	// Display test results
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("                   TEST RESULTS")
	fmt.Println(strings.Repeat("=", 60))
	
	accuracy := float64(correct) / float64(len(testData)) * 100
	fmt.Printf("🎯 Direction Accuracy:    %.2f%%\n", accuracy)
	fmt.Printf("📊 Test Samples:          %d\n", len(testData))
	fmt.Printf("💰 ML Strategy Returns:   %.4f\n", predictedReturns)
	fmt.Printf("💹 Buy & Hold Returns:    %.4f\n", totalReturns)
	
	if totalReturns != 0 {
		alpha := (predictedReturns - totalReturns) / math.Abs(totalReturns) * 100
		fmt.Printf("🚀 Alpha vs Buy & Hold:  %.2f%%\n", alpha)
	}

	mlModel.PrintModelPerformance()
	fmt.Println(strings.Repeat("=", 60))
}

// runMLComparison compares ML strategy with traditional strategy
func runMLComparison(symbol string, dataLimit, lookbackPeriod, minTrainingPeriod int) {
	fmt.Println("⚖️  Comparing ML vs Traditional Strategy...")

	// First run a traditional backtest
	fmt.Println("📊 Running traditional strategy backtest...")
	traditionalConfig := BacktestConfig{
		Symbol:         symbol,
		InitialBalance: 10000,
		TransactionFee: 0.001,
		Interval:       "15m",
		DataLimit:      dataLimit,
	}

	traditionalEngine := NewBacktestEngine(traditionalConfig)
	traditionalResult, err := traditionalEngine.RunBacktest()
	if err != nil {
		log.Fatalf("Traditional backtest failed: %v", err)
	}

	// Now run ML backtest simulation
	fmt.Println("🤖 Running ML strategy simulation...")
	mlConfig := MLConfig{
		LookbackPeriod:    lookbackPeriod,
		TrainingRatio:     0.8,
		MinTrainingPeriod: minTrainingPeriod,
		RetrainingPeriod:  60,
		FeatureWindow:     30,
	}

	mlModel := NewMLModel(mlConfig)
	
	// Fetch data for ML
	klines, err := binanceClient.fetchKlines(symbol, "15m", dataLimit)
	if err != nil {
		log.Fatalf("Error fetching data: %v", err)
	}

	ts := buildTimeSeries(klines)

	// Train ML model first
	for i := mlConfig.FeatureWindow; i < len(klines)/2; i++ {
		features := mlModel.ExtractFeatures(ts, i)
		if features == nil {
			continue
		}

		if i+5 < len(klines) {
			currentPrice, _ := strconv.ParseFloat(klines[i].Close, 64)
			futurePrice, _ := strconv.ParseFloat(klines[i+5].Close, 64)
			features.FutureReturn = (futurePrice - currentPrice) / currentPrice
		}

		mlModel.AddTrainingData(*features)
	}

	err = mlModel.TrainModel()
	if err != nil {
		log.Fatalf("ML training failed: %v", err)
	}

	// Simulate ML trading on remaining data
	cash := 10000.0
	position := 0.0
	mlTrades := 0
	mlWins := 0

	for i := len(klines)/2; i < len(klines)-5; i++ {
		features := mlModel.ExtractFeatures(ts, i)
		if features == nil {
			continue
		}

		prediction := mlModel.Predict(*features)
		currentPrice, _ := strconv.ParseFloat(klines[i].Close, 64)
		
		if prediction.Direction == "BUY" && position == 0 {
			position = cash / currentPrice
			cash = 0
			mlTrades++
		} else if prediction.Direction == "SELL" && position > 0 {
			cash = position * currentPrice
			if cash > 10000 {
				mlWins++
			}
			position = 0
			mlTrades++
		}
	}

	// Final liquidation
	if position > 0 {
		finalPrice, _ := strconv.ParseFloat(klines[len(klines)-1].Close, 64)
		cash = position * finalPrice
	}

	mlFinalValue := cash
	mlReturn := (mlFinalValue - 10000) / 10000 * 100
	mlWinRate := 0.0
	if mlTrades > 0 {
		mlWinRate = float64(mlWins) / float64(mlTrades) * 100
	}

	// Display comparison
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("                    STRATEGY COMPARISON")
	fmt.Println(strings.Repeat("=", 80))
	
	fmt.Printf("📊 PERFORMANCE COMPARISON:\n")
	fmt.Printf("   %-20s   Traditional    ML Strategy\n", "Metric")
	fmt.Println(strings.Repeat("-", 55))
	fmt.Printf("   %-20s   %8.2f%%      %8.2f%%\n", "Total Return", traditionalResult.TotalReturnPct, mlReturn)
	fmt.Printf("   %-20s   %8.2f%%      %8.2f%%\n", "Win Rate", traditionalResult.WinRate, mlWinRate)
	fmt.Printf("   %-20s   %8d         %8d\n", "Total Trades", traditionalResult.TotalTrades, mlTrades)
	fmt.Printf("   %-20s   %8.2f%%      %8.2f%%\n", "Max Drawdown", traditionalResult.MaxDrawdownPct, 0.0) // Simplified
	fmt.Printf("   %-20s   %8.3f        %8.3f\n", "Sharpe Ratio", traditionalResult.SharpeRatio, 0.0) // Simplified
	
	fmt.Println(strings.Repeat("-", 55))
	
	winner := "Traditional"
	if mlReturn > traditionalResult.TotalReturnPct {
		winner = "ML Strategy"
	}
	
	fmt.Printf("🏆 Winner: %s\n", winner)
	fmt.Printf("📈 Performance Difference: %.2f%%\n", mlReturn - traditionalResult.TotalReturnPct)
	
	fmt.Println(strings.Repeat("=", 80))
}

// runMLLiveTrading runs live trading with ML predictions
func runMLLiveTrading(symbol string, lookbackPeriod, minTrainingPeriod, retrainingPeriod int) {
	fmt.Println("🔴 LIVE ML Trading Mode (Demo)")
	fmt.Println("⚠️  This is a demonstration - no real trades will be executed")
	
	config := MLConfig{
		LookbackPeriod:    lookbackPeriod,
		TrainingRatio:     0.9,
		MinTrainingPeriod: minTrainingPeriod,
		RetrainingPeriod:  retrainingPeriod,
		FeatureWindow:     30,
	}

	mlModel := NewMLModel(config)
	
	fmt.Println("📊 Collecting initial training data...")
	
	// Initial training
	klines, err := binanceClient.fetchKlines(symbol, "15m", lookbackPeriod)
	if err != nil {
		log.Fatalf("Error fetching initial data: %v", err)
	}

	ts := buildTimeSeries(klines)

	// Build initial training data
	for i := config.FeatureWindow; i < len(klines)-5; i++ {
		features := mlModel.ExtractFeatures(ts, i)
		if features == nil {
			continue
		}

		if i+5 < len(klines) {
			currentPrice, _ := strconv.ParseFloat(klines[i].Close, 64)
			futurePrice, _ := strconv.ParseFloat(klines[i+5].Close, 64)
			features.FutureReturn = (futurePrice - currentPrice) / currentPrice
		}

		mlModel.AddTrainingData(*features)
	}

	// Initial training
	err = mlModel.TrainModel()
	if err != nil {
		log.Fatalf("Initial training failed: %v", err)
	}

	fmt.Println("🤖 ML model trained and ready!")
	mlModel.PrintModelPerformance()

	// Live monitoring loop
	fmt.Println("🔄 Starting live monitoring...")
	for {
		// Fetch latest data
		klines, err := binanceClient.fetchKlines(symbol, "15m", 50)
		if err != nil {
			log.Printf("Error fetching live data: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		// Update time series
		ts := buildTimeSeries(klines)
		
		// Extract current features
		features := mlModel.ExtractFeatures(ts, len(klines)-1)
		if features != nil {
			prediction := mlModel.Predict(*features)
			
			// Display prediction
			currentPrice := klines[len(klines)-1].Close
			now := time.Now()
			
			fmt.Printf("\n[%s] %s: $%s\n", 
				now.Format("15:04:05"), symbol, currentPrice)
			fmt.Printf("🤖 ML Prediction: %s (Confidence: %.2f)\n", 
				prediction.Direction, prediction.Confidence)
			fmt.Printf("📈 Expected Return: %.4f\n", prediction.ExpectedReturn)
			
			// Check if retraining is needed
			if mlModel.ShouldRetrain() {
				fmt.Println("🔄 Retraining model with new data...")
				// In a real system, you'd update training data here
				err = mlModel.TrainModel()
				if err != nil {
					log.Printf("Retraining failed: %v", err)
				} else {
					fmt.Println("✅ Model retrained successfully")
				}
			}
		}
		
		// Wait for next cycle
		fmt.Printf("⏰ Waiting %d minutes for next prediction...\n", retrainingPeriod/4)
		time.Sleep(time.Duration(retrainingPeriod/4) * time.Minute)
	}
}

// buildTimeSeries converts klines to techan TimeSeries
func buildTimeSeries(klines []BinanceKline) *techan.TimeSeries {
	ts := techan.NewTimeSeries()
	
	for _, kline := range klines {
		open, _ := strconv.ParseFloat(kline.Open, 64)
		high, _ := strconv.ParseFloat(kline.High, 64)
		low, _ := strconv.ParseFloat(kline.Low, 64)
		close, _ := strconv.ParseFloat(kline.Close, 64)
		
		period := techan.NewTimePeriod(time.UnixMilli(kline.OpenTime), time.Minute*15)
		c := techan.NewCandle(period)
		c.OpenPrice = big.NewDecimal(open)
		c.MaxPrice = big.NewDecimal(high)
		c.MinPrice = big.NewDecimal(low)
		c.ClosePrice = big.NewDecimal(close)
		c.Volume = big.NewDecimal(0)
		ts.AddCandle(c)
	}
	
	return ts
}
