package api

import "net/http"

func withSecurityHeaders(next http.Handler) http.Handler {
	next = fallbackHandler(next)
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		header := writer.Header()
		header.Set(
			"Content-Security-Policy",
			"default-src 'self'; base-uri 'none'; frame-ancestors 'none'; "+
				"form-action 'self'; img-src 'self' data: blob:; media-src 'self' blob:; "+
				"object-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'",
		)
		header.Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(writer, request)
	})
}

func withHSTS(next http.Handler) http.Handler {
	next = fallbackHandler(next)
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if requestTransportFrom(request).secure {
			writer.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(writer, request)
	})
}
