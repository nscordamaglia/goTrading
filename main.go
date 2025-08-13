package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

// Binance API structures
type BinanceKline struct {
	OpenTime                 int64  `json:"0"`
	Open                     string `json:"1"`
	High                     string `json:"2"`
	Low                      string `json:"3"`
	Close                    string `json:"4"`
	Volume                   string `json:"5"`
	CloseTime                int64  `json:"6"`
	QuoteAssetVolume         string `json:"7"`
	NumberOfTrades           int    `json:"8"`
	TakerBuyBaseAssetVolume  string `json:"9"`
	TakerBuyQuoteAssetVolume string `json:"10"`
	Ignore                   string `json:"11"`
}

type BinanceTicker struct {
	Symbol        string `json:"symbol"`
	LastPrice     string `json:"lastPrice"`
	PriceChange   string `json:"priceChange"`
	PrevClosePrice string `json:"prevClosePrice"`
	HighPrice     string `json:"highPrice"`
	LowPrice      string `json:"lowPrice"`
	WeightedAvg   string `json:"weightedAvgPrice"`
}

type BinanceClient struct {
	apiKey    string
	secretKey string
	baseURL   string
}

type TelegramBot struct {
	botToken string
	chatID   string
}

var (
	seriesMap = make(map[string]*techan.TimeSeries)
	binanceClient *BinanceClient
	telegramBot *TelegramBot
	sendAllUpdates bool
)

func NewBinanceClient(apiKey, secretKey string) *BinanceClient {
	return &BinanceClient{
		apiKey:    apiKey,
		secretKey: secretKey,
		baseURL:   "https://api.binance.com",
	}
}

func (bc *BinanceClient) signRequest(params string) string {
	h := hmac.New(sha256.New, []byte(bc.secretKey))
	h.Write([]byte(params))
	return hex.EncodeToString(h.Sum(nil))
}

func (bc *BinanceClient) fetchKlines(symbol string, interval string, limit int) ([]BinanceKline, error) {
	url := fmt.Sprintf("%s/api/v3/klines?symbol=%s&interval=%s&limit=%d", 
		bc.baseURL, symbol, interval, limit)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching klines: %v", err)
	}
	defer resp.Body.Close()

	var rawKlines [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawKlines); err != nil {
		return nil, fmt.Errorf("error decoding klines: %v", err)
	}

	var klines []BinanceKline
	for _, raw := range rawKlines {
		if len(raw) < 12 {
			continue
		}
		
		kline := BinanceKline{
			OpenTime:  int64(raw[0].(float64)),
			Open:      raw[1].(string),
			High:      raw[2].(string),
			Low:       raw[3].(string),
			Close:     raw[4].(string),
			Volume:    raw[5].(string),
			CloseTime: int64(raw[6].(float64)),
		}
		klines = append(klines, kline)
	}
	
	return klines, nil
}

func fetchHistoricalData(symbol string) {
	klines, err := binanceClient.fetchKlines(symbol, "15m", 100)
	if err != nil {
		log.Printf("Error obteniendo klines para %s: %v", symbol, err)
		return
	}

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

	seriesMap[symbol] = ts
	log.Printf("Datos históricos cargados para %s (%d velas)", symbol, len(klines))
}

func (bc *BinanceClient) fetch24hrTickers(symbols []string) (map[string]BinanceTicker, error) {
	url := fmt.Sprintf("%s/api/v3/ticker/24hr", bc.baseURL)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching tickers: %v", err)
	}
	defer resp.Body.Close()

	var allTickers []BinanceTicker
	if err := json.NewDecoder(resp.Body).Decode(&allTickers); err != nil {
		return nil, fmt.Errorf("error decoding tickers: %v", err)
	}

	// Filter only requested symbols
	tickers := make(map[string]BinanceTicker)
	for _, ticker := range allTickers {
		for _, symbol := range symbols {
			if ticker.Symbol == symbol {
				tickers[symbol] = ticker
				break
			}
		}
	}
	
	return tickers, nil
}

func fetchCurrentPrices(symbols []string) map[string]BinanceTicker {
	tickers, err := binanceClient.fetch24hrTickers(symbols)
	if err != nil {
		log.Printf("Error obteniendo precios: %v", err)
		return nil
	}
	
	// Debug: print found tickers
	log.Printf("Tickers obtenidos: %d", len(tickers))
	for symbol, ticker := range tickers {
		log.Printf("Debug - %s: LastPrice=%s, High=%s, Low=%s", symbol, ticker.LastPrice, ticker.HighPrice, ticker.LowPrice)
	}
	
	return tickers
}

func NewTelegramBot(botToken, chatID string) *TelegramBot {
	return &TelegramBot{
		botToken: botToken,
		chatID:   chatID,
	}
}

func (tb *TelegramBot) sendMessage(message string) error {
	if tb.botToken == "" || tb.chatID == "" {
		return fmt.Errorf("telegram bot token or chat ID not configured")
	}
	
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", tb.botToken)
	
	payload := map[string]string{
		"chat_id":    tb.chatID,
		"text":       message,
		"parse_mode": "HTML",
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling telegram payload: %v", err)
	}
	
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("error sending telegram message: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status code: %d", resp.StatusCode)
	}
	
	return nil
}

func formatPriceUpdate(symbols []string, tickers map[string]BinanceTicker) string {
	msg := "<b>📊 Actualización de Precios</b>\n\n"
	
	for _, symbol := range symbols {
		symbol = strings.TrimSpace(symbol)
		ticker, exists := tickers[symbol]
		if !exists {
			continue
		}
		
		priceChange, _ := strconv.ParseFloat(ticker.PriceChange, 64)
		emoji := "🔹"
		if priceChange > 0 {
			emoji = "🟢"
		} else if priceChange < 0 {
			emoji = "🔴"
		}
		
		msg += fmt.Sprintf("%s <b>%s</b>: $%s\n", emoji, symbol, ticker.LastPrice)
		msg += fmt.Sprintf("   📈 Alto: $%s | 📉 Bajo: $%s\n", ticker.HighPrice, ticker.LowPrice)
		msg += fmt.Sprintf("   📊 Cambio: %s\n\n", ticker.PriceChange)
	}
	
	msg += fmt.Sprintf("<i>⏰ %s</i>", time.Now().Format("15:04:05 02/01/2006"))
	return msg
}

func formatSignalMessage(symbol, action, price string) string {
	var emoji, actionText string
	
	switch action {
	case "BUY":
		emoji = "🚀"
		actionText = "SEÑAL DE COMPRA"
	case "SELL":
		emoji = "🔻"
		actionText = "SEÑAL DE VENTA"
	default:
		return "" // Don't send HOLD signals
	}
	
	msg := fmt.Sprintf("<b>%s %s</b>\n\n", emoji, actionText)
	msg += fmt.Sprintf("💰 <b>Par:</b> %s\n", symbol)
	msg += fmt.Sprintf("💵 <b>Precio:</b> $%s\n", price)
	msg += fmt.Sprintf("⏰ <b>Tiempo:</b> %s", time.Now().Format("15:04:05 02/01/2006"))
	
	return msg
}

func analyze(symbol string, ts *techan.TimeSeries) string {
	closePrices := techan.NewClosePriceIndicator(ts)
	emaShort := techan.NewEMAIndicator(closePrices, 9)
	emaLong := techan.NewEMAIndicator(closePrices, 21)

	rsi := techan.NewRelativeStrengthIndexIndicator(closePrices, 14)

	macd := techan.NewMACDIndicator(closePrices, 12, 26)
	macdSignal := techan.NewMACDHistogramIndicator(macd, 9)

	lastIdx := ts.LastIndex()
	if lastIdx < 26 {
		return "WAIT"
	}

	emaShortNow := emaShort.Calculate(lastIdx)
	emaLongNow := emaLong.Calculate(lastIdx)
	emaShortPrev := emaShort.Calculate(lastIdx - 1)
	emaLongPrev := emaLong.Calculate(lastIdx - 1)

	rsiVal := rsi.Calculate(lastIdx)
	macdVal := macd.Calculate(lastIdx)
	macdSignalVal := macdSignal.Calculate(lastIdx)

	if emaShortNow.GT(emaLongNow) && emaShortPrev.LTE(emaLongPrev) &&
		rsiVal.LT(big.NewDecimal(70)) &&
		macdVal.GT(macdSignalVal) {
		return "BUY"
	}

	if emaShortNow.LT(emaLongNow) && emaShortPrev.GTE(emaLongPrev) &&
		rsiVal.GT(big.NewDecimal(30)) &&
		macdVal.LT(macdSignalVal) {
		return "SELL"
	}

	return "HOLD"
}

func main() {
	// Check for backtest flag first before parsing
	for _, arg := range os.Args[1:] {
		if arg == "-backtest" {
			RunBacktestCLI()
			return
		}
	}
	
	// If not backtest, parse flags normally
	backtestFlag := flag.Bool("backtest", false, "Run backtest mode")
	flag.Parse()
	
	if *backtestFlag {
		RunBacktestCLI()
		return
	}

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error cargando .env")
	}

	// Initialize Binance client
	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_SECRET_KEY")
	if apiKey == "" || secretKey == "" {
		log.Fatal("BINANCE_API_KEY and BINANCE_SECRET_KEY must be set in .env file")
	}
	binanceClient = NewBinanceClient(apiKey, secretKey)

	symbols := strings.Split(os.Getenv("TRADING_PAIRS"), ",")
	intervalMin, _ := strconv.Atoi(os.Getenv("INTERVAL_MINUTES"))
	if intervalMin == 0 {
		intervalMin = 5 // default 5 minutes
	}

	// Initialize Telegram bot (optional)
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	sendAllUpdatesStr := strings.ToLower(os.Getenv("SEND_ALL_UPDATES"))
	sendAllUpdates = sendAllUpdatesStr == "true"
	
	if botToken != "" && chatID != "" && botToken != "your_bot_token_here" && chatID != "your_chat_id_here" {
		telegramBot = NewTelegramBot(botToken, chatID)
		log.Printf("Telegram bot configurado - Enviará señales a chat ID: %s", chatID)
		
		// Send startup message
		startupMsg := "🤖 <b>Bot de Trading Iniciado</b>\n\n"
		startupMsg += "📊 Analizando pares: " + strings.Join(symbols, ", ") + "\n"
		startupMsg += fmt.Sprintf("⏰ Intervalo: %d minutos\n", intervalMin)
		startupMsg += "🔍 Buscando señales de trading..."
		
		if err := telegramBot.sendMessage(startupMsg); err != nil {
			log.Printf("Error enviando mensaje de inicio a Telegram: %v", err)
		} else {
			log.Println("Mensaje de inicio enviado a Telegram")
		}
	} else {
		log.Println("Telegram bot no configurado - solo logs locales")
	}

	log.Printf("Iniciando bot de trading con Binance API...")
	log.Printf("Pares a analizar: %v", symbols)
	log.Printf("Intervalo: %d minutos", intervalMin)

	// Cargar datos históricos iniciales
	for _, symbol := range symbols {
		symbol = strings.TrimSpace(symbol)
		log.Printf("Cargando datos históricos para %s...", symbol)
		fetchHistoricalData(symbol)
		time.Sleep(100 * time.Millisecond) // Small delay to avoid rate limits
	}

	// Loop principal
	for {
		log.Println("\n=== Consultando precios actuales ===")
		tickers := fetchCurrentPrices(symbols)
		
		// Send price updates to Telegram if enabled
		if telegramBot != nil && sendAllUpdates && len(tickers) > 0 {
			priceUpdateMsg := formatPriceUpdate(symbols, tickers)
			if err := telegramBot.sendMessage(priceUpdateMsg); err != nil {
				log.Printf("Error enviando actualización de precios a Telegram: %v", err)
			}
		}
		
		for _, symbol := range symbols {
			symbol = strings.TrimSpace(symbol)
			ticker, exists := tickers[symbol]
			if !exists {
				log.Printf("No se encontró precio para %s", symbol)
				continue
			}
			
			ts := seriesMap[symbol]
			if ts == nil {
				log.Printf("No hay datos históricos para %s", symbol)
				continue
			}
			
			// Add current price as new candle
			price, err := strconv.ParseFloat(ticker.LastPrice, 64)
			if err != nil {
				log.Printf("Error parsing price for %s: %v (raw: %s)", symbol, err, ticker.LastPrice)
				continue
			}
			
			high, err := strconv.ParseFloat(ticker.HighPrice, 64)
			if err != nil {
				log.Printf("Error parsing high price for %s: %v (raw: %s)", symbol, err, ticker.HighPrice)
				continue
			}
			
			low, err := strconv.ParseFloat(ticker.LowPrice, 64)
			if err != nil {
				log.Printf("Error parsing low price for %s: %v (raw: %s)", symbol, err, ticker.LowPrice)
				continue
			}
			
			period := techan.NewTimePeriod(time.Now(), time.Minute*15)
			c := techan.NewCandle(period)
			c.OpenPrice = big.NewDecimal(price)  // Simplified
			c.MaxPrice = big.NewDecimal(high)
			c.MinPrice = big.NewDecimal(low)
			c.ClosePrice = big.NewDecimal(price)
			c.Volume = big.NewDecimal(0)
			ts.AddCandle(c)

			action := analyze(symbol, ts)
			log.Printf("[%s] Precio: $%s → Señal: %s", symbol, ticker.LastPrice, action)
			
			// Display additional info for buy/sell signals
			if action == "BUY" {
				log.Printf("🚀 SEÑAL DE COMPRA detectada para %s", symbol)
				
				// Send signal to Telegram
				if telegramBot != nil {
					signalMsg := formatSignalMessage(symbol, action, ticker.LastPrice)
					if err := telegramBot.sendMessage(signalMsg); err != nil {
						log.Printf("Error enviando señal BUY a Telegram: %v", err)
					}
				}
			} else if action == "SELL" {
				log.Printf("🔻 SEÑAL DE VENTA detectada para %s", symbol)
				
				// Send signal to Telegram
				if telegramBot != nil {
					signalMsg := formatSignalMessage(symbol, action, ticker.LastPrice)
					if err := telegramBot.sendMessage(signalMsg); err != nil {
						log.Printf("Error enviando señal SELL a Telegram: %v", err)
					}
				}
			}
		}
		
		log.Printf("\nEsperando %d minutos antes de la próxima consulta...\n", intervalMin)
		time.Sleep(time.Duration(intervalMin) * time.Minute)
	}
}
