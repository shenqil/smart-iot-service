package constant

const RequestIDKey = "X-Request-ID"

const (
	Success             = 200
	BadRequest          = 400
	Unauthorized        = 401
	Forbidden           = 403
	NotFound            = 404
	Conflict            = 409
	InternalServerError = 500
)

const (
	MsgSuccess           = "请求成功"
	MsgInvalidInput      = "请求参数不合法"
	MsgUnauthorized      = "未授权访问"
	MsgForbidden         = "没有权限"
	MsgNotFound          = "资源不存在"
	MsgConflict          = "资源冲突"
	MsgInternalServerErr = "服务器内部错误"
)

const (
	RelayOperationOpen         = byte(1)
	RelayOperationOpenSuccess  = byte(2)
	RelayOperationClose        = byte(3)
	RelayOperationCloseSuccess = byte(4)
)
