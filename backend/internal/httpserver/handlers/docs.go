package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	openapidoc "github.com/touchgal/developer/backend/internal/openapi"
)

const docsCacheControl = "public, max-age=300, must-revalidate"

var (
	openAPIDocument = []byte(openapidoc.Document())
	openAPIETag     = contentETag(openAPIDocument)
	docsHTML        = []byte(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>TouchGal API Docs</title>
<script type="module" src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.59.2"></script>
</head>
<body>
<scalar-api-reference configuration='{"url":"/openapi.yaml"}'></scalar-api-reference>
</body>
</html>`)
	docsHTMLETag = contentETag(docsHTML)
)

type DocsHandler struct{}

func (DocsHandler) OpenAPI(w http.ResponseWriter, r *http.Request) {
	writeCachedDocument(w, r, "application/yaml; charset=utf-8", openAPIDocument, openAPIETag)
}

func (DocsHandler) Swagger(w http.ResponseWriter, r *http.Request) {
	writeCachedDocument(w, r, "text/html; charset=utf-8", docsHTML, docsHTMLETag)
}

func writeCachedDocument(w http.ResponseWriter, r *http.Request, contentType string, data []byte, etag string) {
	header := w.Header()
	header.Set("Content-Type", contentType)
	header.Set("Cache-Control", docsCacheControl)
	header.Set("ETag", etag)

	if ifNoneMatch(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func ifNoneMatch(header, etag string) bool {
	if header == "" {
		return false
	}
	for candidate := range strings.SplitSeq(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == etag || strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}

func contentETag(data []byte) string {
	sum := sha256.Sum256(data)
	var out [73]byte
	out[0] = '"'
	copy(out[1:], "sha256-")
	hex.Encode(out[8:72], sum[:])
	out[72] = '"'
	return string(out[:])
}
