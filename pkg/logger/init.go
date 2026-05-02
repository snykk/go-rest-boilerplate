package logger

// Auto-initialize a sensible default logger so package-level
// calls (logger.Info, logger.Error, ...) work before main() has a
// chance to call InitDefault. The bootstrap config is intentionally
// minimal: console output, info level, no JSON, no app name. main()
// is expected to override it with config-driven settings.
//
// If InitDefault is later called the global is replaced atomically;
// in-flight log calls just see whichever instance is current.
func init() {
	_ = InitDefault(Config{
		Level:         LevelInfo,
		EnableConsole: true,
	}, InstanceZap)
}
