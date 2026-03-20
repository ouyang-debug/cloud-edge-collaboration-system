module agent

go 1.24.0

toolchain go1.24.4

require (
	github.com/eclipse/paho.mqtt.golang v1.4.3
	github.com/go-sql-driver/mysql v1.9.3
	github.com/lib/pq v1.10.9
	github.com/pkg/sftp v1.13.10
	github.com/robfig/cron/v3 v3.0.1
	github.com/tjfoc/gmsm v1.4.1
	golang.org/x/crypto v0.47.0
	google.golang.org/grpc v1.70.0
	google.golang.org/protobuf v1.36.11
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.1
	modernc.org/sqlite v1.44.3
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-sqlite3 v1.14.33 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	go.opentelemetry.io/otel v1.34.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.34.0 // indirect
	golang.org/x/exp v0.0.0-20251023183803-a4bb9ffd2546 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
	modernc.org/libc v1.67.6 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
)

replace example.com/agent_base_plus => ..

exclude google.golang.org/genproto v0.0.0-20190819201941-24fa4b261c55

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20250115164207-1a7da9e5054f
