package main

import (
    "github.com/sdcoffey/big"
    "github.com/sdcoffey/techan"
)

// UseMLAnalyze toggles ML-based analysis when true. Defaults to false.
var UseMLAnalyze bool

// analyze dispatches to either the classic rule-based analysis or ML-based analysis.
func analyze(symbol string, ts *techan.TimeSeries) string {
    if UseMLAnalyze {
        return analyzeML(symbol, ts)
    }
    return analyzeClassic(symbol, ts)
}

// analyzeClassic produces a simple BUY/SELL/HOLD signal using EMA cross, RSI, and MACD
func analyzeClassic(symbol string, ts *techan.TimeSeries) string {
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

// analyzeML is a placeholder for future ML-based analysis. Returns HOLD until wired to a model.
func analyzeML(symbol string, ts *techan.TimeSeries) string {
    // TODO: plug in an ML predictor here in the future (e.g., via a global predictor or DI)
    return "HOLD"
}


