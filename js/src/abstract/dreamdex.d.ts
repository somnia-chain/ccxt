import { implicitReturnType } from '../base/types.js';
import { Exchange as _Exchange } from '../base/Exchange.js';
interface Exchange {
    publicGetV0Currencies(params?: {}): Promise<implicitReturnType>;
    publicGetV0Markets(params?: {}): Promise<implicitReturnType>;
    publicGetV0MarketsSymbolTickers(params?: {}): Promise<implicitReturnType>;
    publicGetV0MarketsSymbolTrades(params?: {}): Promise<implicitReturnType>;
    publicGetV0MarketsSymbolCandles(params?: {}): Promise<implicitReturnType>;
    publicGetV0Orderbooks(params?: {}): Promise<implicitReturnType>;
    publicGetV0Tickers(params?: {}): Promise<implicitReturnType>;
    publicGetV0AuthNonce(params?: {}): Promise<implicitReturnType>;
    publicPostV0AuthLogin(params?: {}): Promise<implicitReturnType>;
    privateGetV0Orders(params?: {}): Promise<implicitReturnType>;
    privateGetV0MarketsSymbolOrders(params?: {}): Promise<implicitReturnType>;
    privateGetV0MarketsSymbolOrdersId(params?: {}): Promise<implicitReturnType>;
    privateGetV0MarketsSymbolTradesMine(params?: {}): Promise<implicitReturnType>;
    privateGetV0MarketsSymbolVaultBalance(params?: {}): Promise<implicitReturnType>;
    privateGetV0MarketsSymbolStopOrders(params?: {}): Promise<implicitReturnType>;
    privatePostV0MarketsSymbolOrders(params?: {}): Promise<implicitReturnType>;
    privatePostV0MarketsSymbolVaultDeposit(params?: {}): Promise<implicitReturnType>;
    privatePostV0MarketsSymbolVaultWithdraw(params?: {}): Promise<implicitReturnType>;
    privatePostV0MarketsSymbolVaultApprove(params?: {}): Promise<implicitReturnType>;
    privatePostV0MarketsSymbolStopOrders(params?: {}): Promise<implicitReturnType>;
    privatePatchV0MarketsSymbolOrdersIdReduce(params?: {}): Promise<implicitReturnType>;
    privateDeleteV0MarketsSymbolOrdersId(params?: {}): Promise<implicitReturnType>;
    privateDeleteV0MarketsSymbolStopOrdersId(params?: {}): Promise<implicitReturnType>;
}
declare abstract class Exchange extends _Exchange {
}
export default Exchange;
