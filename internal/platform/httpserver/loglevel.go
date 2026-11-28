package httpserver

import (
	"log/slog"
	"net/http"
)

// LogLevel serves levelVar. GET answers {"level":"INFO"}. PUT with {"level":"debug"} sets the level, logs the change
// at WARN with log, and answers the new level; slog's names are accepted in any case, with an offset such as
// "info+2". A level slog does not know answers 422 invalid_level, and a body DecodeJSON refuses answers its
// problem; neither changes the level.
func LogLevel(levelVar *slog.LevelVar, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			var body struct {
				Level string `json:"level"`
			}
			if problem := DecodeJSON(r, &body); problem != nil {
				WriteProblem(w, r, *problem)
				return
			}
			var level slog.Level
			if err := level.UnmarshalText([]byte(body.Level)); err != nil {
				WriteProblem(w, r, Problem{
					Status: http.StatusUnprocessableEntity,
					Code:   "invalid_level",
					Detail: "level must be debug, info, warn or error",
				})
				return
			}

			from := levelVar.Level()
			levelVar.Set(level)
			log.LogAttrs(r.Context(), slog.LevelWarn, "log level changed",
				slog.String("from", from.String()), slog.String("to", level.String()))
		}

		WriteJSON(w, http.StatusOK, map[string]string{"level": levelVar.Level().String()})
	}
}
