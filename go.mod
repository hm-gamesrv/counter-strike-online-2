module github.com/KouKouChan/CSO2-Server

go 1.27.0

require (
	// YAML for the configuration file. It is the only format the project needs
	// a library for; everything else is standard library.
	go.yaml.in/yaml/v3 v3.0.4

	// Character-set conversion for non-UTF-8 clients (GBK / Big5 / EUC-KR /
	// Shift_JIS). This replaces the old cgo-based github.com/djimenez/iconv-go,
	// which made cross-compilation and minimal container images painful.
	golang.org/x/text v0.37.0
)
