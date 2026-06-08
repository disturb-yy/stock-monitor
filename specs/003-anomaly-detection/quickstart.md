# Quickstart: 异动检测规则引擎

## 前置条件

- Tushare Provider 正常工作（001 功能完成）
- 交易日历可用（002 功能完成）

## 配置

`configs/config.yaml` 中新增或确认：

```yaml
anomaly:
  enabled: true
  rules:
    - type: price_change
      upper_threshold: 3.0
      lower_threshold: -2.0
      enabled: true
    - type: volume_spike
      lookback_days: 5
      spike_multiplier: 2.0
      enabled: true
    - type: consecutive
      days: 5
      enabled: true
```

## 验证步骤

### 1. 查询异动事件

```bash
curl -s http://localhost:8080/api/market/anomalies | jq .
```

预期：返回 `events` 数组（可能为空）。

### 2. 验证涨跌幅检测

调低阈值到极值触发检测：

```yaml
rules:
  - type: price_change
    upper_threshold: 0.01    # 几乎任何涨幅都触发
    lower_threshold: -0.01
```

重启后请求：

```bash
curl -s http://localhost:8080/api/market/anomalies | jq '.data.events[] | select(.type | startswith("price"))'
```

### 3. 验证成交量检测

```bash
curl -s http://localhost:8080/api/market/anomalies | jq '.data.events[] | select(.type == "volume_spike")'
```

### 4. 禁用验证

```yaml
anomaly:
  enabled: false
```

重启后请求返回 `"anomaly detection disabled"`。

### 5. 调回正常阈值

恢复原配置，重启即可。
