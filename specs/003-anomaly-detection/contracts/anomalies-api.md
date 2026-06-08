# Anomaly Detection API Contract

## Endpoint

```
GET /api/market/anomalies
```

**Auth**: 与 `/api/market/*` 一致（`auth.enabled` 为 true 时需 JWT）。

## Response

### 成功（200）

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "events": [
      {
        "id": "price_up_000001_SH_20260608_103000",
        "type": "price_up",
        "symbol": "000001.SH",
        "indexName": "上证指数",
        "triggerValue": 3.5,
        "threshold": 3.0,
        "description": "上证指数涨幅 3.50% 超过阈值 3.00%",
        "triggeredAt": "2026-06-08T10:30:00+08:00"
      },
      {
        "id": "volume_spike_399006_SZ_20260608_103000",
        "type": "volume_spike",
        "symbol": "399006.SZ",
        "indexName": "创业板指",
        "triggerValue": 2.5,
        "threshold": 2.0,
        "description": "创业板指成交量 2.50 倍于近 5 日均值，超过阈值 2.00",
        "triggeredAt": "2026-06-08T10:30:00+08:00"
      }
    ],
    "count": 2
  }
}
```

### 无异动

```json
{
  "code": 0,
  "msg": "success",
  "data": { "events": [], "count": 0 }
}
```

### 规则引擎禁用

```json
{
  "code": 0,
  "msg": "anomaly detection disabled",
  "data": { "events": [], "count": 0 }
}
```

## 事件类型枚举

| type | 含义 | triggerValue 含义 |
|------|------|-------------------|
| `price_up` | 涨超上限 | 实际涨幅 % |
| `price_down` | 跌超下限 | 实际跌幅 %（绝对值） |
| `volume_spike` | 放量异动 | 实际倍数 |
| `consecutive_up` | 连涨 | 实际连续上涨天数 |
| `consecutive_down` | 连跌 | 实际连续下跌天数 |
