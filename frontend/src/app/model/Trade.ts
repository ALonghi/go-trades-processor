export default interface Trade {
  trade_id: string;
  entity: string;
  instrument_type: string;
  symbol: string;
  quantity: number;
  price: number | null;
  ts: string;
}
