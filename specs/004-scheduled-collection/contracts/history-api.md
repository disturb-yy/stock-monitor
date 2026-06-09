# History API Contract

## Endpoint

```
GET /api/market/history
```

**Query Parameters**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `start` | string (YYYY-MM-DD) | 是 | 起始日期（含） |
| `end` | string (YYYY-MM-DD) | 否 | 截止日期（含），默认今天 |
| `symbol` | string | 否 | 指数代码，不传返回全部 |

**Auth**: 与 `/api/market/*` 一致。

## Response

### 成功 (200)

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "symbol": "000001.SH",
    "name": "上证指数",
    "quotes": [
      {
        "market": "CN_A",
        "symbol": "000001.SH",
        "name": "上证指数",
        "price": 4027.74,
        "change": -30.04,
        "changePercent": -0.74,
        "open": 4055.0,
        "high": 4060.5,
        "low": 4020.1,
        "preClose": 4057.78,
        "volume": 12345678,
        "amount": 987654321,
        "updatedAt": "2026-06-05T15:00:00+08:00"
      }
    ],
    "count": 1
  }
}
```

### 多指数查询（symbol 不传）

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "indices": {
      "000001.SH": { "name": "上证指数", "quotes": [...], "count": 3 },
      "399001.SZ": { "name": "深证成指", "quotes": [...], "count": 3 }
    }
  }
}
```

### 无数据

```json
{
  "code": 0,
  "msg": "success",
  "data": { "symbol": "000001.SH", "name": "上证指数", "quotes": [], "count": 0 }
}
```

### 参数错误

```json
{
  "code": 1001,
  "msg": "start date is required"
}
```
