package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// RunBacktestCLI runs the backtesting CLI
func RunBacktestCLI() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Check for help flag first
	for _, arg := range os.Args {
		if arg == "-help" || arg == "--help" {
			printBacktestHelp()
			return
		}
	}

	// Parse simple flags from command line
	symbol := "BTCUSDT"
	initialBalance := 10000.0
	fee := 0.001
	interval := "15m"
	dataLimit := 500
	// analysis mode toggle (classic vs ML)
	useML := false

	// Simple argument parsing
	args := os.Args[1:] // Skip program name
	for i, arg := range args {
		if arg == "-backtest" {
			continue
		}
		if arg == "-useml" {
			useML = true
			continue
		}
		if strings.HasPrefix(arg, "-useml=") {
			v := strings.TrimPrefix(arg, "-useml=")
			v = strings.ToLower(strings.TrimSpace(v))
			useML = (v == "true" || v == "1" || v == "yes")
			continue
		}
		if strings.HasPrefix(arg, "-symbol=") {
			symbol = strings.TrimPrefix(arg, "-symbol=")
		} else if strings.HasPrefix(arg, "-balance=") {
			if val, err := strconv.ParseFloat(strings.TrimPrefix(arg, "-balance="), 64); err == nil {
				initialBalance = val
			}
		} else if strings.HasPrefix(arg, "-fee=") {
			if val, err := strconv.ParseFloat(strings.TrimPrefix(arg, "-fee="), 64); err == nil {
				fee = val
			}
		} else if strings.HasPrefix(arg, "-interval=") {
			interval = strings.TrimPrefix(arg, "-interval=")
		} else if strings.HasPrefix(arg, "-limit=") {
			if val, err := strconv.Atoi(strings.TrimPrefix(arg, "-limit=")); err == nil {
				dataLimit = val
			}
		} else if arg == "-symbol" && i+1 < len(args) {
			symbol = args[i+1]
		} else if arg == "-balance" && i+1 < len(args) {
			if val, err := strconv.ParseFloat(args[i+1], 64); err == nil {
				initialBalance = val
			}
		} else if arg == "-fee" && i+1 < len(args) {
			if val, err := strconv.ParseFloat(args[i+1], 64); err == nil {
				fee = val
			}
		} else if arg == "-interval" && i+1 < len(args) {
			interval = args[i+1]
		} else if arg == "-limit" && i+1 < len(args) {
			if val, err := strconv.Atoi(args[i+1]); err == nil {
				dataLimit = val
			}
		} else if arg == "-useml" && i+1 < len(args) {
			v := strings.ToLower(strings.TrimSpace(args[i+1]))
			useML = (v == "true" || v == "1" || v == "yes")
		}
	}

	// Env override for analyze mode
	if v := strings.ToLower(os.Getenv("USE_ML_ANALYZE")); v == "true" || v == "1" || v == "yes" {
		useML = true
	}
	// set global toggle
	if useML {
		UseMLAnalyze = true
		log.Printf("Backtest analyze(): ML mode enabled")
	}

	// Initialize Binance client
	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_SECRET_KEY")
	if apiKey == "" || secretKey == "" {
		log.Fatal("BINANCE_API_KEY and BINANCE_SECRET_KEY must be set in .env file")
	}
	binanceClient = NewBinanceClient(apiKey, secretKey)

	fmt.Printf("🚀 Starting backtest for %s\n", symbol)
	fmt.Printf("💰 Initial Balance: $%.2f\n", initialBalance)
	fmt.Printf("💸 Transaction Fee: %.3f%%\n", fee*100)
	fmt.Printf("⏱️  Interval: %s\n", interval)
	fmt.Printf("📊 Data Points: %d candles\n", dataLimit)
	fmt.Println(strings.Repeat("-", 50))

	// Create backtest configuration
	config := BacktestConfig{
		Symbol:         symbol,
		InitialBalance: initialBalance,
		TransactionFee: fee,
		Interval:       interval,
		DataLimit:      dataLimit,
	}

	// Create and run backtest engine
	engine := NewBacktestEngine(config)
	result, err := engine.RunBacktest()
	if err != nil {
		log.Fatalf("Backtest failed: %v", err)
	}

	// Print results
	PrintBacktestResults(result)

	// Optionally save results to file
	if shouldSaveResults() {
		saveBacktestResults(result)
	}
}

func printBacktestHelp() {
	fmt.Println(`
🔍 GoTrading Backtest CLI

USAGE:
  go run . -backtest [OPTIONS]

 OPTIONS:
  -symbol      Trading pair to test (default: BTCUSDT)
  -balance     Initial balance in USD (default: 10000)
  -fee         Transaction fee percentage (default: 0.001)
  -interval    Candle interval: 1m, 5m, 15m, 1h, 4h, 1d (default: 15m)
  -limit       Number of historical candles (default: 500)
  -useml       Use ML-based analyze() instead of classic rules (also via USE_ML_ANALYZE env)
  -help        Show this help message

EXAMPLES:
  # Basic backtest with BTC
  go run . -backtest -symbol=BTCUSDT

  # Test ETH with custom balance and fee
  go run . -backtest -symbol=ETHUSDT -balance=5000 -fee=0.0015

  # Test with hourly candles
  go run . -backtest -symbol=ADAUSDT -interval=1h -limit=1000

REQUIREMENTS:
  - Set BINANCE_API_KEY and BINANCE_SECRET_KEY in .env file
  - Ensure you have an active internet connection

The backtest will analyze your trading strategy against historical data
and provide detailed performance metrics including:
- Total return vs Buy & Hold
- Win rate and trade statistics
- Maximum drawdown
- Sharpe ratio
- Detailed trade history
`)
}

func shouldSaveResults() bool {
	fmt.Print("\n💾 Save results to file? (y/N): ")
	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func saveBacktestResults(result *BacktestResult) {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("backtest_%s_%s.txt", result.Symbol, timestamp)

	file, err := os.Create(filename)
	if err != nil {
		log.Printf("Error creating results file: %v", err)
		return
	}
	defer file.Close()

	// Redirect stdout to file temporarily
	oldStdout := os.Stdout
	os.Stdout = file

	PrintBacktestResults(result)

	// Write additional details to file
	fmt.Fprintf(file, "\n\n📋 DETAILED TRADE LOG:\n")
	fmt.Fprintf(file, "%s\n", strings.Repeat("-", 80))
	for i, trade := range result.Trades {
		fmt.Fprintf(file, "%d. %s %.6f %s at $%.2f on %s (Fee: $%.2f)\n",
			i+1, trade.Type, trade.Quantity, trade.Symbol, trade.Price,
			trade.Timestamp.Format("2006-01-02 15:04:05"), trade.Fee)
	}

	// Restore stdout
	os.Stdout = oldStdout

	fmt.Printf("✅ Results saved to: %s\n", filename)
}

// parseInterval converts interval string to minutes for internal use
func parseInterval(interval string) (int, error) {
	switch strings.ToLower(interval) {
	case "1m":
		return 1, nil
	case "5m":
		return 5, nil
	case "15m":
		return 15, nil
	case "30m":
		return 30, nil
	case "1h":
		return 60, nil
	case "4h":
		return 240, nil
	case "1d":
		return 1440, nil
	default:
		return 0, fmt.Errorf("unsupported interval: %s", interval)
	}
}

// runBatchBacktest runs backtests for multiple symbols
func runBatchBacktest(symbols []string, config BacktestConfig) {
	fmt.Println("🔄 Running batch backtest...")

	results := make(map[string]*BacktestResult)

	for _, symbol := range symbols {
		symbol = strings.TrimSpace(strings.ToUpper(symbol))
		fmt.Printf("\n📊 Testing %s...\n", symbol)

		config.Symbol = symbol
		engine := NewBacktestEngine(config)

		result, err := engine.RunBacktest()
		if err != nil {
			log.Printf("❌ Backtest failed for %s: %v", symbol, err)
			continue
		}

		results[symbol] = result
		fmt.Printf("✅ %s completed: %.2f%% return\n", symbol, result.TotalReturnPct)
	}

	// Print comparison summary
	printBatchSummary(results)
}

func printBatchSummary(results map[string]*BacktestResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("                         BATCH BACKTEST SUMMARY")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf("%-10s %-12s %-12s %-12s %-10s %-8s\n",
		"Symbol", "Return %", "Buy&Hold %", "Alpha %", "Trades", "Win Rate")
	fmt.Println(strings.Repeat("-", 80))

	bestPerformer := ""
	bestReturn := -999.0

	for symbol, result := range results {
		alpha := result.TotalReturnPct - result.BuyAndHoldReturnPct
		fmt.Printf("%-10s %11.2f%% %11.2f%% %11.2f%% %9d %7.1f%%\n",
			symbol, result.TotalReturnPct, result.BuyAndHoldReturnPct,
			alpha, result.TotalTrades, result.WinRate)

		if result.TotalReturnPct > bestReturn {
			bestReturn = result.TotalReturnPct
			bestPerformer = symbol
		}
	}

	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("🏆 Best Performer: %s (%.2f%% return)\n", bestPerformer, bestReturn)
	fmt.Println(strings.Repeat("=", 80))
}
