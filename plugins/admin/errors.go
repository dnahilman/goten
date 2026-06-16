package adminplugin

// Error codes returned in the "code" field of admin error responses.
const (
	codeUnauthorized   = "UNAUTHORIZED"
	codeForbidden      = "FORBIDDEN"
	codeInvalidBody    = "INVALID_BODY"
	codeInternal       = "INTERNAL"
	codeUserNotFound   = "USER_NOT_FOUND"
	codeUserExists     = "USER_ALREADY_EXISTS"
	codeInvalidRole    = "INVALID_ROLE"
	codeInvalidInput   = "INVALID_INPUT"
	codeCannotBanSelf  = "CANNOT_BAN_SELF"
	codeCannotDelSelf  = "CANNOT_DELETE_SELF"
	codeNotImpersonate = "NOT_IMPERSONATING"
)
