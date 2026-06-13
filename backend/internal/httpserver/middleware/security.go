package middleware

import "net/http"

const (
	contentTypeOptionsHeader = "nosniff"
	referrerPolicyHeader     = "no-referrer"
	frameOptionsHeader       = "DENY"
	permissionsPolicyHeader  = "accelerometer=(), autoplay=(), camera=(), display-capture=(), encrypted-media=(), fullscreen=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), midi=(), payment=(), picture-in-picture=(), publickey-credentials-get=(), usb=(), xr-spatial-tracking=()"

	apiContentSecurityPolicy  = "default-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"
	docsContentSecurityPolicy = "default-src 'self'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'; script-src 'self' https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'"
)

func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := w.Header()
			header.Set("X-Content-Type-Options", contentTypeOptionsHeader)
			header.Set("Referrer-Policy", referrerPolicyHeader)
			header.Set("X-Frame-Options", frameOptionsHeader)
			header.Set("Permissions-Policy", permissionsPolicyHeader)
			header.Set("Content-Security-Policy", contentSecurityPolicy(r.URL.Path))
			next.ServeHTTP(w, r)
		})
	}
}

func NoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("Cache-Control", "no-store")
		header.Set("Pragma", "no-cache")
		header.Set("Expires", "0")
		next.ServeHTTP(w, r)
	})
}

func contentSecurityPolicy(path string) string {
	if path == "/docs" {
		return docsContentSecurityPolicy
	}
	return apiContentSecurityPolicy
}
