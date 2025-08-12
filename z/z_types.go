package z

// Status 统一状态码类型 / Unified status code type
type Status int

// 通用成功/失败 / General success/failure codes
const (
	StatusOK       Status = 10000 // 成功 / Success
	StatusFailed   Status = 10001 // 失败（通用）/ General failure
	StatusUnknown  Status = 10002 // 未知错误 / Unknown error
	StatusPending  Status = 10003 // 处理中/异步中 / Processing/Async in progress
	StatusAccepted Status = 10004 // 已接受 / Accepted
)

// 客户端错误（4xxxx）/ Client errors (4xxxx)
const (
	StatusBadRequest            Status = 40000 // 请求参数错误 / Bad request parameters
	StatusUnauthorized          Status = 40001 // 未认证/未登录 / Unauthorized/Not logged in
	StatusPaymentRequired       Status = 40002 // 需要付费 / Payment required
	StatusForbidden             Status = 40003 // 无权限 / Forbidden
	StatusNotFound              Status = 40004 // 资源不存在 / Resource not found
	StatusMethodNotAllowed      Status = 40005 // 方法不允许 / Method not allowed
	StatusNotAcceptable         Status = 40006 // 不可接受的请求 / Not acceptable
	StatusProxyAuthRequired     Status = 40007 // 代理认证需要 / Proxy authentication required
	StatusRequestTimeout        Status = 40008 // 请求超时 / Request timeout
	StatusConflict              Status = 40009 // 资源冲突（如重复）/ Resource conflict
	StatusGone                  Status = 40010 // 资源已删除 / Resource gone
	StatusLengthRequired        Status = 40011 // 需要内容长度 / Length required
	StatusPreconditionFailed    Status = 40012 // 前置条件失败 / Precondition failed
	StatusPayloadTooLarge       Status = 40013 // 请求体过大 / Payload too large
	StatusURITooLong            Status = 40014 // URI 过长 / URI too long
	StatusUnsupportedMedia      Status = 40015 // 不支持的媒体类型 / Unsupported media type
	StatusRangeNotSatisfiable   Status = 40016 // 范围无法满足 / Range not satisfiable
	StatusExpectationFailed     Status = 40017 // 期望失败 / Expectation failed
	StatusUnprocessableEntity   Status = 40022 // 语义错误/参数校验不通过 / Unprocessable entity
	StatusLocked                Status = 40023 // 资源被锁定 / Resource locked
	StatusFailedDependency      Status = 40024 // 依赖失败 / Failed dependency
	StatusUpgradeRequired       Status = 40026 // 需要升级 / Upgrade required
	StatusPreconditionRequired  Status = 40028 // 需要前置条件 / Precondition required
	StatusTooManyRequests       Status = 40029 // 频率限制 / Rate limit exceeded
	StatusRequestHeaderTooLarge Status = 40031 // 请求头过大 / Request header too large
	StatusUnavailableForLegal   Status = 40051 // 因法律原因不可用 / Unavailable for legal reasons
)

// 服务端错误（5xxxx）/ Server errors (5xxxx)
const (
	StatusInternalError           Status = 50000 // 服务器内部错误 / Internal server error
	StatusNotImplemented          Status = 50001 // 功能未实现 / Not implemented
	StatusBadGateway              Status = 50002 // 网关错误 / Bad gateway
	StatusServiceUnavailable      Status = 50003 // 服务不可用 / Service unavailable
	StatusGatewayTimeout          Status = 50004 // 网关超时 / Gateway timeout
	StatusHTTPVersionNotSupported Status = 50005 // HTTP版本不支持 / HTTP version not supported
	StatusVariantAlsoNegotiates   Status = 50006 // 变体协商 / Variant also negotiates
	StatusInsufficientStorage     Status = 50007 // 存储空间不足 / Insufficient storage
	StatusLoopDetected            Status = 50008 // 检测到循环 / Loop detected
	StatusNotExtended             Status = 50010 // 未扩展 / Not extended
	StatusNetworkAuthRequired     Status = 50011 // 需要网络认证 / Network authentication required
	StatusDependencyFailed        Status = 50020 // 下游依赖失败 / Downstream dependency failed
	StatusCircuitBreakerOpen      Status = 50021 // 熔断器开启 / Circuit breaker open
	StatusRateLimitExceeded       Status = 50022 // 服务端限流 / Server rate limit exceeded
)

// 认证与授权（2xxxx）/ Authentication and authorization (2xxxx)
const (
	StatusAuthTokenInvalid    Status = 20001 // 访问令牌无效 / Invalid access token
	StatusAuthTokenExpired    Status = 20002 // 访问令牌过期 / Access token expired
	StatusPermissionDenied    Status = 20003 // 拒绝访问 / Permission denied
	StatusLoginRequired       Status = 20004 // 需要登录 / Login required
	StatusRefreshTokenInvalid Status = 20005 // 刷新令牌无效 / Invalid refresh token
	StatusRefreshTokenExpired Status = 20006 // 刷新令牌过期 / Refresh token expired
	StatusAccountLocked       Status = 20007 // 账户被锁定 / Account locked
	StatusAccountDisabled     Status = 20008 // 账户被禁用 / Account disabled
	StatusPasswordExpired     Status = 20009 // 密码过期 / Password expired
	StatusTwoFactorRequired   Status = 20010 // 需要双因子认证 / Two-factor authentication required
	StatusSessionExpired      Status = 20011 // 会话过期 / Session expired
	StatusInvalidCredentials  Status = 20012 // 凭据无效 / Invalid credentials
)

// 数据与资源相关（3xxxx）/ Data and resource related (3xxxx)
const (
	StatusResourceExists   Status = 30001 // 资源已存在 / Resource already exists
	StatusResourceNotFound Status = 30004 // 资源未找到（区别于 40004，语义化业务层）/ Resource not found (business layer)
	StatusDataValidation   Status = 30022 // 数据校验失败（更贴近业务）/ Data validation failed
	StatusDataConflict     Status = 30009 // 数据冲突（业务语义）/ Data conflict (business semantics)
	StatusDataCorrupted    Status = 30010 // 数据损坏 / Data corrupted
	StatusDataInconsistent Status = 30011 // 数据不一致 / Data inconsistent
	StatusQuotaExceeded    Status = 30012 // 配额超限 / Quota exceeded
	StatusResourceLocked   Status = 30013 // 资源被锁定 / Resource locked
	StatusVersionConflict  Status = 30014 // 版本冲突 / Version conflict
	StatusDuplicateEntry   Status = 30015 // 重复条目 / Duplicate entry
	StatusInvalidFormat    Status = 30016 // 格式无效 / Invalid format
	StatusMissingRequired  Status = 30017 // 缺少必需字段 / Missing required field
	StatusOutOfRange       Status = 30018 // 超出范围 / Out of range
	StatusInvalidState     Status = 30019 // 状态无效 / Invalid state
)

// 业务逻辑相关（7xxxx）/ Business logic related (7xxxx)
const (
	StatusBusinessRuleViolation Status = 70001 // 违反业务规则 / Business rule violation
	StatusWorkflowError         Status = 70002 // 工作流错误 / Workflow error
	StatusOperationNotAllowed   Status = 70003 // 操作不被允许 / Operation not allowed
	StatusInvalidOperation      Status = 70004 // 无效操作 / Invalid operation
	StatusOrderNotFound         Status = 70005 // 订单未找到 / Order not found
	StatusOrderCancelled        Status = 70006 // 订单已取消 / Order cancelled
	StatusPaymentFailed         Status = 70007 // 支付失败 / Payment failed
	StatusInventoryInsufficient Status = 70008 // 库存不足 / Insufficient inventory
	StatusUserNotFound          Status = 70009 // 用户未找到 / User not found
	StatusUserAlreadyExists     Status = 70010 // 用户已存在 / User already exists
)

// 依赖/系统类（6xxxx）/ Dependencies/System related (6xxxx)
const (
	StatusDBError            Status = 60001 // 数据库错误 / Database error
	StatusCacheError         Status = 60002 // 缓存错误 / Cache error
	StatusConfigError        Status = 60003 // 配置错误 / Configuration error
	StatusIOError            Status = 60004 // IO 读写错误 / IO read/write error
	StatusThirdPartyError    Status = 60005 // 第三方依赖/外部服务错误 / Third-party/External service error
	StatusNetworkError       Status = 60006 // 网络错误 / Network error
	StatusTimeoutError       Status = 60007 // 超时错误 / Timeout error
	StatusMemoryError        Status = 60008 // 内存错误 / Memory error
	StatusDiskError          Status = 60009 // 磁盘错误 / Disk error
	StatusQueueError         Status = 60010 // 队列错误 / Queue error
	StatusLockError          Status = 60011 // 锁错误 / Lock error
	StatusSerializationError Status = 60012 // 序列化错误 / Serialization error
)

// IsSuccess 判断是否为成功状态码 / Check if status code indicates success
func IsSuccess(s Status) bool { return s >= 10000 && s < 20000 }

// IsClientError 判断是否为客户端错误码 / Check if status code is client error
func IsClientError(s Status) bool { return s >= 40000 && s < 50000 }

// IsServerError 判断是否为服务端错误码 / Check if status code is server error
func IsServerError(s Status) bool { return s >= 50000 && s < 60000 }

// IsAuthStatus 判断是否为认证授权相关状态 / Check if status code is authentication/authorization related
func IsAuthStatus(s Status) bool { return s >= 20000 && s < 30000 }

// IsDataStatus 判断是否为数据/资源相关状态 / Check if status code is data/resource related
func IsDataStatus(s Status) bool { return s >= 30000 && s < 40000 }

// IsDependencyStatus 判断是否为依赖/系统相关状态 / Check if status code is dependency/system related
func IsDependencyStatus(s Status) bool { return s >= 60000 && s < 70000 }

// IsBusinessStatus 判断是否为业务逻辑相关状态 / Check if status code is business logic related
func IsBusinessStatus(s Status) bool { return s >= 70000 && s < 80000 }

// IsError 判断是否为错误状态码 / Check if status code indicates error
func IsError(s Status) bool {
	return IsClientError(s) || IsServerError(s) || IsAuthStatus(s) ||
		IsDataStatus(s) || IsDependencyStatus(s) || IsBusinessStatus(s) ||
		s == StatusFailed || s == StatusUnknown
}

// String 返回状态码的字符串表示 / Return string representation of status code
func (s Status) String() string {
	switch s {
	// 通用状态码
	case StatusOK:
		return "OK"
	case StatusFailed:
		return "FAILED"
	case StatusUnknown:
		return "UNKNOWN"
	case StatusPending:
		return "PENDING"
	case StatusAccepted:
		return "ACCEPTED"

	// 客户端错误
	case StatusBadRequest:
		return "BAD_REQUEST"
	case StatusUnauthorized:
		return "UNAUTHORIZED"
	case StatusForbidden:
		return "FORBIDDEN"
	case StatusNotFound:
		return "NOT_FOUND"
	case StatusConflict:
		return "CONFLICT"
	case StatusTooManyRequests:
		return "TOO_MANY_REQUESTS"
	case StatusUnprocessableEntity:
		return "UNPROCESSABLE_ENTITY"

	// 服务端错误
	case StatusInternalError:
		return "INTERNAL_ERROR"
	case StatusNotImplemented:
		return "NOT_IMPLEMENTED"
	case StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	case StatusGatewayTimeout:
		return "GATEWAY_TIMEOUT"

	// 认证授权
	case StatusAuthTokenInvalid:
		return "AUTH_TOKEN_INVALID"
	case StatusAuthTokenExpired:
		return "AUTH_TOKEN_EXPIRED"
	case StatusPermissionDenied:
		return "PERMISSION_DENIED"
	case StatusLoginRequired:
		return "LOGIN_REQUIRED"

	// 数据资源
	case StatusResourceExists:
		return "RESOURCE_EXISTS"
	case StatusResourceNotFound:
		return "RESOURCE_NOT_FOUND"
	case StatusDataValidation:
		return "DATA_VALIDATION"
	case StatusDataConflict:
		return "DATA_CONFLICT"

	// 依赖系统
	case StatusDBError:
		return "DB_ERROR"
	case StatusCacheError:
		return "CACHE_ERROR"
	case StatusThirdPartyError:
		return "THIRD_PARTY_ERROR"

	// 业务逻辑
	case StatusBusinessRuleViolation:
		return "BUSINESS_RULE_VIOLATION"
	case StatusWorkflowError:
		return "WORKFLOW_ERROR"
	case StatusOperationNotAllowed:
		return "OPERATION_NOT_ALLOWED"

	default:
		return "UNKNOWN_STATUS"
	}
}
