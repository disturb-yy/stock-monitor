# Quickstart: Tushare 真实行情接入

## 前置条件

1. Go 1.22+
2. 有效的 Tushare Pro API Token（在 https://tushare.pro 注册获取）
3. 项目已 `make build` 编译通过

## 配置

设置环境变量并编辑 `configs/config.yaml`：

export TUSHARE_TOKEN="你的TUSHARE_TOKEN"

```yaml
provider:
  type: tushare           # 从 mock 切换为 tushare

tushare:
  # token 通过环境变量 TUSHARE_TOKEN 设置
  # base_url 和 timeout 使用默认值即可
  indices:
    - symbol: "000001.SH"
      name: "上证指数"
    - symbol: "399001.SZ"
      name: "深证成指"
    - symbol: "399006.SZ"
      name: "创业板指"
    - symbol: "000688.SH"
      name: "科创50"
    - symbol: "000300.SH"
      name: "沪深300"
    - symbol: "000905.SH"
      name: "中证500"
```

## 验证步骤

### 1. 启动服务

```bash
make run
```

预期日志包含服务启动信息，无 panic/error。

### 2. 验证市场状态（不同时段）

```bash
# 交易日 9:30-11:30 或 13:00-15:00
curl -s http://localhost:8080/api/market/status | jq .

# 预期: status="trading", isTrading=true
```

```bash
# 交易日 11:30-13:00
curl -s http://localhost:8080/api/market/status | jq .

# 预期: status="lunch_break", isTrading=false
```

```bash
# 收盘后或周末
curl -s http://localhost:8080/api/market/status | jq .

# 预期: status="closed", isTrading=false
```

### 3. 验证指数行情

```bash
curl -s http://localhost:8080/api/market/indices | jq '.data[:2]'

# 预期: 返回 6 个指数对象，每个包含 price, change, changePercent 等字段
# 涨跌幅与当日公开行情一致（允许小幅偏差）
```

### 4. 验证大盘总览

```bash
curl -s http://localhost:8080/api/market/overview | jq .

# 预期: 包含 status + indices + summary
# summary 含 risingIndexCount, fallingIndexCount, totalAmount
```

### 5. 验证切换回 mock

修改 `configs/config.yaml` 中 `provider.type` 为 `mock`，重启服务：

```bash
curl -s http://localhost:8080/api/market/indices | jq '.data[0].symbol'

# 预期: 返回 mock 数据中的固定指数代码
```

### 6. 验证 Token 错误处理

故意填写无效 Token，重启服务后请求：

```bash
curl -s http://localhost:8080/api/market/indices | jq .

# 预期: 返回错误 JSON，code 非 0，msg 含错误描述
```

## 切换检查清单

- [ ] Tushare Token 已正确配置
- [ ] `provider.type` 设为 `tushare`
- [ ] 服务启动无报错
- [ ] 3 个 API 端点均返回真实数据
- [ ] 切换回 mock 后数据恢复为模拟值
- [ ] Token 无效时返回明确错误
