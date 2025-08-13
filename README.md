# 🚀 GoTrading Bot - Cryptocurrency Trading Signals

A Go-based cryptocurrency trading bot that analyzes market data using technical indicators and sends signals to Telegram.

## Features

- 📊 **Real-time Analysis**: Fetches live price data from Binance API
- 🔍 **Technical Indicators**: EMA, RSI, and MACD analysis
- 🤖 **Telegram Integration**: Sends trading signals directly to your Telegram chat
- 📈 **Multiple Pairs**: Supports analysis of multiple cryptocurrency pairs simultaneously
- ⚙️ **Configurable**: Customizable intervals and signal parameters
- 📈 **Backtesting System**: Test your strategy against historical data
- 📊 **Performance Analytics**: Detailed metrics including Sharpe ratio, drawdown, win rate
- 💰 **Portfolio Simulation**: Realistic trading simulation with fees and slippage

## Setup Instructions

### 1. Clone and Install

```bash
git clone <your-repo>
cd goTrading
go mod tidy
```

### 2. Binance API Setup

1. Create a Binance account at [binance.com](https://binance.com)
2. Go to API Management in your account settings
3. Create a new API key with reading permissions
4. Copy your API Key and Secret Key

### 3. Telegram Bot Setup

#### Step 1: Create a Telegram Bot
1. Open Telegram and search for `@BotFather`
2. Start a chat with BotFather and send `/newbot`
3. Follow the prompts to create your bot:
   - Choose a name for your bot (e.g., "My Trading Bot")
   - Choose a username (must end in 'bot', e.g., "my_trading_bot")
4. BotFather will give you a **Bot Token** - save this!

#### Step 2: Get Your Chat ID
1. Start a chat with your new bot (click the link BotFather provided)
2. Send any message to your bot (e.g., "Hello")
3. Open this URL in your browser (replace YOUR_BOT_TOKEN):
   ```
   https://api.telegram.org/botYOUR_BOT_TOKEN/getUpdates
   ```
4. Look for the `"chat":{"id":` field - this number is your **Chat ID**

#### Step 3: Test the Bot (Optional)
Send a test message to verify:
```bash
curl -X POST "https://api.telegram.org/botYOUR_BOT_TOKEN/sendMessage" \
  -H "Content-Type: application/json" \
  -d '{"chat_id": "YOUR_CHAT_ID", "text": "Test message"}'
```

### 4. Configure Environment Variables

Edit the `.env` file with your credentials:

```env
# Binance API Configuration
BINANCE_API_KEY=your_actual_binance_api_key_here
BINANCE_SECRET_KEY=your_actual_binance_secret_key_here

# Telegram Bot Configuration  
TELEGRAM_BOT_TOKEN=1234567890:ABCdefGHIjklMNOpqrsTUVwxyz
TELEGRAM_CHAT_ID=987654321
SEND_ALL_UPDATES=false

# Trading pairs (symbol format for Binance)
TRADING_PAIRS=BTCUSDT,SOLUSDT,ETHUSDT,FLOKIUSDT,ALGOUSDT,ONDOUSDT,XRPUSDT
INTERVAL_MINUTES=5
```

### 5. Configuration Options

- **SEND_ALL_UPDATES**: Set to `true` to receive price updates every interval (can be noisy)
- **SEND_ALL_UPDATES**: Set to `false` to only receive BUY/SELL signals (recommended)
- **INTERVAL_MINUTES**: How often to check for signals (default: 5 minutes)
- **TRADING_PAIRS**: Comma-separated list of Binance trading pairs to monitor

## Run the Bot

```bash
go run main.go
```

The bot will:
1. Load historical data for each trading pair
2. Send a startup message to Telegram (if configured)
3. Continuously monitor prices and analyze signals
4. Send BUY/SELL signals to your Telegram chat when detected

## Telegram Message Examples

### Startup Message
```
🤖 Bot de Trading Iniciado

📊 Analizando pares: BTCUSDT, ETHUSDT, SOLUSDT
⏰ Intervalo: 5 minutos
🔍 Buscando señales de trading...
```

### Buy Signal
```
🚀 SEÑAL DE COMPRA

💰 Par: BTCUSDT
💵 Precio: $45,230.50
⏰ Tiempo: 14:25:30 12/08/2024
```

### Price Update (if SEND_ALL_UPDATES=true)
```
📊 Actualización de Precios

🟢 BTCUSDT: $45,230.50
   📈 Alto: $46,100.00 | 📉 Bajo: $44,800.00
   📊 Cambio: 450.25

🔴 ETHUSDT: $2,845.30
   📈 Alto: $2,890.00 | 📉 Bajo: $2,820.00
   📊 Cambio: -25.50

⏰ 14:25:30 12/08/2024
```

## Technical Indicators Used

- **EMA (Exponential Moving Average)**: 9-period and 21-period crossover
- **RSI (Relative Strength Index)**: 14-period, oversold/overbought levels
- **MACD (Moving Average Convergence Divergence)**: 12/26 period with signal line

## Trading Signals

- **BUY Signal**: EMA9 crosses above EMA21, RSI < 70, MACD > Signal
- **SELL Signal**: EMA9 crosses below EMA21, RSI > 30, MACD < Signal

## 📈 Backtesting System

The bot now includes a comprehensive backtesting system to test your trading strategy against historical data.

### Run a Backtest

```bash
# Basic backtest with default settings
go run . -backtest

# Custom backtest with specific parameters
go run . -backtest -symbol=ETHUSDT -balance=5000 -fee=0.0015

# Test with different intervals and data points
go run . -backtest -symbol=ADAUSDT -interval=1h -limit=1000
```

### Backtest Options

- `-symbol`: Trading pair to test (default: BTCUSDT)
- `-balance`: Initial balance in USD (default: 10000)
- `-fee`: Transaction fee percentage (default: 0.001 = 0.1%)
- `-interval`: Candle interval: 1m, 5m, 15m, 1h, 4h, 1d (default: 15m)
- `-limit`: Number of historical candles to fetch (default: 500)
- `-help`: Show help message

### Example Backtest Results

```
================================================================================
                    BACKTEST RESULTS - BTCUSDT
================================================================================
📊 PERFORMANCE OVERVIEW
   Initial Balance:      $10000.00
   Final Value:          $12750.50
   Total Return:         $2750.50 (27.51%)
   Buy & Hold Return:    $1850.25 (18.50%)
   Alpha vs Buy & Hold:  9.01%
   Max Drawdown:         $1250.00 (10.85%)
   Sharpe Ratio:         1.428
   Duration:             62 days

📈 TRADE STATISTICS
   Total Trades:         24
   Winning Trades:       16
   Losing Trades:        8
   Win Rate:             66.7%
   Average Win:          $285.50
   Average Loss:         $142.75
   Profit Factor:        2.00

📋 RECENT TRADES (Last 10)
   🟢 BUY 0.234567 BTCUSDT at $42750.00 (2024-08-10 14:30)
   🔴 SELL 0.234567 BTCUSDT at $44200.00 (2024-08-12 09:15)
   ...

🏆 STRATEGY RATING: EXCELLENT
================================================================================
```

### Performance Metrics Explained

- **Total Return**: Absolute profit/loss vs initial balance
- **Buy & Hold Return**: What you would have made just buying and holding
- **Alpha**: How much better (or worse) your strategy performed vs buy & hold
- **Max Drawdown**: Largest peak-to-valley loss during the period
- **Sharpe Ratio**: Risk-adjusted return metric (higher is better)
- **Win Rate**: Percentage of profitable trades
- **Profit Factor**: Ratio of total wins to total losses

### Save Results

After running a backtest, you'll be prompted to save results to a file:

```
💾 Save results to file? (y/N): y
✅ Results saved to: backtest_BTCUSDT_20240813_143052.txt
```

The saved file contains the complete results plus a detailed trade log.

## Troubleshooting

### Bot doesn't send messages
1. Check your bot token and chat ID are correct
2. Make sure you've started a conversation with your bot
3. Check the logs for error messages

### API errors
1. Verify your Binance API credentials
2. Ensure your API key has reading permissions
3. Check for rate limiting messages

### No signals detected
- The bot needs sufficient historical data (26+ candles)
- Signals are only generated on crossovers, not static conditions
- Try different trading pairs or intervals

## Security Notes

- ⚠️ **Never share your API keys or bot tokens**
- 🔒 Keep your `.env` file private and don't commit it to version control
- 📊 This bot is for analysis only - it doesn't execute trades
- 💡 Always do your own research before making trading decisions

## Support

This is an educational project. Trading involves risk - only invest what you can afford to lose.
