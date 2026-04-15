import Exchange from './abstract/dreamdex.js';
import type { Dict, int, Int, Str, Strings, Num, Market, Currencies, Order, OrderType, OrderSide, Balances, OrderBook, Ticker, Tickers, Trade, OHLCV } from './base/types.js';
/**
 * @class dreamdex
 * @augments Exchange
 * @description Dreamdex (Somnia DEX) - a non-custodial decentralized exchange on the Somnia network (chain ID 50312).
 * createOrder returns an unsigned EVM transaction for the user to sign and broadcast on-chain.
 * Note: Somnia is currently in testnet. The API base URL points to the testnet environment and will be
 * updated to the production URL once mainnet launches.
 */
export default class dreamdex extends Exchange {
    describe(): any;
    /**
     * @method
     * @name dreamdex#fetchCurrencies
     * @description fetches all available currencies on the exchange
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @returns {object} an associative dictionary of currencies
     */
    fetchCurrencies(params?: {}): Promise<Currencies>;
    /**
     * @method
     * @name dreamdex#fetchMarkets
     * @description retrieves data on all markets for dreamdex
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @returns {Market[]} an array of objects representing market data
     */
    fetchMarkets(params?: {}): Promise<Market[]>;
    parseMarket(market: Dict): Market;
    /**
     * @method
     * @name dreamdex#fetchOrderBook
     * @description fetches information on open orders with bid (buy) and ask (sell) prices, volumes and other data
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {string} symbol unified market symbol
     * @param {int} [limit] the maximum amount of order book entries to return
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @returns {object} A dictionary of [order book structures]{@link https://docs.ccxt.com/#/?id=order-book-structure} indexed by market symbols
     */
    fetchOrderBook(symbol: string, limit?: Int, params?: {}): Promise<OrderBook>;
    /**
     * @method
     * @name dreamdex#fetchTicker
     * @description fetches a price ticker, a statistical calculation with the information for a specific market
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {string} symbol unified market symbol
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @returns {object} a [ticker structure]{@link https://docs.ccxt.com/#/?id=ticker-structure}
     */
    fetchTicker(symbol: string, params?: {}): Promise<Ticker>;
    /**
     * @method
     * @name dreamdex#fetchTickers
     * @description fetches price tickers for multiple markets
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {string[]|undefined} [symbols] unified market symbols to fetch tickers for, all tickers are returned if not specified
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @returns {object} a dictionary of [ticker structures]{@link https://docs.ccxt.com/#/?id=ticker-structure}
     */
    fetchTickers(symbols?: Strings, params?: {}): Promise<Tickers>;
    parseTicker(ticker: Dict, market?: Market): Ticker;
    /**
     * @method
     * @name dreamdex#fetchTrades
     * @description get the list of most recent trades for a particular symbol
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {string} symbol unified market symbol
     * @param {int} [since] timestamp in ms of the earliest trade to fetch
     * @param {int} [limit] the maximum number of trades to fetch
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @returns {Trade[]} a list of [trade structures]{@link https://docs.ccxt.com/#/?id=trade-structure}
     */
    fetchTrades(symbol: string, since?: Int, limit?: Int, params?: {}): Promise<Trade[]>;
    /**
     * @method
     * @name dreamdex#fetchMyTrades
     * @description fetch all trades made by the user
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {string} symbol unified market symbol, required for dreamdex
     * @param {int} [since] timestamp in ms of the earliest trade to fetch
     * @param {int} [limit] the maximum number of trades to fetch
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @returns {Trade[]} a list of [trade structures]{@link https://docs.ccxt.com/#/?id=trade-structure}
     */
    fetchMyTrades(symbol?: Str, since?: Int, limit?: Int, params?: {}): Promise<Trade[]>;
    parseTrade(trade: Dict, market?: Market): Trade;
    /**
     * @method
     * @name dreamdex#fetchOHLCV
     * @description fetches historical candlestick data containing the open, high, low, close price, and the volume of a market
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {string} symbol unified market symbol
     * @param {string} [timeframe] the length of time each candle represents, default '1m'
     * @param {int} [since] timestamp in ms of the earliest candle to fetch
     * @param {int} [limit] the maximum number of candles to fetch
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @returns {int[][]} A list of candles ordered as timestamp, open, high, low, close, volume
     */
    fetchOHLCV(symbol: string, timeframe?: string, since?: Int, limit?: Int, params?: {}): Promise<OHLCV[]>;
    parseOHLCV(ohlcv: any, market?: Market): OHLCV;
    /**
     * @method
     * @name dreamdex#fetchBalance
     * @description query for balance in a specific market vault. DreamDEX uses per-market vaults rather than a single exchange-wide wallet, so params.symbol is required. The API does not distinguish between free and locked (in-order) balances, so all balance is reported as free.
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @param {string} params.symbol unified market symbol (required — vault is per-market)
     * @returns {object} a [balance structure]{@link https://docs.ccxt.com/#/?id=balance-structure}
     */
    fetchBalance(params?: {}): Promise<Balances>;
    parseBalance(response: any): Balances;
    /**
     * @method
     * @name dreamdex#vaultApprove
     * @description generates an unsigned EVM transaction that approves the pool contract to spend a token on behalf of the wallet.
     * Must be called before vaultDeposit. DreamDEX uses per-market vaults: each trading pair has its own vault contract
     * that holds deposited tokens. This differs from centralized exchanges where deposit/withdraw are exchange-wide.
     * The approve step (ERC-20 allowance) has no equivalent in the standard CCXT unified interface.
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {string} symbol unified market symbol identifying the vault
     * @param {string} currency currency code to approve (e.g. 'SOM' or 'USDC')
     * @param {float} amount the amount to approve for spending
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @param {string} [params.walletAddress] the wallet address (defaults to this.walletAddress)
     * @returns {object} an unsigned EVM transaction { to, data, value, chainId, gasLimit, nonce }
     */
    vaultApprove(symbol: string, currency: string, amount: Num, params?: {}): Promise<Dict>;
    /**
     * @method
     * @name dreamdex#vaultDeposit
     * @description generates an unsigned EVM transaction for depositing tokens into a per-market vault.
     * The token must first be approved via vaultApprove. DreamDEX vaults are per-market (each trading pair
     * has its own vault contract), unlike centralized exchanges where funds are deposited exchange-wide.
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {string} symbol unified market symbol identifying the vault
     * @param {string} currency currency code to deposit (e.g. 'SOM' or 'USDC')
     * @param {float} amount the amount to deposit
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @param {string} [params.walletAddress] the wallet address (defaults to this.walletAddress)
     * @returns {object} an unsigned EVM transaction { to, data, value, chainId, gasLimit, nonce }
     */
    vaultDeposit(symbol: string, currency: string, amount: Num, params?: {}): Promise<Dict>;
    /**
     * @method
     * @name dreamdex#vaultWithdraw
     * @description generates an unsigned EVM transaction for withdrawing tokens from a per-market vault back to the wallet.
     * DreamDEX vaults are per-market (each trading pair has its own vault contract), unlike centralized
     * exchanges where withdrawals are exchange-wide.
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {string} symbol unified market symbol identifying the vault
     * @param {string} currency currency code to withdraw (e.g. 'SOM' or 'USDC')
     * @param {float} amount the amount to withdraw
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @param {string} [params.walletAddress] the wallet address (defaults to this.walletAddress)
     * @returns {object} an unsigned EVM transaction { to, data, value, chainId, gasLimit, nonce }
     */
    vaultWithdraw(symbol: string, currency: string, amount: Num, params?: {}): Promise<Dict>;
    vaultAction(action: string, symbol: string, currency: string, amount: Num, params?: {}): Promise<Dict>;
    /**
     * @method
     * @name dreamdex#createOrder
     * @description creates an order by returning an unsigned EVM transaction for the user to sign and broadcast on-chain.
     * The order is not placed until the transaction is submitted to the Somnia network (chain ID 50312).
     * The returned order structure has the unsigned transaction payload in the info field.
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {string} symbol unified market symbol
     * @param {string} type 'limit' or 'market'
     * @param {string} side 'buy' or 'sell'
     * @param {float} amount how much of currency you want to trade in units of base currency
     * @param {float} [price] the price at which the order is to be fulfilled, in units of the quote currency, ignored in market orders
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @param {string} [params.walletAddress] the wallet address to place the order from (defaults to this.walletAddress)
     * @param {float} [params.triggerPrice] the price at which a stop order is triggered - routes to the stop-orders endpoint
     * @param {float} [params.stopPrice] alias for triggerPrice
     * @param {string} [params.triggerOperator] 'gte' or 'lte' - trigger condition (default: 'lte' for sell, 'gte' for buy)
     * @param {string} [params.timeInForce] 'IOC', 'FOK', or 'PO' - maps to API orderType (immediateOrCancel, fillOrKill, postOnly)
     * @param {bool} [params.postOnly] true to create a post-only order (alternative to timeInForce 'PO')
     * @param {string} [params.fundingSource] 'wallet' or 'vault' - where to source tokens (default is 'wallet', 'vault' uses pre-deposited balance)
     * @param {string} [params.selfMatchingOption] 'cancelTaker' or 'cancelMaker' - self-trade prevention behavior
     * @returns {object} an [order structure]{@link https://docs.ccxt.com/#/?id=order-structure} with the unsigned EVM transaction in the info field
     */
    createOrder(symbol: string, type: OrderType, side: OrderSide, amount: number, price?: Num, params?: {}): Promise<Order>;
    /**
     * @method
     * @name dreamdex#fetchOrder
     * @description fetches information on an order made by the user
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {string} id the order id
     * @param {string} symbol unified market symbol, required for dreamdex
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @returns {object} an [order structure]{@link https://docs.ccxt.com/#/?id=order-structure}
     */
    fetchOrder(id: string, symbol?: Str, params?: {}): Promise<Order>;
    /**
     * @method
     * @name dreamdex#fetchOrders
     * @description fetches a list of orders placed by the user
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {string} [symbol] unified market symbol; when omitted returns orders across all markets
     * @param {int} [since] timestamp in ms of the earliest order to retrieve
     * @param {int} [limit] the maximum number of orders to retrieve
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @param {string} [params.status] order status to filter by: 'open', 'closed', 'canceled', 'expired', 'rejected' (or 'pending', 'triggered', 'cancelled', 'failed' for stop orders)
     * @param {bool} [params.stop] set to true to fetch stop orders instead of regular orders (requires symbol)
     * @param {bool} [params.trigger] alias for params.stop
     * @returns {Order[]} a list of [order structures]{@link https://docs.ccxt.com/#/?id=order-structure}
     */
    fetchOrders(symbol?: Str, since?: Int, limit?: Int, params?: {}): Promise<Order[]>;
    /**
     * @method
     * @name dreamdex#fetchOpenOrders
     * @description fetches a list of open orders placed by the user
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {string} [symbol] unified market symbol; when omitted returns open orders across all markets
     * @param {int} [since] timestamp in ms of the earliest order to retrieve
     * @param {int} [limit] the maximum number of orders to retrieve
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @param {bool} [params.stop] set to true to fetch pending stop orders (requires symbol)
     * @param {bool} [params.trigger] alias for params.stop
     * @returns {Order[]} a list of [order structures]{@link https://docs.ccxt.com/#/?id=order-structure}
     */
    fetchOpenOrders(symbol?: Str, since?: Int, limit?: Int, params?: {}): Promise<Order[]>;
    /**
     * @method
     * @name dreamdex#cancelOrder
     * @description cancels an open order
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {string} id order id
     * @param {string} symbol unified market symbol, required for dreamdex
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @param {bool} [params.stop] set to true to cancel a stop order (returns unsigned EVM transaction)
     * @param {bool} [params.trigger] alias for params.stop
     * @returns {object} an [order structure]{@link https://docs.ccxt.com/#/?id=order-structure}
     */
    cancelOrder(id: string, symbol?: Str, params?: {}): Promise<Order>;
    /**
     * @method
     * @name dreamdex#editOrder
     * @description reduces the remaining quantity of an open order (the only edit the API supports)
     * @see https://dev.dreamdex.somnia.host/v0/.well-known/oapi.json
     * @param {string} id order id
     * @param {string} symbol unified market symbol, required for dreamdex
     * @param {string} type not used, kept for CCXT unified signature
     * @param {string} side not used, kept for CCXT unified signature
     * @param {float} amount the new remaining quantity (must be less than current remaining)
     * @param {float} [price] not supported -- will throw if provided
     * @param {object} [params] extra parameters specific to the exchange API endpoint
     * @returns {object} an [order structure]{@link https://docs.ccxt.com/#/?id=order-structure} with the unsigned EVM transaction in the info field
     */
    editOrder(id: string, symbol: string, type: OrderType, side: OrderSide, amount?: Num, price?: Num, params?: {}): Promise<Order>;
    parseOrder(order: Dict, market?: Market): Order;
    parseOrderStatus(status: Str): string;
    parseStopOrder(order: Dict, market?: Market): Order;
    parseStopOrderStatus(status: Str): string;
    hashMessage(message: any): string;
    signHash(hash: any, privateKey: any): string;
    authenticateRest(params?: {}): Promise<string>;
    sign(path: any, api?: string, method?: string, params?: {}, headers?: any, body?: any): {
        url: string;
        method: string;
        body: any;
        headers: any;
    };
    handleErrors(httpCode: int, reason: string, url: string, method: string, headers: Dict, body: string, response: any, requestHeaders: any, requestBody: any): any;
}
