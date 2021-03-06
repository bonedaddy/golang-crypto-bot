// Copyright © 2017 Alessandro Sanino <saninoale@gmail.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package exchanges

import (
	"context"
	"errors"
	"fmt"

	"github.com/adshao/go-binance"
	"github.com/thomasxnguy/golang-crypto-bot/environment"
	"github.com/shopspring/decimal"
)

// BinanceWrapper represents the wrapper for the Binance exchange.
type BinanceWrapper struct {
	api *binance.Client
}

// NewBinanceWrapper creates a generic wrapper of the binance API.
func NewBinanceWrapper(publicKey string, secretKey string) ExchangeWrapper {
	client := binance.NewClient(publicKey, secretKey)
	return BinanceWrapper{
		api: client,
	}
}

// Name returns the name of the wrapped exchange.
func (wrapper BinanceWrapper) Name() string {
	return "binance"
}

func (wrapper BinanceWrapper) String() string {
	return wrapper.Name()
}

// GetMarkets Gets all the markets info.
func (wrapper BinanceWrapper) GetMarkets() ([]*environment.Market, error) {
	binanceMarkets, err := wrapper.api.NewListPricesService().Do(context.Background())
	if err != nil {
		return nil, err
	}
	ret := make([]*environment.Market, len(binanceMarkets))
	for i, market := range binanceMarkets {
		if len(market.Symbol) == 6 {
			quote := market.Symbol[0:2]
			base := market.Symbol[3:5]
			ret[i] = &environment.Market{
				Name:           market.Symbol,
				BaseCurrency:   base,
				MarketCurrency: quote,
			}
		} else {
			panic("Handle this case")
		}
	}

	return ret, nil
}

// GetOrderBook gets the order(ASK + BID) book of a market.
func (wrapper BinanceWrapper) GetOrderBook(market *environment.Market) (*environment.OrderBook, error) {
	binanceOrderBook, err := wrapper.api.NewListOrdersService().Symbol(MarketNameFor(market, wrapper)).Do(context.Background())
	if err != nil {
		return nil, err
	}

	var orderBook environment.OrderBook
	for _, order := range binanceOrderBook {
		qty, err := decimal.NewFromString(order.ExecutedQuantity)
		if err != nil {
			return nil, err
		}

		value, err := decimal.NewFromString(order.Price)
		if err != nil {
			return nil, err
		}

		if order.Type == "ASK" {
			orderBook.Asks = append(orderBook.Asks, environment.Order{
				Quantity: qty,
				Value:    value,
			})
		} else if order.Type == "BID" {
			orderBook.Bids = append(orderBook.Bids, environment.Order{
				Quantity: qty,
				Value:    value,
			})
		}
	}

	return &orderBook, nil
}

// BuyLimit performs a limit buy action.
func (wrapper BinanceWrapper) BuyLimit(market *environment.Market, amount float64, limit float64) (string, error) {
	orderNumber, err := wrapper.api.NewCreateOrderService().Type(binance.OrderTypeLimit).Side(binance.SideTypeBuy).Symbol(MarketNameFor(market, wrapper)).Price(fmt.Sprint(limit)).Quantity(fmt.Sprint(amount)).Do(context.Background())
	return fmt.Sprint(orderNumber.ClientOrderID), err
}

// SellLimit performs a limit sell action.
func (wrapper BinanceWrapper) SellLimit(market *environment.Market, amount float64, limit float64) (string, error) {
	orderNumber, err := wrapper.api.NewCreateOrderService().Type(binance.OrderTypeLimit).Side(binance.SideTypeSell).Symbol(MarketNameFor(market, wrapper)).Price(fmt.Sprint(limit)).Quantity(fmt.Sprint(amount)).Do(context.Background())
	return fmt.Sprint(orderNumber.ClientOrderID), err
}

// GetTicker gets the updated ticker for a market.
func (wrapper BinanceWrapper) GetTicker(market *environment.Market) (*environment.Ticker, error) {
	binanceTicker, err := wrapper.api.NewBookTickerService().Symbol("BTCUSDT").Do(context.Background())
	if err != nil {
		return nil, err
	}

	ask, _ := decimal.NewFromString(binanceTicker.AskPrice)
	bid, _ := decimal.NewFromString(binanceTicker.BidPrice)
	fmt.Println(ask)
	return &environment.Ticker{
		Last: ask, // TODO: find a better way for last value, if any
		Ask:  ask,
		Bid:  bid,
	}, nil
}

// GetMarketSummary gets the current market summary.
func (wrapper BinanceWrapper) GetMarketSummary(market *environment.Market) (*environment.MarketSummary, error) {
	hilo, err := wrapper.api.NewListPriceChangeStatsService().Do(context.Background())
	if err != nil {
		return nil, err
	}

	var binanceSummary *binance.PriceChangeStats

	for _, val := range hilo {
		if val.Symbol == MarketNameFor(market, wrapper) {
			binanceSummary = val
			break
		}
	}

	if binanceSummary == nil {
		return nil, errors.New("Symbol not found")
	}

	ask, _ := decimal.NewFromString(binanceSummary.AskPrice)
	bid, _ := decimal.NewFromString(binanceSummary.BidPrice)
	high, _ := decimal.NewFromString(binanceSummary.HighPrice)
	low, _ := decimal.NewFromString(binanceSummary.LowPrice)
	volume, _ := decimal.NewFromString(binanceSummary.Volume)

	return &environment.MarketSummary{
		Last:   ask,
		Ask:    ask,
		Bid:    bid,
		High:   high,
		Low:    low,
		Volume: volume,
	}, nil
}

// CalculateTradingFees calculates the trading fees for an order on a specified market.
//
//     NOTE: In Binance fees are currently hardcoded.
func (wrapper BinanceWrapper) CalculateTradingFees(market *environment.Market, amount float64, limit float64, orderType TradeType) float64 {
	var feePercentage float64
	if orderType == MakerTrade {
		feePercentage = 0.0010
	} else if orderType == TakerTrade {
		feePercentage = 0.0010
	} else {
		panic("Unknown trade type")
	}

	return amount * limit * feePercentage
}

// CalculateWithdrawFees calculates the withdrawal fees on a specified market.
func (wrapper BinanceWrapper) CalculateWithdrawFees(market *environment.Market, amount float64) float64 {
	panic("Not Implemented")
}

// FeedConnect connects to the feed of the exchange.
func (wrapper BinanceWrapper) FeedConnect() {
	//empty
}

var unsubscribe = make(map[string]chan struct{})
var unsubscribed = make(map[string]chan struct{})

// SubscribeMarketSummaryFeed subscribes to the Market Summary Feed service.
func (wrapper BinanceWrapper) SubscribeMarketSummaryFeed(market *environment.Market, onUpdate func(environment.MarketSummary)) {
	doneC, stopC, err := binance.WsMarketStatServe(MarketNameFor(market, wrapper), func(event *binance.WsMarketStatEvent) {
		high, _ := decimal.NewFromString(event.HighPrice)
		low, _ := decimal.NewFromString(event.LowPrice)
		ask, _ := decimal.NewFromString(event.AskPrice)
		bid, _ := decimal.NewFromString(event.BidPrice)
		last, _ := decimal.NewFromString(event.LastPrice)
		volume, _ := decimal.NewFromString(event.BaseVolume)

		onUpdate(environment.MarketSummary{
			High:   high,
			Low:    low,
			Ask:    ask,
			Bid:    bid,
			Last:   last,
			Volume: volume,
		})
	}, func(error) {})

	if err != nil {
		panic(err)
	}

	unsubscribe[MarketNameFor(market, wrapper)] = stopC
	unsubscribed[MarketNameFor(market, wrapper)] = doneC
}

// UnsubscribeMarketSummaryFeed unsubscribes from the Market Summary Feed service.
func (wrapper BinanceWrapper) UnsubscribeMarketSummaryFeed(market *environment.Market) {
	tickerKey := MarketNameFor(market, wrapper)

	unsubscribe[tickerKey] <- struct{}{}

	<-unsubscribed[tickerKey]

	close(unsubscribe[tickerKey])
	close(unsubscribed[tickerKey])
	delete(unsubscribe, tickerKey)
	delete(unsubscribed, tickerKey)
}

// GetKlines Gets candlestick bar information
func (wrapper BinanceWrapper) GetKlines(start int64, symbol string, interval string) (*environment.CandleStickChart, error) {
	trades, err := wrapper.api.NewKlinesService().
		Symbol(symbol).Interval(interval).Limit(100).StartTime(start).
		Do(context.Background())

	if err != nil {
		fmt.Println(err)
		return nil,err
	}
	chart := &environment.CandleStickChart{}
	chart.CandlePeriod = 60 * 60 * 1000
	chart.CandleSticks = make([]environment.CandleStick, len(trades))
	for i, t := range trades {
		c := environment.CandleStick{}
		c.Close, _ = decimal.NewFromString(t.Close)
		c.High, _ = decimal.NewFromString(t.High)
		c.Low, _ = decimal.NewFromString(t.Low)
		c.Open, _ = decimal.NewFromString(t.Open)
		c.Volume, _ = decimal.NewFromString(t.Volume)
		c.TradeNb = t.TradeNum
		c.OpenTime = t.OpenTime/1000
		chart.CandleSticks[i] = c
	}

	return chart, nil
}
