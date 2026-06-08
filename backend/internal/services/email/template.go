package email

import "fmt"

func VerificationSubject(purpose string) string {
	if purpose == "register" {
		return "TouchGal API 注册验证码"
	}
	return "TouchGal API 登录验证码"
}

func VerificationBody(code string, ttlMinutes int) string {
	return fmt.Sprintf(`您好：

您的 TouchGal API 验证码是：%s

验证码将在 %d 分钟后过期。若不是本人操作，请忽略此邮件。
`, code, ttlMinutes)
}
