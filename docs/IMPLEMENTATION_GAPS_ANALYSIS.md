# Genkit-Giskard HTTP Bridge - Implementation Gaps Analysis

## Executive Summary
After reviewing the research document and existing codebase structure, I've identified **17 critical gaps** that need to be addressed in the implementation plan.

---

## 🔴 CRITICAL GAPS

### 1. **Genkit-Gin Integration Strategy**

**Gap**: The research shows Genkit using `server.Start()` from `github.com/firebase/genkit/go/plugins/server`, but the existing boilerplate uses Gin framework. The integration pattern is unclear.

**Research Shows**:
```go
mux := http.NewServeMux()
mux.HandleFunc("POST /customerServiceFlow", genkit.Handler(customerServiceFlow))
server.Start(ctx, "127.0.0.1:8080", mux)
```

**Existing Code Uses**:
```go
router := gin.New()
server := &http.Server{Handler: router}
server.ListenAndServe()
```

**Solution Required**:
- **Option A**: Wrap Genkit handlers for Gin compatibility
  ```go
  func GenkitToGin(flow genkit.Flow) gin.HandlerFunc {
      handler := genkit.Handler(flow)
      return func(c *gin.Context) {
          handler.ServeHTTP(c.Writer, c.Request)
      }
  }
  ```
- **Option B**: Run Genkit on separate port (NOT RECOMMENDED - adds complexity)
- **Option C**: Replace Gin with pure net/http (NOT RECOMMENDED - breaks existing API)

**Recommendation**: Implement Option A with proper middleware compatibility testing.

---

### 2. **Request/Response Format Wrapping**

**Gap**: Genkit expects `{"data": <input>}` and returns `{"result": <output>}`. Current plan doesn't specify WHO handles this wrapping.

**Research Format**:
```json
// Request
{"data": {"message": "...", "context": "..."}}

// Response
{"result": {"response": "...", "tool_calls": [...]}}
```

**Existing Pattern**:
```go
// Current handlers use direct JSON binding
ctx.ShouldBindJSON(&UserRequest)
NewSuccessResponse(ctx, statusCode, "message", data)
```

**Questions to Answer**:
1. Does `genkit.Handler()` automatically wrap/unwrap?
2. Do we need custom middleware for format translation?
3. How does this affect Giskard Python client expectations?

**Solution Required**:
Create a middleware/wrapper that:
- Unwraps `{"data": X}` before passing to Genkit
- Wraps Genkit output as `{"result": X}`
- OR verify genkit.Handler does this automatically

---

### 3. **Genkit Instance Management**

**Gap**: Plan says "create pkg/genkit module" but doesn't address:
- **When** to initialize Genkit (startup vs lazy)
- **Where** to store the instance (global vs dependency injection)
- **How** to inject it into routes/handlers

**Research Shows**:
```go
g, err := genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{}))
// g is needed for DefineFlow
```

**Existing Pattern**:
- Routes receive dependencies via constructor
- Services initialized in `server.NewApp()`
- No global singletons

**Solution Required**:
```go
// In pkg/genkit/genkit.go
type GenkitService struct {
    g *genkit.Genkit
    ctx context.Context
}

func NewGenkitService(ctx context.Context, apiKey string) (*GenkitService, error) {
    g, err := genkit.Init(ctx, genkit.WithPlugins(
        &googlegenai.GoogleAI{APIKey: apiKey},
    ))
    return &GenkitService{g: g, ctx: ctx}, err
}

// In server.NewApp()
genkitService, err := genkit.NewGenkitService(ctx, config.AppConfig.GenkitAPIKey)

// Pass to routes
routes.NewGenkitRoute(api, genkitService).Routes()
```

---

### 4. **Configuration Schema Updates**

**Gap**: Plan mentions adding env vars but doesn't update the Config struct validation.

**Required Changes**:

**File: `internal/config/config.go`**
```go
type Config struct {
    // ... existing fields ...

    // Genkit Configuration
    GenkitAPIKey     string  `mapstructure:"GENKIT_API_KEY"`      // REQUIRED
    GenkitModel      string  `mapstructure:"GENKIT_MODEL"`        // Optional, default: gemini-1.5-flash
    GenkitTemp       float64 `mapstructure:"GENKIT_TEMPERATURE"`  // Optional, default: 0.7
    GenkitEnabled    bool    `mapstructure:"GENKIT_ENABLED"`      // Optional, default: true
}
```

**File: `internal/config/.env.example`**
```bash
# GENKIT AI
GENKIT_API_KEY=your-google-genai-api-key-here
GENKIT_MODEL=gemini-1.5-flash
GENKIT_TEMPERATURE=0.7
GENKIT_ENABLED=true
```

**Validation Update**:
```go
// In InitializeAppConfig()
if AppConfig.GenkitEnabled {
    if AppConfig.GenkitAPIKey == "" {
        return errors.New("GENKIT_API_KEY is required when GENKIT_ENABLED=true")
    }
    if AppConfig.GenkitModel == "" {
        AppConfig.GenkitModel = "gemini-1.5-flash" // default
    }
    if AppConfig.GenkitTemp == 0 {
        AppConfig.GenkitTemp = 0.7 // default
    }
}
```

---

### 5. **Tool Call Extraction Implementation**

**Gap**: Research shows `extractToolCalls()` function but doesn't implement it.

**Research Shows**:
```go
func extractToolCalls(text string) []string {
    // Parse tool calls from response
    // Implementation depends on your format
    return []string{}
}
```

**Required Implementation**:
```go
// File: pkg/genkit/parser.go
package genkit

import (
    "encoding/json"
    "regexp"
)

type ToolCall struct {
    Tool       string                 `json:"tool"`
    Parameters map[string]interface{} `json:"parameters"`
}

func ExtractToolCalls(text string) ([]ToolCall, error) {
    // Pattern: <tool_call>{"tool": "name", "parameters": {...}}</tool_call>
    pattern := regexp.MustCompile(`<tool_call>\s*(\{.*?\})\s*</tool_call>`)
    matches := pattern.FindAllStringSubmatch(text, -1)

    var toolCalls []ToolCall
    for _, match := range matches {
        if len(match) < 2 {
            continue
        }

        var tc ToolCall
        if err := json.Unmarshal([]byte(match[1]), &tc); err != nil {
            return nil, err
        }
        toolCalls = append(toolCalls, tc)
    }

    return toolCalls, nil
}

func ExtractToolNames(text string) []string {
    toolCalls, _ := ExtractToolCalls(text)
    names := make([]string, len(toolCalls))
    for i, tc := range toolCalls {
        names[i] = tc.Tool
    }
    return names
}
```

---

### 6. **Domain Interface Definition**

**Gap**: Plan mentions creating domain models but doesn't define the interfaces following existing patterns.

**Existing Pattern** (`domain.users.go`):
```go
type UserDomain struct { ... }
type UserUsecase interface { ... }
type UserRepository interface { ... }
```

**Required Structure**:

**File: `internal/business/domains/v1/domain.genkit.go`**
```go
package v1

import "context"

// Domain Models
type CustomerQuery struct {
    Message string `json:"message" validate:"required,min=1,max=1000"`
    Context string `json:"context" validate:"max=2000"`
}

type BotResponse struct {
    Response  string   `json:"response"`
    ToolCalls []string `json:"tool_calls,omitempty"`
    Error     string   `json:"error,omitempty"`
}

// Usecase Interface (following existing pattern)
type GenkitUsecase interface {
    ExecuteCustomerServiceFlow(ctx context.Context, query *CustomerQuery) (response BotResponse, statusCode int, err error)
    HealthCheck(ctx context.Context) (healthy bool, err error)
    ListAvailableFlows() []string
}

// Note: No Repository needed - AI calls don't require DB
```

---

### 7. **Flow Definition Location**

**Gap**: Should flow logic live in Usecase or in pkg/genkit?

**Research Shows** (monolithic):
```go
customerServiceFlow := genkit.DefineFlow(g, "customerServiceFlow",
    func(ctx context.Context, input CustomerQuery) (BotResponse, error) {
        // ALL logic here
    })
```

**Better Architecture** (following existing patterns):
```go
// File: internal/business/usecases/v1/usecase.genkit.go
type genkitUsecase struct {
    genkitService *genkit.GenkitService
}

func NewGenkitUsecase(gs *genkit.GenkitService) GenkitUsecase {
    return &genkitUsecase{genkitService: gs}
}

func (uc *genkitUsecase) ExecuteCustomerServiceFlow(ctx context.Context, query *CustomerQuery) (BotResponse, statusCode int, err error) {
    // Define or retrieve flow
    flow := uc.genkitService.GetFlow("customerServiceFlow")

    // Execute
    result, err := flow.Run(ctx, query)
    if err != nil {
        return BotResponse{}, http.StatusInternalServerError, err
    }

    return result, http.StatusOK, nil
}
```

But where does `genkit.DefineFlow()` get called?

**Solution**: Flows should be defined once at initialization in `GenkitService`.

---

### 8. **HTTP Handler Wrapper Pattern**

**Gap**: How to properly wrap Genkit handlers while maintaining existing error response format?

**Existing Pattern**:
```go
func (h UserHandler) Login(ctx *gin.Context) {
    // ... validation ...
    result, statusCode, err := h.usecase.Login(ctx.Request.Context(), domain)
    if err != nil {
        NewErrorResponse(ctx, statusCode, err.Error())
        return
    }
    NewSuccessResponse(ctx, statusCode, "login success", result)
}
```

**Genkit Pattern** (from research):
```go
mux.HandleFunc("POST /customerServiceFlow", genkit.Handler(customerServiceFlow))
```

**Required Wrapper**:
```go
// File: internal/http/handlers/v1/handler.genkit.go
type GenkitHandler struct {
    usecase V1Domains.GenkitUsecase
}

func (h GenkitHandler) ExecuteFlow(ctx *gin.Context) {
    var query requests.CustomerQueryRequest

    // Validation
    if err := ctx.ShouldBindJSON(&query); err != nil {
        NewErrorResponse(ctx, http.StatusBadRequest, err.Error())
        return
    }

    if err := validators.ValidatePayloads(query); err != nil {
        NewErrorResponse(ctx, http.StatusBadRequest, err.Error())
        return
    }

    // Execute via usecase
    domain := query.ToV1Domain()
    result, statusCode, err := h.usecase.ExecuteCustomerServiceFlow(
        ctx.Request.Context(),
        domain,
    )

    if err != nil {
        NewErrorResponse(ctx, statusCode, err.Error())
        return
    }

    NewSuccessResponse(ctx, statusCode, "flow executed successfully", result)
}
```

**Question**: Do we use `genkit.Handler()` at all, or just call flows programmatically?

---

### 9. **Middleware Compatibility**

**Gap**: Will Genkit work with existing middleware (CORS, Auth, Logging)?

**Existing Middleware Stack**:
```go
router.Use(middlewares.CORSMiddleware())
router.Use(gin.LoggerWithFormatter(logger.HTTPLogger))
router.Use(gin.Recovery())

// Per-route
userRoute.Use(authMiddleware)
```

**Scenarios to Test**:
1. Can CORS middleware handle Genkit responses?
2. Does logging middleware capture Genkit request/response?
3. Can we protect Genkit endpoints with JWT auth?
4. Does recovery middleware catch Genkit panics?

**Required Testing**:
- Integration tests for each middleware + Genkit endpoint
- Verify middleware execution order

---

### 10. **Python Bridge - DataFrame Column Mapping**

**Gap**: Research shows specific column names but doesn't validate against API schema.

**Research Shows**:
```python
feature_names=["customer_message", "context"]

# DataFrame
test_data = pd.DataFrame({
    "customer_message": [...],
    "context": [...],
    "expected_tool": [...]
})
```

**Go API Expects** (based on domain):
```json
{
  "message": "...",  // NOT "customer_message"
  "context": "..."
}
```

**Solution Required**:
```python
# File: python/giskard_bridge.py
class GenkitGiskardBridge:
    def __init__(self, service_url, flow_name, column_mapping=None):
        self.column_mapping = column_mapping or {
            "customer_message": "message",
            "context": "context"
        }

    def predict(self, df):
        for _, row in df.iterrows():
            # Map DataFrame columns to API fields
            payload = {
                "data": {
                    self.column_mapping.get(col, col): row[col]
                    for col in row.index
                    if col in self.column_mapping
                }
            }
            # ... make request
```

---

### 11. **Error Response Format Consistency**

**Gap**: Genkit errors vs existing error format.

**Existing Format**:
```json
{
  "success": false,
  "message": "error message",
  "data": null
}
```

**Genkit Might Return**:
```json
{
  "error": "some error",
  "code": "INTERNAL"
}
```

**Solution**: Ensure all Genkit errors are caught and reformatted in handler wrapper.

---

### 12. **Health Check Implementation**

**Gap**: Health check should verify Genkit connection, not just HTTP availability.

**Required**:
```go
func (h GenkitHandler) HealthCheck(ctx *gin.Context) {
    healthy, err := h.usecase.HealthCheck(ctx.Request.Context())
    if err != nil {
        NewErrorResponse(ctx, http.StatusServiceUnavailable, err.Error())
        return
    }

    NewSuccessResponse(ctx, http.StatusOK, "healthy", map[string]interface{}{
        "service": "genkit-go-service",
        "genkit": healthy,
        "timestamp": time.Now().Unix(),
    })
}

// In usecase
func (uc *genkitUsecase) HealthCheck(ctx context.Context) (bool, error) {
    // Try a simple AI call with timeout
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    // Test flow execution
    result, err := uc.genkitService.TestConnection(ctx)
    return result, err
}
```

---

### 13. **Testing Strategy Specifics**

**Gap**: Plan says "create tests" but doesn't specify WHAT to test.

**Required Test Files**:

**`internal/business/usecases/v1/usecase.genkit_test.go`**:
- [ ] TestExecuteCustomerServiceFlow_Success
- [ ] TestExecuteCustomerServiceFlow_InvalidInput
- [ ] TestExecuteCustomerServiceFlow_AIError
- [ ] TestExtractToolCalls_ValidFormat
- [ ] TestExtractToolCalls_InvalidJSON
- [ ] TestExtractToolCalls_NoToolCalls
- [ ] TestHealthCheck_Success
- [ ] TestHealthCheck_Timeout

**`internal/http/handlers/v1/handler.genkit_test.go`**:
- [ ] TestGenkitHandler_ExecuteFlow_Success
- [ ] TestGenkitHandler_ExecuteFlow_ValidationError
- [ ] TestGenkitHandler_ExecuteFlow_UsecaseError
- [ ] TestGenkitHandler_HealthCheck

**`pkg/genkit/genkit_test.go`**:
- [ ] TestNewGenkitService
- [ ] TestDefineFlow
- [ ] TestFlowExecution

**Integration Tests** (new file: `cmd/api/server/server_genkit_test.go`):
- [ ] TestGenkitEndpoint_WithCORS
- [ ] TestGenkitEndpoint_WithAuth
- [ ] TestGenkitEndpoint_WithLogging

---

### 14. **Docker Image Dependencies**

**Gap**: Does the Go Docker image include curl/wget for health checks?

**Research Shows**:
```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
```

**Current Dockerfile** (likely uses scratch or alpine):
```dockerfile
FROM golang:1.19 AS builder
# ... build ...
FROM alpine:latest  # or scratch
COPY --from=builder /app/server .
```

**Solution**:
```dockerfile
FROM alpine:latest
RUN apk add --no-cache curl ca-certificates
COPY --from=builder /app/server .

# OR use wget
healthcheck:
  test: ["CMD", "wget", "--spider", "http://localhost:8080/api/genkit/health"]
```

---

### 15. **Prompt Template Management**

**Gap**: Research hardcodes prompt in flow. Should this be configurable?

**Research Shows**:
```go
ai.WithPrompt(`You are a customer service bot with these tools:
- search_knowledge_base(query): Search help docs
...`)
```

**Better Approach**:
```go
// File: internal/business/usecases/v1/prompts.go
package v1

const CustomerServicePromptTemplate = `You are a customer service bot with these tools:
- search_knowledge_base(query): Search help docs
- check_order_status(order_id): Look up orders
- create_ticket(description): Escalate to human
- issue_refund(order_id, amount): Process refunds

Customer message: %s
Context: %s

When you need a tool, output: <tool_call>{"tool": "name", "parameters": {...}}</tool_call>`

// OR load from file
func LoadPromptTemplate(name string) (string, error) {
    path := fmt.Sprintf("prompts/%s.txt", name)
    return os.ReadFile(path)
}
```

---

### 16. **Genkit Server Plugin Usage**

**Gap**: Is `github.com/firebase/genkit/go/plugins/server` needed at all?

**Research Shows**:
```go
import "github.com/firebase/genkit/go/plugins/server"
server.Start(ctx, "127.0.0.1:8080", mux)
```

**But We're Using Gin**:
- We don't need `server.Start()` - Gin handles HTTP
- We only need `genkit.DefineFlow()` and maybe `genkit.Handler()`

**Verification Needed**:
- Can we use Genkit WITHOUT the server plugin?
- What does the server plugin provide that Gin doesn't?

**Likely Answer**: Server plugin is optional. We only need core Genkit + AI plugin.

---

### 17. **Context Propagation**

**Gap**: Genkit flows need `context.Context` with proper cancellation/timeout.

**Current Pattern**:
```go
func (h UserHandler) Login(ctx *gin.Context) {
    result, _, err := h.usecase.Login(ctx.Request.Context(), domain)
    // ^^^ extracts context.Context from gin.Context
}
```

**Genkit Pattern**:
```go
flow := genkit.DefineFlow(g, "name",
    func(ctx context.Context, input T) (T, error) {
        // ctx must support cancellation
    })
```

**Required**:
- Verify `ctx.Request.Context()` propagates to Genkit
- Add timeout for AI calls:
  ```go
  func (uc *genkitUsecase) ExecuteFlow(ctx context.Context, query *CustomerQuery) {
      // Add timeout
      ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
      defer cancel()

      result, err := flow.Run(ctx, query)
      // ...
  }
  ```

---

## 📊 Summary Table

| # | Gap | Severity | Status | Solution |
|---|-----|----------|--------|----------|
| 1 | Genkit-Gin Integration | 🔴 Critical | Open | Wrapper function |
| 2 | Request/Response Wrapping | 🔴 Critical | Open | Middleware or auto |
| 3 | Instance Management | 🔴 Critical | Open | Dependency injection |
| 4 | Config Schema | 🟡 High | Open | Update Config struct |
| 5 | Tool Call Extraction | 🟡 High | Open | Regex parser |
| 6 | Domain Interfaces | 🟡 High | Open | Follow existing pattern |
| 7 | Flow Definition Location | 🟡 High | Open | Init in GenkitService |
| 8 | Handler Wrapper | 🟡 High | Open | Follow existing pattern |
| 9 | Middleware Compatibility | 🟠 Medium | Open | Integration tests |
| 10 | DataFrame Column Mapping | 🟠 Medium | Open | Configurable mapping |
| 11 | Error Format Consistency | 🟠 Medium | Open | Error wrapper |
| 12 | Health Check | 🟠 Medium | Open | AI connection test |
| 13 | Testing Strategy | 🟢 Low | Open | Test specs defined |
| 14 | Docker Dependencies | 🟢 Low | Open | Add curl to image |
| 15 | Prompt Management | 🟢 Low | Open | Constants file |
| 16 | Server Plugin Usage | 🟢 Low | Open | Verify not needed |
| 17 | Context Propagation | 🟠 Medium | Open | Add timeouts |

---

## ✅ Recommended Updated Plan

### Phase 1: Foundation (Address Gaps 1-7)
1. Create Genkit wrapper for Gin integration
2. Update Config struct with validation
3. Implement GenkitService with dependency injection
4. Define domain interfaces following existing patterns
5. Create tool call extraction utility
6. Implement flow initialization logic

### Phase 2: Integration (Address Gaps 8-12)
7. Create HTTP handlers following existing patterns
8. Add middleware compatibility tests
9. Implement health check with AI connection test
10. Create Python bridge with column mapping
11. Ensure error format consistency

### Phase 3: Polish (Address Gaps 13-17)
12. Write comprehensive unit/integration tests
13. Update Dockerfile with health check support
14. Extract prompt templates to constants
15. Add context timeouts
16. Verify server plugin is not needed

---

## 🎯 Next Steps

1. **Review this analysis** - Confirm gaps are valid
2. **Prioritize gaps** - Which must be solved before starting?
3. **Create proof-of-concept** - Test Genkit-Gin integration first
4. **Update implementation plan** - Incorporate gap solutions
5. **Begin implementation** - Start with Phase 1

---

## Questions for Validation

1. **Should we use `genkit.Handler()` directly or call flows programmatically?**
   - Research shows both approaches, need to decide

2. **Where should prompt templates live?**
   - Hardcoded constants
   - Configuration files
   - Database (overkill?)

3. **Do we need the Genkit server plugin at all?**
   - Verify we can use just the core + AI plugin

4. **How to handle streaming responses?**
   - Research doesn't show streaming
   - Giskard expects batch responses
   - Probably not needed for MVP

5. **Authentication for Genkit endpoints?**
   - Should they require JWT like other endpoints?
   - Or separate API key auth?
   - Or public (for testing)?

---

*Generated: 2025-11-08*
*Author: Claude Code Analysis*
