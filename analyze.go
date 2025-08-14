package main

import (
    "github.com/sdcoffey/big"
    "github.com/sdcoffey/techan"
)

// analyze produces a simple BUY/SELL/HOLD signal using EMA cross, RSI, and MACD
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


