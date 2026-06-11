#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# Stock Monitor API 验证脚本
# 覆盖所有接口：基础、行情、异动、告警、健康检查
# 用法: bash scripts/verify_api.sh [base_url]
# 默认 base_url = http://localhost:30083
# ============================================================

BASE_URL="${1:-http://localhost:30083}"
PASS=0
FAIL=0
RESULTS=()

# ── 颜色输出 ──
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

pass() { PASS=$((PASS+1)); RESULTS+=("✅ $1"); echo -e "${GREEN}✅ PASS${NC} $1"; }
fail() { FAIL=$((FAIL+1)); RESULTS+=("❌ $1 (got: $2)"); echo -e "${RED}❌ FAIL${NC} $1 → $2"; }
info() { echo ""; echo -e "${CYAN}── ${1}${NC}"; }

# ── 辅助函数 ──
check_code() {
  local resp="$1" expected="$2"
  local actual=$(echo "$resp" | jq -r '.code // empty' 2>/dev/null)
  if [[ "$actual" == "$expected" ]]; then echo "true"; else echo "false"; fi
}

echo -e "${YELLOW}═══════════════════════════════════════════"
echo "  Stock Monitor API Verification"
echo "  Target: $BASE_URL"
echo "═══════════════════════════════════════════${NC}"

# ============================================================
# 场景 1: 健康检查公共端点 (无认证)
# ============================================================
info "场景1: 健康检查公共端点"

# 1.1 GET /v1/index
resp=$(curl -s -w "\n%{http_code}" "$BASE_URL/v1/index")
body=$(echo "$resp" | sed '$d')
code=$(echo "$resp" | tail -1)
if [[ "$code" == "200" ]] && [[ "$(check_code "$body" 0)" == "true" ]]; then
  svc=$(echo "$body" | jq -r '.data.service // empty')
  pass "GET /v1/index → 200, service=$svc"
else
  fail "GET /v1/index" "HTTP $code body=$body"
fi

# 1.2 GET /v1/ready
resp=$(curl -s -w "\n%{http_code}" "$BASE_URL/v1/ready")
code=$(echo "$resp" | tail -1)
if [[ "$code" == "200" ]]; then
  pass "GET /v1/ready → 200"
else
  fail "GET /v1/ready" "HTTP $code"
fi

# 1.3 GET /v1/heart
resp=$(curl -s -w "\n%{http_code}" "$BASE_URL/v1/heart")
code=$(echo "$resp" | tail -1)
if [[ "$code" == "200" ]]; then
  pass "GET /v1/heart → 200"
else
  fail "GET /v1/heart" "HTTP $code"
fi

# ============================================================
# 场景 2: 认证接口 POST /api/auth/token
# ============================================================
info "场景2: 认证接口"

# 2.1 正常获取 token
resp=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/token" \
  -H "Content-Type: application/json" \
  -d '{"subject":"test-user"}')
body=$(echo "$resp" | sed '$d')
code=$(echo "$resp" | tail -1)
TOKEN=""
if [[ "$code" == "200" ]] && [[ "$(check_code "$body" 0)" == "true" ]]; then
  TOKEN=$(echo "$body" | jq -r '.data.token // empty')
  pass "POST /api/auth/token → 200, token issued"
else
  fail "POST /api/auth/token" "HTTP $code"
fi

# 2.2 缺少 subject
resp=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/token" \
  -H "Content-Type: application/json" \
  -d '{}')
body=$(echo "$resp" | sed '$d')
code=$(echo "$resp" | tail -1)
if [[ "$code" == "400" ]] || [[ "$(check_code "$body" 40001)" == "true" ]]; then
  pass "POST /api/auth/token (missing subject) → 400"
else
  fail "POST /api/auth/token (missing subject)" "HTTP $code"
fi

# ============================================================
# 场景 3: 行情状态 GET /api/market/status
# ============================================================
info "场景3: 行情状态"

resp=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/market/status")
body=$(echo "$resp" | sed '$d')
code=$(echo "$resp" | tail -1)
if [[ "$code" == "200" ]] && [[ "$(check_code "$body" 0)" == "true" ]]; then
  session=$(echo "$body" | jq -r '.data.session // empty')
  is_open=$(echo "$body" | jq -r '.data.is_open // empty')
  pass "GET /api/market/status → session=$session, is_open=$is_open"
else
  fail "GET /api/market/status" "HTTP $code"
fi

# ============================================================
# 场景 4: 行情指数 GET /api/market/indices
# ============================================================
info "场景4: 行情指数"

resp=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/market/indices")
body=$(echo "$resp" | sed '$d')
code=$(echo "$resp" | tail -1)
if [[ "$code" == "200" ]] && [[ "$(check_code "$body" 0)" == "true" ]]; then
  count=$(echo "$body" | jq '.data | length')
  pass "GET /api/market/indices → ${count} indices"
else
  fail "GET /api/market/indices" "HTTP $code"
fi

# ============================================================
# 场景 5: 行情概览 GET /api/market/overview
# ============================================================
info "场景5: 行情概览"

resp=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/market/overview")
body=$(echo "$resp" | sed '$d')
code=$(echo "$resp" | tail -1)
if [[ "$code" == "200" ]] && [[ "$(check_code "$body" 0)" == "true" ]]; then
  up=$(echo "$body" | jq -r '.data.up_count // -1')
  down=$(echo "$body" | jq -r '.data.down_count // -1')
  pass "GET /api/market/overview → up=$up down=$down"
else
  fail "GET /api/market/overview" "HTTP $code"
fi

# ============================================================
# 场景 6: 历史行情 GET /api/market/history (正常 + 边界)
# ============================================================
info "场景6: 历史行情"

# 6.1 缺少 start 参数 → 400
resp=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/market/history")
body=$(echo "$resp" | sed '$d')
code=$(echo "$resp" | tail -1)
if [[ "$code" == "400" ]] || [[ "$(check_code "$body" 40001)" == "true" ]]; then
  pass "GET /api/market/history (no start) → 400"
else
  fail "GET /api/market/history (no start)" "HTTP $code"
fi

# 6.2 正常查询 (最近7天)
start=$(date -d "7 days ago" +%Y-%m-%d 2>/dev/null || date -v-7d +%Y-%m-%d 2>/dev/null || echo "2026-06-03")
resp=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/market/history?start=$start")
body=$(echo "$resp" | sed '$d')
code=$(echo "$resp" | tail -1)
if [[ "$code" == "200" ]]; then
  pass "GET /api/market/history?start=$start → 200"
else
  fail "GET /api/market/history?start=$start" "HTTP $code"
fi

# 6.3 按 symbol 过滤
resp=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/market/history?start=$start&symbol=000001.SH")
code=$(echo "$resp" | tail -1)
if [[ "$code" == "200" ]]; then
  pass "GET /api/market/history?start=...&symbol=000001.SH → 200"
else
  fail "GET /api/market/history?symbol=000001.SH" "HTTP $code"
fi

# 6.4 带 end 参数
end=$(date +%Y-%m-%d)
resp=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/market/history?start=$start&end=$end")
code=$(echo "$resp" | tail -1)
if [[ "$code" == "200" ]]; then
  pass "GET /api/market/history?start=...&end=$end → 200"
else
  fail "GET /api/market/history?start&end" "HTTP $code"
fi

# 6.5 不存在的 symbol → 空结果但仍是 200
resp=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/market/history?start=$start&symbol=NOT.EXIST")
code=$(echo "$resp" | tail -1)
if [[ "$code" == "200" ]]; then
  pass "GET /api/market/history?symbol=NOT.EXIST → 200 (empty)"
else
  fail "GET /api/market/history?symbol=NOT.EXIST" "HTTP $code"
fi

# ============================================================
# 场景 7: 异动检测 GET /api/market/anomalies
# ============================================================
info "场景7: 异动检测"

resp=$(curl -s -w "\n%{http_code}" --max-time 35 "$BASE_URL/api/market/anomalies")
body=$(echo "$resp" | sed '$d')
code=$(echo "$resp" | tail -1)
if [[ "$code" == "200" ]] && [[ "$(check_code "$body" 0)" == "true" ]]; then
  count=$(echo "$body" | jq -r '.data.count // -1')
  pass "GET /api/market/anomalies → count=$count"
else
  fail "GET /api/market/anomalies" "HTTP $code"
fi

# ============================================================
# 场景 8: 告警推送历史 GET /api/market/alerts/history
# ============================================================
info "场景8: 告警推送历史"

resp=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/market/alerts/history")
body=$(echo "$resp" | sed '$d')
code=$(echo "$resp" | tail -1)
if [[ "$code" == "200" ]] && [[ "$(check_code "$body" 0)" == "true" ]]; then
  total=$(echo "$body" | jq -r '.data.total // -1')
  pass "GET /api/market/alerts/history → total=$total"
else
  fail "GET /api/market/alerts/history" "HTTP $code"
fi

# ============================================================
# 场景 9: 认证中间件状态确认
# ============================================================
info "场景9: 认证中间件 (observe only)"

resp=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/market/status")
code=$(echo "$resp" | tail -1)
if [[ "$code" == "200" ]]; then
  echo -e "  ${YELLOW}⚠ SKIP${NC}  Auth disabled — 200 OK (启用后应返回 401)"
elif [[ "$code" == "401" ]]; then
  echo -e "  ${GREEN}ℹ OBSERVE${NC}  Auth is enabled, market endpoints require token"
fi

# ============================================================
# 场景 10: 响应格式一致性
# ============================================================
info "场景10: 响应格式一致性"

endpoints=(
  "/v1/index"
  "/v1/ready"
  "/v1/heart"
  "/api/market/status"
  "/api/market/indices"
  "/api/market/overview"
  "/api/market/history?start=2026-06-01"
  "/api/market/anomalies"
  "/api/market/alerts/history"
)

failures=0
for ep in "${endpoints[@]}"; do
  resp=$(curl -s --max-time 35 "$BASE_URL$ep")
  has_code=$(echo "$resp" | jq 'has("code")' 2>/dev/null || echo "false")
  has_msg=$(echo "$resp" | jq 'has("msg")' 2>/dev/null || echo "false")
  if [[ "$has_code" != "true" || "$has_msg" != "true" ]]; then
    failures=$((failures+1))
    echo -e "  ${RED}✗${NC} $ep missing {code, msg}"
  fi
done
if [[ $failures -eq 0 ]]; then
  pass "所有端点响应格式一致 (code + msg)"
else
  fail "响应格式一致性" "$failures non-compliant"
fi

# ============================================================
# 总结
# ============================================================
echo ""
echo -e "${YELLOW}═══════════════════════════════════════════"
TOTAL=$((PASS+FAIL))
echo "  Results: ${GREEN}$PASS passed${NC} / ${RED}$FAIL failed${NC} / $TOTAL total"
echo "═══════════════════════════════════════════${NC}"

if [[ $FAIL -gt 0 ]]; then
  echo ""
  echo -e "${RED}Failure details:${NC}"
  for r in "${RESULTS[@]}"; do
    if [[ "$r" == ❌* ]]; then
      echo "  $r"
    fi
  done
fi

exit $FAIL
