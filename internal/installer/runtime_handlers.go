package installer

import "net/http"

func RegisterRuntimeHandlers(mux *http.ServeMux) {
	// Runtime handlers are registered by NewServerWithConfig.
	_ = mux
}
