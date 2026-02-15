package validator

// AllowedOperatorRoles — допустимые роли оператора для directory.
var AllowedOperatorRoles = map[string]bool{
	"operator":   true,
	"supervisor": true,
	"admin":      true,
}

// IsValidRole возвращает true, если role входит в список допустимых.
func IsValidRole(role string) bool {
	return AllowedOperatorRoles[role]
}
