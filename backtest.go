package main

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

// BacktestConfig holds configuration for backtesting
type BacktestConfig struct {
	Symbol           string
	InitialBalance   float64
	TransactionFee   float64 // Fee percentage (e.g., 0.001 for 0.1%)
	StartDate        time.Time
	EndDate          time.Time
	Interval         string
	DataLimit        int // Number of candles to fetch
}

// Trade represents a single trade execution
type Trade struct {
	Symbol      string
	Type        string    // "BUY" or "SELL"
	Price       float64
	Quantity    float64
	Timestamp   time.Time
	Fee         float64
	Balance     float64
	TotalValue  float64
}

// BacktestResult holds the results of a backtest
type BacktestResult struct {
	Symbol            string
	InitialBalance    float64
	FinalBalance      float64
	FinalValue        float64
	TotalReturn       float64
	TotalReturnPct    float64
	MaxDrawdown       float64
	MaxDrawdownPct    float64
	WinRate           float64
	TotalTrades       int
	WinningTrades     int
	LosingTrades      int
	AverageWin        float64
	AverageLoss       float64
	SharpeRatio       float64
	Trades            []Trade
	DailyReturns      []float64
	EquityCurve       []float64
	Duration          time.Duration
	BuyAndHoldReturn  float64
	BuyAndHoldReturnPct float64
}

// Portfolio represents the current portfolio state
type Portfolio struct {
	Cash       float64
	Holdings   map[string]float64 // symbol -> quantity
	LastPrices map[string]float64 // symbol -> last price
}

// BacktestEngine performs backtesting operations
type BacktestEngine struct {
	config    BacktestConfig
	portfolio Portfolio
	trades    []Trade
	startTime time.Time
	endTime   time.Time
}

// NewBacktestEngine creates a new backtesting engine
func NewBacktestEngine(config BacktestConfig) *BacktestEngine {
	return &BacktestEngine{
		config: config,
		portfolio: Portfolio{
			Cash:       config.InitialBalance,
			Holdings:   make(map[string]float64),
			LastPrices: make(map[string]float64),
		},
		trades: make([]Trade, 0),
	}
}

// GetPortfolioValue calculates the total portfolio value
func (be *BacktestEngine) GetPortfolioValue() float64 {
	totalValue := be.portfolio.Cash
	
	for symbol, quantity := range be.portfolio.Holdings {
		if price, exists := be.portfolio.LastPrices[symbol]; exists {
			totalValue += quantity * price
		}
	}
	
	return totalValue
}

// ExecuteTrade executes a buy or sell trade
func (be *BacktestEngine) ExecuteTrade(symbol, tradeType string, price float64, timestamp time.Time) bool {
	fee := price * be.config.TransactionFee
	
	switch tradeType {
	case "BUY":
		// Calculate maximum quantity we can buy
		availableCash := be.portfolio.Cash
		costPerUnit := price + fee
		maxQuantity := availableCash / costPerUnit
		
		if maxQuantity <= 0 {
			log.Printf("Insufficient funds to buy %s at $%.2f", symbol, price)
			return false
		}
		
		totalCost := maxQuantity * price
		totalFee := maxQuantity * fee
		
		// Update portfolio
		be.portfolio.Cash -= (totalCost + totalFee)
		be.portfolio.Holdings[symbol] += maxQuantity
		be.portfolio.LastPrices[symbol] = price
		
		// Record trade
		trade := Trade{
			Symbol:     symbol,
			Type:       tradeType,
			Price:      price,
			Quantity:   maxQuantity,
			Timestamp:  timestamp,
			Fee:        totalFee,
			Balance:    be.portfolio.Cash,
			TotalValue: be.GetPortfolioValue(),
		}
		be.trades = append(be.trades, trade)
		
		log.Printf("BUY: %.6f %s at $%.2f (Fee: $%.2f, Cash: $%.2f)", 
			maxQuantity, symbol, price, totalFee, be.portfolio.Cash)
		return true
		
	case "SELL":
		// Check if we have holdings to sell
		quantity, exists := be.portfolio.Holdings[symbol]
		if !exists || quantity <= 0 {
			log.Printf("No holdings to sell for %s", symbol)
			return false
		}
		
		totalRevenue := quantity * price
		totalFee := quantity * fee
		netRevenue := totalRevenue - totalFee
		
		// Update portfolio
		be.portfolio.Cash += netRevenue
		delete(be.portfolio.Holdings, symbol)
		be.portfolio.LastPrices[symbol] = price
		
		// Record trade
		trade := Trade{
			Symbol:     symbol,
			Type:       tradeType,
			Price:      price,
			Quantity:   quantity,
			Timestamp:  timestamp,
			Fee:        totalFee,
			Balance:    be.portfolio.Cash,
			TotalValue: be.GetPortfolioValue(),
		}
		be.trades = append(be.trades, trade)
		
		log.Printf("SELL: %.6f %s at $%.2f (Fee: $%.2f, Cash: $%.2f)", 
			quantity, symbol, price, totalFee, be.portfolio.Cash)
		return true
	}
	
	return false
}

// RunBacktest executes the backtest for a given symbol
func (be *BacktestEngine) RunBacktest() (*BacktestResult, error) {
	log.Printf("Starting backtest for %s...", be.config.Symbol)
	
	// Fetch historical data
	klines, err := binanceClient.fetchKlines(be.config.Symbol, be.config.Interval, be.config.DataLimit)
	if err != nil {
		return nil, fmt.Errorf("error fetching historical data: %v", err)
	}
	
	if len(klines) == 0 {
		return nil, fmt.Errorf("no historical data available for %s", be.config.Symbol)
	}
	
	log.Printf("Loaded %d candles for backtesting", len(klines))
	
	// Create time series
	ts := techan.NewTimeSeries()
	prices := make([]float64, 0, len(klines))
	
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
		
		prices = append(prices, close)
	}
	
	be.startTime = time.UnixMilli(klines[0].OpenTime)
	be.endTime = time.UnixMilli(klines[len(klines)-1].CloseTime)
	
	// Run strategy simulation
	equityCurve := make([]float64, 0, len(klines))
	dailyReturns := make([]float64, 0)
	maxValue := be.config.InitialBalance
	maxDrawdown := 0.0
	
	for i := 26; i < len(klines); i++ { // Start after enough data for indicators
		// Update current price
		currentPrice := prices[i]
		be.portfolio.LastPrices[be.config.Symbol] = currentPrice
		
		// Create a sub-series up to current point
		subSeries := techan.NewTimeSeries()
		for j := 0; j <= i; j++ {
			candle := ts.Candles[j]
			subSeries.AddCandle(candle)
		}
		
		// Get trading signal
		signal := analyze(be.config.Symbol, subSeries)
		timestamp := time.UnixMilli(klines[i].OpenTime)
		
		// Execute trade based on signal
		if signal == "BUY" {
			be.ExecuteTrade(be.config.Symbol, "BUY", currentPrice, timestamp)
		} else if signal == "SELL" {
			be.ExecuteTrade(be.config.Symbol, "SELL", currentPrice, timestamp)
		}
		
		// Track portfolio value
		currentValue := be.GetPortfolioValue()
		equityCurve = append(equityCurve, currentValue)
		
		// Calculate daily return
		if len(equityCurve) > 1 {
			prevValue := equityCurve[len(equityCurve)-2]
			if prevValue > 0 {
				dailyReturn := (currentValue - prevValue) / prevValue
				dailyReturns = append(dailyReturns, dailyReturn)
			}
		}
		
		// Track maximum drawdown
		if currentValue > maxValue {
			maxValue = currentValue
		}
		
		drawdown := maxValue - currentValue
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}
	
	// Calculate final results
	finalValue := be.GetPortfolioValue()
	totalReturn := finalValue - be.config.InitialBalance
	totalReturnPct := (totalReturn / be.config.InitialBalance) * 100
	maxDrawdownPct := (maxDrawdown / maxValue) * 100
	
	// Calculate buy and hold return
	firstPrice := prices[0]
	lastPrice := prices[len(prices)-1]
	buyAndHoldReturn := ((lastPrice - firstPrice) / firstPrice) * be.config.InitialBalance
	buyAndHoldReturnPct := ((lastPrice - firstPrice) / firstPrice) * 100
	
	// Calculate trade statistics
	winningTrades := 0
	losingTrades := 0
	totalWins := 0.0
	totalLosses := 0.0
	
	// Pair buy and sell trades to calculate P&L
	buyTrades := make([]Trade, 0)
	for _, trade := range be.trades {
		if trade.Type == "BUY" {
			buyTrades = append(buyTrades, trade)
		} else if trade.Type == "SELL" && len(buyTrades) > 0 {
			// Match with most recent buy
			buyTrade := buyTrades[len(buyTrades)-1]
			buyTrades = buyTrades[:len(buyTrades)-1]
			
			pnl := (trade.Price - buyTrade.Price) * trade.Quantity - trade.Fee - buyTrade.Fee
			if pnl > 0 {
				winningTrades++
				totalWins += pnl
			} else {
				losingTrades++
				totalLosses += math.Abs(pnl)
			}
		}
	}
	
	var winRate, avgWin, avgLoss float64
	totalCompletedTrades := winningTrades + losingTrades
	if totalCompletedTrades > 0 {
		winRate = (float64(winningTrades) / float64(totalCompletedTrades)) * 100
	}
	if winningTrades > 0 {
		avgWin = totalWins / float64(winningTrades)
	}
	if losingTrades > 0 {
		avgLoss = totalLosses / float64(losingTrades)
	}
	
	// Calculate Sharpe ratio
	var sharpeRatio float64
	if len(dailyReturns) > 1 {
		mean := calculateMean(dailyReturns)
		stdDev := calculateStdDev(dailyReturns, mean)
		if stdDev > 0 {
			sharpeRatio = (mean * math.Sqrt(252)) / (stdDev * math.Sqrt(252)) // Annualized
		}
	}
	
	result := &BacktestResult{
		Symbol:              be.config.Symbol,
		InitialBalance:      be.config.InitialBalance,
		FinalBalance:        be.portfolio.Cash,
		FinalValue:          finalValue,
		TotalReturn:         totalReturn,
		TotalReturnPct:      totalReturnPct,
		MaxDrawdown:         maxDrawdown,
		MaxDrawdownPct:      maxDrawdownPct,
		WinRate:             winRate,
		TotalTrades:         len(be.trades),
		WinningTrades:       winningTrades,
		LosingTrades:        losingTrades,
		AverageWin:          avgWin,
		AverageLoss:         avgLoss,
		SharpeRatio:         sharpeRatio,
		Trades:              be.trades,
		DailyReturns:        dailyReturns,
		EquityCurve:         equityCurve,
		Duration:            be.endTime.Sub(be.startTime),
		BuyAndHoldReturn:    buyAndHoldReturn,
		BuyAndHoldReturnPct: buyAndHoldReturnPct,
	}
	
	log.Printf("Backtest completed for %s", be.config.Symbol)
	return result, nil
}

// Helper functions for statistical calculations
func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	
	return math.Sqrt(sumSquares / float64(len(values)-1))
}

// PrintBacktestResults prints a detailed report of backtest results
func PrintBacktestResults(result *BacktestResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("                    BACKTEST RESULTS - %s\n", result.Symbol)
	fmt.Println(strings.Repeat("=", 80))
	
	fmt.Printf("📊 PERFORMANCE OVERVIEW\n")
	fmt.Printf("   Initial Balance:      $%.2f\n", result.InitialBalance)
	fmt.Printf("   Final Value:          $%.2f\n", result.FinalValue)
	fmt.Printf("   Total Return:         $%.2f (%.2f%%)\n", result.TotalReturn, result.TotalReturnPct)
	fmt.Printf("   Buy & Hold Return:    $%.2f (%.2f%%)\n", result.BuyAndHoldReturn, result.BuyAndHoldReturnPct)
	fmt.Printf("   Alpha vs Buy & Hold:  %.2f%%\n", result.TotalReturnPct - result.BuyAndHoldReturnPct)
	fmt.Printf("   Max Drawdown:         $%.2f (%.2f%%)\n", result.MaxDrawdown, result.MaxDrawdownPct)
	fmt.Printf("   Sharpe Ratio:         %.3f\n", result.SharpeRatio)
	fmt.Printf("   Duration:             %v\n", result.Duration.Round(24*time.Hour))
	
	fmt.Printf("\n📈 TRADE STATISTICS\n")
	fmt.Printf("   Total Trades:         %d\n", result.TotalTrades)
	fmt.Printf("   Winning Trades:       %d\n", result.WinningTrades)
	fmt.Printf("   Losing Trades:        %d\n", result.LosingTrades)
	fmt.Printf("   Win Rate:             %.1f%%\n", result.WinRate)
	fmt.Printf("   Average Win:          $%.2f\n", result.AverageWin)
	fmt.Printf("   Average Loss:         $%.2f\n", result.AverageLoss)
	
	if result.AverageLoss > 0 {
		profitFactor := result.AverageWin / result.AverageLoss
		fmt.Printf("   Profit Factor:        %.2f\n", profitFactor)
	}
	
	// Show recent trades
	fmt.Printf("\n📋 RECENT TRADES (Last 10)\n")
	recentTrades := result.Trades
	if len(recentTrades) > 10 {
		recentTrades = recentTrades[len(recentTrades)-10:]
	}
	
	for _, trade := range recentTrades {
		emoji := "🟢"
		if trade.Type == "SELL" {
			emoji = "🔴"
		}
		fmt.Printf("   %s %s %.6f %s at $%.2f (%s)\n", 
			emoji, trade.Type, trade.Quantity, trade.Symbol, 
			trade.Price, trade.Timestamp.Format("2006-01-02 15:04"))
	}
	
	fmt.Println(strings.Repeat("=", 80))
	
	// Performance rating
	var rating string
	var ratingEmoji string
	
	if result.TotalReturnPct > result.BuyAndHoldReturnPct+10 {
		rating = "EXCELLENT"
		ratingEmoji = "🏆"
	} else if result.TotalReturnPct > result.BuyAndHoldReturnPct {
		rating = "GOOD"
		ratingEmoji = "✅"
	} else if result.TotalReturnPct > -10 {
		rating = "FAIR"
		ratingEmoji = "⚠️"
	} else {
		rating = "POOR"
		ratingEmoji = "❌"
	}
	
	fmt.Printf("%s STRATEGY RATING: %s\n", ratingEmoji, rating)
	fmt.Println(strings.Repeat("=", 80))
}
