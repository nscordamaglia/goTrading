# 🚀 GoTrading Bot - Cryptocurrency Trading Signals

A Go-based cryptocurrency trading bot that analyzes market data using technical indicators and sends signals to Telegram.

## Features

- 📊 **Real-time Analysis**: Fetches live price data from Binance API
- 🔍 **Technical Indicators**: EMA, RSI, and MACD analysis
- 🤖 **Telegram Integration**: Sends trading signals directly to your Telegram chat
- 📈 **Multiple Pairs**: Supports analysis of multiple cryptocurrency pairs simultaneously
- ⚙️ **Configurable**: Customizable intervals and signal parameters

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
