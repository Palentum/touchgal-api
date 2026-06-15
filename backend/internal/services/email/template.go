package email

import (
	"fmt"
	"github.com/touchgal/developer/backend/internal/model"
	"html"
)

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

func VerificationHTMLBody(purpose, code string, ttlMinutes int) string {
	purposeLabel := "登录"
	purposeCopy := "登录 TouchGal API 开发者门户"
	if purpose == "register" {
		purposeLabel = "注册"
		purposeCopy = "完成 TouchGal API 开发者门户注册"
	}

	escapedCode := html.EscapeString(code)
	return fmt.Sprintf(`<!doctype html>
<html lang="zh-Hans">
<head>
  <meta http-equiv="Content-Type" content="text/html; charset=utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
</head>
<body style="margin:0;padding:0;background:#faf9f5;color:#141413;font-family:Inter,-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;">
  <div style="display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;">验证码将在 %d 分钟后过期。若不是本人操作，请忽略此邮件。</div>
  <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="width:100%%;background:#faf9f5;">
    <tr>
      <td align="center" style="padding:40px 16px;">
        <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="width:100%%;max-width:600px;">
          <tr>
            <td style="padding:0 0 16px 0;color:#141413;font-size:14px;font-weight:500;line-height:1.4;">
              <span style="display:inline-block;width:28px;height:28px;margin-right:10px;border-radius:9999px;background:#181715;color:#faf9f5;text-align:center;line-height:28px;">✣</span>
              TouchGal API
            </td>
          </tr>
          <tr>
            <td style="background:#efe9de;border:1px solid #e6dfd8;border-radius:16px;padding:32px;">
              <div style="display:inline-block;margin:0 0 18px 0;padding:5px 12px;border-radius:9999px;background:#cc785c;color:#ffffff;font-size:12px;font-weight:500;letter-spacing:1.5px;line-height:1.4;text-transform:uppercase;">%s CODE</div>
              <h1 style="margin:0;color:#141413;font-family:'Cormorant Garamond','Tiempos Headline',Georgia,'Times New Roman',serif;font-size:34px;font-weight:500;letter-spacing:-0.5px;line-height:1.15;">请确认这次%s请求</h1>
              <p style="margin:16px 0 0 0;color:#3d3d3a;font-size:16px;font-weight:400;line-height:1.55;">输入下方验证码以%s。验证码只用于本次操作。</p>

              <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="margin:28px 0;width:100%%;background:#181715;border-radius:12px;">
                <tr>
                  <td align="center" style="padding:28px 16px;">
                    <div style="margin:0 0 8px 0;color:#a09d96;font-size:12px;font-weight:500;letter-spacing:1.5px;line-height:1.4;text-transform:uppercase;">Verification Code</div>
                    <div style="color:#faf9f5;font-family:'JetBrains Mono','SFMono-Regular',Consolas,'Liberation Mono',monospace;font-size:38px;font-weight:600;letter-spacing:8px;line-height:1.2;">%s</div>
                  </td>
                </tr>
              </table>

              <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="width:100%%;background:#faf9f5;border:1px solid #e6dfd8;border-radius:12px;">
                <tr>
                  <td style="padding:18px 20px;color:#3d3d3a;font-size:14px;line-height:1.55;">
                    <strong style="color:#252523;font-weight:500;">有效期：</strong>%d 分钟<br>
                    <strong style="color:#252523;font-weight:500;">安全提示：</strong>TouchGal API 不会通过邮件索要密码、API token 或会话信息。若不是本人操作，请直接忽略此邮件。
                  </td>
                </tr>
              </table>
            </td>
          </tr>
          <tr>
            <td style="padding:18px 4px 0 4px;color:#8e8b82;font-size:13px;font-weight:400;line-height:1.5;">
              此邮件由系统自动发送，请勿回复。
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, html.EscapeString(VerificationSubject(purpose)), ttlMinutes, purposeLabel, purposeLabel, purposeCopy, escapedCode, ttlMinutes)
}

func ApplicationSubmittedSubject() string {
	return "TouchGal API 新应用申请待审核"
}

func ApplicationSubmittedBody(app model.Application, reviewURL string) string {
	projectName := app.ProjectName
	if projectName == "" {
		projectName = "未填写"
	}
	if reviewURL == "" {
		reviewURL = "/admin/applications"
	}
	return fmt.Sprintf(`您好：

收到新的 TouchGal API 应用申请，请及时审核。

申请人：%s
项目名称：%s
项目地址：%s
预计日请求量：%d
使用场景：%s
审核入口：%s
`, app.ApplicantName, projectName, app.ProjectURL, app.ExpectedDailyRequests, app.UsageScenario, reviewURL)
}

func ApplicationSubmittedHTMLBody(app model.Application, reviewURL string) string {
	projectName := app.ProjectName
	if projectName == "" {
		projectName = "未填写"
	}
	if reviewURL == "" {
		reviewURL = "/admin/applications"
	}

	escapedApplicantName := html.EscapeString(app.ApplicantName)
	escapedProjectName := html.EscapeString(projectName)
	escapedProjectURL := html.EscapeString(app.ProjectURL)
	escapedUsageScenario := html.EscapeString(app.UsageScenario)
	escapedReviewURL := html.EscapeString(reviewURL)

	return fmt.Sprintf(`<!doctype html>
<html lang="zh-Hans">
<head>
  <meta http-equiv="Content-Type" content="text/html; charset=utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s</title>
</head>
<body style="margin:0;padding:0;background:#faf9f5;color:#181715;font-family:Inter,-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'Helvetica Neue',Arial,sans-serif;">
  <div style="display:none;max-height:0;overflow:hidden;opacity:0;color:transparent;">收到新的 TouchGal API 应用申请，请及时审核。</div>
  <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="width:100%%;background:#faf9f5;">
    <tr>
      <td align="center" style="padding:40px 16px;">
        <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="width:100%%;max-width:640px;">
          <tr>
            <td style="padding:0 0 16px 0;color:#181715;font-size:14px;font-weight:500;line-height:1.4;">
              <span style="display:inline-block;width:28px;height:28px;margin-right:10px;border-radius:9999px;background:#181715;color:#faf9f5;text-align:center;line-height:28px;">✣</span>
              TouchGal API
            </td>
          </tr>
          <tr>
            <td style="background:#efe9de;border:1px solid #e6dfd8;border-radius:16px;padding:32px;">
              <div style="display:inline-block;margin:0 0 18px 0;padding:5px 12px;border-radius:9999px;background:#cc785c;color:#ffffff;font-size:12px;font-weight:500;letter-spacing:1.5px;line-height:1.4;text-transform:uppercase;">APPLICATION REVIEW</div>
              <h1 style="margin:0;color:#181715;font-family:'Cormorant Garamond','Tiempos Headline',Georgia,'Times New Roman',serif;font-size:34px;font-weight:500;letter-spacing:-0.5px;line-height:1.15;">新应用申请待审核</h1>
              <p style="margin:16px 0 0 0;color:#3d3d3a;font-size:16px;font-weight:400;line-height:1.55;">开发者已提交新的 API 使用申请，请在管理员后台完成审核。</p>

              <table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0" style="margin:28px 0;width:100%%;background:#faf9f5;border:1px solid #e6dfd8;border-radius:12px;">
                <tr>
                  <td style="padding:18px 20px;color:#3d3d3a;font-size:14px;line-height:1.65;">
                    <strong style="color:#252523;font-weight:500;">申请人：</strong>%s<br>
                    <strong style="color:#252523;font-weight:500;">项目名称：</strong>%s<br>
                    <strong style="color:#252523;font-weight:500;">项目地址：</strong>%s<br>
                    <strong style="color:#252523;font-weight:500;">预计日请求量：</strong>%d<br>
                    <strong style="color:#252523;font-weight:500;">使用场景：</strong><span style="white-space:pre-wrap;">%s</span>
                  </td>
                </tr>
              </table>

              <a href="%s" style="display:inline-block;padding:12px 18px;border-radius:9999px;background:#181715;color:#faf9f5;font-size:14px;font-weight:500;line-height:1.4;text-decoration:none;">打开审核入口</a>
              <p style="margin:14px 0 0 0;color:#6f6a62;font-size:13px;line-height:1.5;">审核入口：%s</p>
            </td>
          </tr>
          <tr>
            <td style="padding:18px 4px 0 4px;color:#8e8b82;font-size:13px;font-weight:400;line-height:1.5;">
              此邮件由系统自动发送，请勿回复。邮件不包含 API token、验证码或会话信息。
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, html.EscapeString(ApplicationSubmittedSubject()), escapedApplicantName, escapedProjectName, escapedProjectURL, app.ExpectedDailyRequests, escapedUsageScenario, escapedReviewURL, escapedReviewURL)
}
