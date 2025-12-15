package gorouter

import (
	"regexp"
	"strings"
)

// ValidationError 验证错误
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors 多个验证错误
type ValidationErrors []ValidationError

// Error 实现error接口
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return ""
	}

	var messages []string
	for _, err := range ve {
		messages = append(messages, err.Field+": "+err.Message)
	}
	return strings.Join(messages, "; ")
}

// Validator 验证器
type Validator struct {
	errors ValidationErrors
}

// NewValidator 创建新的验证器
func NewValidator() *Validator {
	return &Validator{
		errors: make(ValidationErrors, 0),
	}
}

// AddError 添加验证错误
func (v *Validator) AddError(field, message string) {
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// HasErrors 是否有错误
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Errors 获取所有错误
func (v *Validator) Errors() ValidationErrors {
	return v.errors
}

// ValidateRequired 验证必填字段
func (v *Validator) ValidateRequired(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.AddError(field, "required: "+field)
	}
}

// ValidateEmail 验证邮箱格式
func (v *Validator) ValidateEmail(field, email string) {
	if email == "" {
		return
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		v.AddError(field, "invalid email format")
	}
}

// ValidateLength 验证字符串长度
func (v *Validator) ValidateLength(field, value string, min, max int) {
	length := len(strings.TrimSpace(value))
	if length < min {
		v.AddError(field, "length must be greater than "+string(rune(min)))
	}
	if length > max {
		v.AddError(field, "length must be less than "+string(rune(max)))
	}
}

// ValidateRange 验证数值范围
func (v *Validator) ValidateRange(field string, value, min, max int) {
	if value < min {
		v.AddError(field, "value must be greater than "+string(rune(min)))
	}
	if value > max {
		v.AddError(field, "value must be less than "+string(rune(max)))
	}
}

// ValidateRegex 验证正则表达式
func (v *Validator) ValidateRegex(field, value, pattern, message string) {
	if value == "" {
		return
	}

	regex := regexp.MustCompile(pattern)
	if !regex.MatchString(value) {
		v.AddError(field, message)
	}
}

// ValidatePassword 验证密码强度
func (v *Validator) ValidatePassword(field, password string) {
	if password == "" {
		return
	}

	// 至少8位，包含字母和数字
	if len(password) < 8 {
		v.AddError(field, "password length must be greater than 8")
	}

	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)

	if !hasLetter {
		v.AddError(field, "password must contain letters")
	}
	if !hasDigit {
		v.AddError(field, "password must contain digits")
	}
}
