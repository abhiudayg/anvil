// Package curriculum defines what Anvil teaches and in what order.
//
// The plan has three parts:
//   - Levels say when a stage is taken (readiness and difficulty).
//   - Projects are build tracks. Each has stages, and each stage sits at one level.
//   - Concepts are what is taught. Each concept is introduced in exactly one
//     stage, and any prerequisite it names must come earlier in the path.
//
// Path() flattens all of that into the single order a learner follows.
package curriculum

// Level groups stages by readiness. A level assumes every level below it.
type Level struct {
	Number int
	Title  string
	Goal   string
}

// Levels in the order they are taken.
var Levels = []Level{
	{1, "Foundations", "Write, run and test small Go programs. Understand values, types and control flow."},
	{2, "Types, interfaces and design", "Model data, design APIs around small interfaces, and use generics."},
	{3, "HTTP services", "Serve and route HTTP with the standard library and middleware."},
	{4, "Concurrency", "Share state safely across goroutines, control cancellation and run worker pools."},
	{5, "Storage", "Store data in Postgres, use transactions, and keep state correct under concurrency."},
	{6, "Quality and performance", "Measure, profile, test and harden code. Understand the runtime."},
	{7, "Networking and RPC", "Build gRPC and Connect services, streams and realtime updates."},
	{8, "Advanced Go and shipping", "Use reflection, unsafe and cgo carefully. Secure, version and release binaries."},
	{9, "Kubernetes API and clients", "Read the Kubernetes API model, talk to it with client-go, and keep local caches in sync."},
	{10, "Kubernetes workloads and networking", "Run jobs and services on a cluster: pods, scheduling, Services and configuration."},
	{11, "Kubernetes operators", "Build controllers and CRDs with controller-runtime, and test them, including admission webhooks."},
	{12, "Kubernetes in production", "Deploy and upgrade operators safely, scale them and recover from failures."},
}

// Concept is one thing a learner must understand.
type Concept struct {
	ID    string   // stable slug, unique across the curriculum
	Name  string   // what the learner sees
	Needs []string // IDs that must be taught earlier in the path
}

// Stage is one step of a project. Its concepts are taught before the build
// work in that stage, and the stage's injected bug targets one of them.
type Stage struct {
	Title    string
	Level    int // a Level.Number
	Concepts []Concept
}

// Project is one "build your own X" track made of ordered stages.
type Project struct {
	Slug    string
	Name    string
	Summary string
	Stages  []Stage
}

// PathStep is one stage in the single learning order.
type PathStep struct {
	Level       Level
	Project     Project
	StageNumber int // 1-based position within the project
	Stage       Stage
}

func c(id, name string, needs ...string) Concept {
	return Concept{ID: id, Name: name, Needs: needs}
}

// Projects lists the build tracks. Their order is also the tie-break inside a level.
var Projects = []Project{
	{
		Slug:    "redis",
		Name:    "Build your Redis",
		Summary: "A RESP-speaking, concurrent, persistent in-memory key-value server.",
		Stages: []Stage{
			{Title: "RESP reader and TCP server", Level: 1, Concepts: []Concept{
				c("toolchain", "Installing Go and checking the version"),
				c("go-mod-init", "Creating a module with go mod init and reading go.mod"),
				c("go-env", "Reading the environment with go env"),
				c("package-main", "package main and func main"),
				c("run-build-install", "go run versus go build versus go install"),
				c("fmt-vet", "gofmt and go vet as everyday tools"),
				c("variables", "var and := declarations"),
				c("blank-identifier", "The blank identifier _"),
				c("constants", "const and typed constants"),
				c("untyped-constants", "Untyped constants and their default types"),
				c("iota", "iota for enums"),
				c("basic-types", "bool, int, float64 and string"),
				c("complex-numbers", "Complex numbers"),
				c("math-package", "The math package and floating-point pitfalls"),
				c("integer-sizes", "Integer sizes and overflow wraparound"),
				c("explicit-conversion", "Explicit conversions between numeric types"),
				c("zero-values", "Zero values and why they matter"),
				c("string-bytes", "Strings as immutable byte sequences"),
				c("runes-utf8", "Runes and UTF-8 decoding"),
				c("string-literals", "Raw versus interpreted string literals"),
				c("if-with-init", "if with an init statement"),
				c("switch", "switch without fallthrough by default"),
				c("for-forms", "The three forms of for"),
				c("range-loops", "range over slices, strings and maps"),
				c("labels-break-continue", "Labels with break and continue"),
				c("goto-statement", "goto and why it is rare"),
				c("functions-basic", "Function declarations and parameters"),
				c("multiple-returns", "Multiple return values"),
				c("named-results", "Named results and the naked return trap"),
				c("errors-as-values", "Returning errors instead of exceptions"),
				c("defer-lifo", "defer runs in last-in first-out order"),
				c("shadowing", "Variable shadowing in nested scopes"),
				c("init-functions", "init functions and package initialization order"),
				c("fmt-printf", "fmt.Printf verbs and formatting"),
				c("fmt-debug-verbs", "%v, %+v, %#v and %T for debugging", "fmt-printf"),
				c("os-args", "Reading os.Args"),
				c("bufio-scanner", "bufio.Scanner for reading lines"),
				c("net-listen", "net.Listen and Accept"),
				c("table-test-first", "Writing a first table-driven test"),
			}},
			{Title: "Key-value store and commands", Level: 2, Concepts: []Concept{
				c("slices-view", "Slices as views over an array"),
				c("append-growth", "append and capacity growth", "slices-view"),
				c("slice-aliasing", "Slicing shares memory", "slices-view"),
				c("copy-builtin", "The copy builtin"),
				c("arrays-value", "Arrays are values and get copied"),
				c("maps-basics", "Maps and comma-ok lookups"),
				c("nil-map-write", "Writing to a nil map panics"),
				c("struct-types", "Struct types and field tags"),
				c("anonymous-structs", "Anonymous structs for one-off data"),
				c("composite-literals", "Composite literals for structs, slices and maps"),
				c("struct-equality", "Comparable structs and map keys"),
				c("methods", "Methods and receivers"),
				c("method-values-expressions", "Method values and method expressions", "methods"),
				c("receiver-choice", "Choosing value versus pointer receivers", "methods"),
				c("implicit-interfaces", "Implicit interface satisfaction"),
				c("small-interfaces", "Small interfaces defined by the consumer", "implicit-interfaces"),
				c("typed-nil-interfaces", "Why a typed nil is not a nil interface", "implicit-interfaces"),
				c("type-switch", "Type switches"),
				c("type-assertion", "Type assertions with the ok form", "implicit-interfaces"),
				c("func-values", "Functions as values and handler types"),
				c("closures", "Closures capturing variables"),
				c("loop-var-per-iteration", "Per-iteration loop variables since Go 1.22", "closures"),
				c("defer-evaluation", "When deferred arguments are evaluated", "defer-lifo"),
				c("struct-embedding", "Struct embedding and promoted methods"),
				c("stringer", "fmt.Stringer for printing"),
				c("strconv", "strconv for parsing and formatting numbers"),
				c("strings-pkg", "The strings package"),
				c("bytes-pkg", "The bytes package and its relation to strings"),
				c("unicode-utf8", "unicode and unicode/utf8"),
				c("regexp-basics", "regexp for pattern matching and its costs"),
				c("slices-pkg", "The slices and maps packages"),
				c("sort-pkg", "Sorting with sort and slices"),
				c("min-max-clear", "min, max and clear builtins"),
				c("custom-errors", "Custom error types"),
				c("error-wrapping", "Wrapping errors with %w"),
				c("errors-is-as", "errors.Is and errors.As", "error-wrapping"),
				c("errors-join", "errors.Join for multiple errors", "error-wrapping"),
				c("sentinel-errors", "Sentinel errors and their tradeoffs"),
				c("panic-recover", "panic and recover at boundaries"),
				c("range-over-func", "Range over function iterators", "closures"),
				c("iter-seq-pull", "iter.Seq and iter.Pull", "range-over-func"),
			}},
			{Title: "Design idioms for Go APIs", Level: 2, Concepts: []Concept{
				c("accept-interfaces-return-structs", "Accept interfaces, return concrete types", "small-interfaces"),
				c("functional-options", "The functional options pattern", "func-values"),
				c("zero-value-useful", "Types whose zero value is useful", "zero-values"),
				c("package-naming", "Package naming and layout"),
				c("exported-api-design", "Designing exported APIs and doc comments"),
				c("api-stability", "API stability and deprecation"),
				c("composition-over-inheritance", "Composition over inheritance", "struct-embedding"),
				c("dont-panic-control-flow", "Why not to panic for control flow", "panic-recover"),
				c("error-handling-once", "Handle or return an error, not both", "errors-is-as"),
				c("review-comments", "Reading Go code review comments and idioms"),
			}},
			{Title: "Shared state with goroutines", Level: 4, Concepts: []Concept{
				c("goroutines", "Starting goroutines and their lifecycle"),
				c("unbuffered-channels", "Unbuffered channels as synchronization", "goroutines"),
				c("buffered-channels", "Buffered channels and capacity", "unbuffered-channels"),
				c("close-range-channels", "Closing channels and ranging over them", "unbuffered-channels"),
				c("channel-directions", "Send-only and receive-only channel types", "unbuffered-channels"),
				c("select", "select with default and timeouts", "unbuffered-channels"),
				c("mutex", "sync.Mutex for protecting state", "goroutines"),
				c("rwmutex", "sync.RWMutex for read-heavy state", "mutex"),
				c("sync-cond", "sync.Cond for waiting on state changes", "mutex"),
				c("sync-map", "sync.Map and when it is worth using", "mutex"),
				c("waitgroup", "sync.WaitGroup for joining work", "goroutines"),
				c("sync-once", "sync.Once for one-time setup"),
				c("atomic-ops", "sync/atomic counters and flags"),
				c("typed-atomics", "Typed atomics such as atomic.Int64", "atomic-ops"),
				c("race-detector", "The race detector and -race", "goroutines"),
				c("common-races", "Common data races: loop variables and shared maps", "race-detector"),
				c("memory-model", "The Go memory model and happens-before", "goroutines", "mutex"),
				c("goroutine-leaks", "Recognising goroutine leaks", "goroutines"),
				c("deadlocks", "Reading deadlock errors"),
				c("singleflight", "Deduplicating concurrent calls with singleflight", "goroutines"),
				c("weighted-semaphore", "Bounding concurrency with a weighted semaphore", "goroutines"),
				c("waitgroup-go", "WaitGroup.Go for starting tracked goroutines", "waitgroup"),
			}},
			{Title: "Expiry, timers and cancellation", Level: 4, Concepts: []Concept{
				c("time-duration", "time.Duration arithmetic"),
				c("timers-tickers", "time.After, time.Timer and time.Ticker"),
				c("time-zones-monotonic", "Time zones and the monotonic clock"),
				c("time-formatting-layouts", "Time formatting with reference layouts"),
				c("context-basics", "The context.Context contract"),
				c("context-cancel", "WithCancel and WithTimeout", "context-basics"),
				c("cancel-cause", "Causes of cancellation with WithCancelCause", "context-cancel"),
				c("context-afterfunc", "context.AfterFunc and WithoutCancel", "context-cancel"),
				c("context-values", "Request-scoped context values and their limits", "context-basics"),
				c("rate-limiting", "Rate limiting with golang.org/x/time/rate", "timers-tickers"),
				c("expiry-strategies", "Lazy versus active expiry"),
			}},
			{Title: "Append-only persistence", Level: 5, Concepts: []Concept{
				c("io-reader-writer", "io.Reader and io.Writer"),
				c("io-eof", "io.EOF and partial reads"),
				c("io-copy-pipe", "io.Copy and io.Pipe", "io-reader-writer"),
				c("os-files", "Opening, creating and permissions with os"),
				c("file-paths-filepath", "Paths with path/filepath"),
				c("bufio-writer", "Buffered writing and Flush"),
				c("fsync", "Durability with File.Sync", "os-files"),
				c("close-check-errors", "Checking the error from Close"),
				c("encoding-json", "encoding/json for snapshots"),
				c("encoding-json-streaming", "Streaming JSON with Decoder and Encoder", "encoding-json"),
				c("encoding-binary", "encoding/binary for compact records"),
				c("encoding-base64-hex", "base64 and hex encodings"),
				c("csv-xml", "encoding/csv and encoding/xml"),
			}},
			{Title: "Performance and the runtime", Level: 6, Concepts: []Concept{
				c("gmp-scheduler", "The goroutine scheduler: G, M and P", "goroutines"),
				c("escape-analysis", "Escape analysis and why values move to the heap"),
				c("stack-heap", "Stack versus heap allocation"),
				c("gc-overview", "How the garbage collector works at a high level"),
				c("gc-tuning", "GOGC and GOMEMLIMIT", "gc-overview"),
				c("preallocation", "Preallocating slices and maps"),
				c("string-builder", "strings.Builder and []byte conversion costs"),
				c("sync-pool", "sync.Pool and its limits"),
				c("benchmarks", "testing.B and b.ReportAllocs"),
				c("benchstat-comparison", "Comparing benchmark runs with benchstat", "benchmarks"),
				c("pprof-cpu-heap", "CPU and heap profiles with pprof"),
				c("pgo", "Profile-guided optimization", "pprof-cpu-heap"),
				c("execution-trace", "runtime/trace for scheduling problems"),
				c("runtime-introspection", "runtime.NumGoroutine and GOMAXPROCS"),
			}},
			{Title: "Tests, quality and tooling", Level: 6, Concepts: []Concept{
				c("table-driven-tests", "Table-driven tests in depth", "table-test-first"),
				c("subtests", "Subtests with t.Run"),
				c("t-helper-cleanup", "t.Helper and t.Cleanup"),
				c("t-parallel", "t.Parallel and test isolation"),
				c("testdata-fixtures", "Fixtures in testdata directories"),
				c("fakes-interfaces", "Fakes through small interfaces", "small-interfaces"),
				c("golden-files", "Golden file tests"),
				c("test-coverage", "Coverage profiles"),
				c("fuzz-testing", "Fuzz tests"),
				c("static-analysis", "staticcheck and go vet findings"),
				c("test-flags", "go test flags: -run, -count, -race, -v"),
				c("test-json-output", "Machine-readable output with go test -json"),
				c("gopls-editor", "gopls and editor integration"),
				c("delve-debugger", "Debugging with Delve"),
			}},
			{Title: "Advanced: reflection, unsafe and cgo", Level: 8, Concepts: []Concept{
				c("reflect-basics", "reflect.TypeOf and reflect.ValueOf"),
				c("struct-tags-reflect", "Reading struct tags with reflection", "reflect-basics"),
				c("unsafe-pointer", "unsafe.Pointer and its conversion rules"),
				c("unsafe-layout", "unsafe.Sizeof, Alignof and Offsetof", "unsafe-pointer"),
				c("cgo-basics", "Calling C from Go with cgo"),
				c("cgo-costs", "The cost of crossing the cgo boundary", "cgo-basics"),
				c("compiler-output", "Reading compiler output with -gcflags=-S"),
				c("unique-weak", "unique handles and weak pointers"),
			}},
			{Title: "Shipping and supply chain", Level: 8, Concepts: []Concept{
				c("semver", "Semantic versioning for modules"),
				c("go-mod-tidy-vendor", "go mod tidy and vendoring"),
				c("go-sum-checksums", "go.sum and the checksum database"),
				c("goproxy-private", "GOPROXY and GOPRIVATE for private modules"),
				c("go-mod-replace", "replace and retract directives", "go-mod-tidy-vendor"),
				c("go-work", "Workspaces with go.work"),
				c("build-tags", "Build tags"),
				c("cross-compile", "Cross-compiling with GOOS and GOARCH"),
				c("ldflags-version", "Injecting version info with -ldflags"),
				c("go-generate", "go:generate directives"),
				c("embed-files", "Embedding files with the embed package"),
				c("publish-module", "Publishing a module"),
				c("reproducible-builds", "Reproducible builds with -trimpath"),
			}},
		},
	},
	{
		Slug:    "jobq",
		Name:    "Build your Job Queue",
		Summary: "An HTTP job API backed by Postgres, with a worker pool, retries, realtime progress, gRPC and Connect services, security hardening and a CLI.",
		Stages: []Stage{
			{Title: "Generics and pluggable jobs", Level: 2, Concepts: []Concept{
				c("job-interface", "A Job interface", "implicit-interfaces"),
				c("interface-embedding", "Embedding interfaces"),
				c("handler-registry", "A handler registry keyed by name", "job-interface"),
				c("type-params", "Type parameters on functions and types"),
				c("constraints", "Constraints and type sets", "type-params"),
				c("generic-queue", "A generic Queue[T]", "type-params"),
				c("type-inference-generics", "Type inference for generic calls", "type-params"),
				c("typed-payloads", "Type-safe payload decoding", "generic-queue"),
				c("why-generics", "Generics versus interface{} and any"),
				c("go-plugins", "The plugin package and its tradeoffs"),
			}},
			{Title: "HTTP API with the standard library", Level: 3, Concepts: []Concept{
				c("http-handler", "http.Handler and http.HandlerFunc"),
				c("servemux-patterns", "ServeMux with methods and path wildcards"),
				c("json-marshal", "Encoding and decoding JSON with encoding/json"),
				c("json-tags", "JSON field tags and omitempty"),
				c("http-status", "Status codes and when to use them"),
				c("body-limit", "Limiting request bodies with MaxBytesReader"),
				c("httptest-recorder", "Testing handlers with httptest"),
				c("server-timeouts", "http.Server timeouts"),
				c("flag-config", "Configuration with the flag package"),
			}},
			{Title: "Routing and middleware with chi", Level: 3, Concepts: []Concept{
				c("chi-router", "Routing with the chi package", "servemux-patterns"),
				c("middleware-chain", "Middleware as func(http.Handler) http.Handler", "http-handler"),
				c("request-logging", "Request logging middleware"),
				c("error-responses", "A consistent JSON error format", "http-status"),
				c("request-context", "Request context from r.Context()"),
				c("http-client", "net/http client with timeouts"),
				c("url-parsing", "Parsing and building URLs with net/url"),
			}},
			{Title: "Worker pool and dispatch", Level: 4, Concepts: []Concept{
				c("dispatcher", "A dispatcher goroutine feeding workers"),
				c("pool-sizing", "Choosing the number of workers"),
				c("backpressure", "Backpressure with bounded channels"),
				c("fan-out-workers", "Fanning jobs out to workers"),
				c("result-pipeline", "Pipelines that collect results"),
				c("ticker-polling", "Polling for jobs with a ticker"),
				c("concurrency-patterns", "Common concurrency patterns for workers"),
				c("errgroup", "errgroup for coordinated goroutines", "waitgroup"),
			}},
			{Title: "Cancellation and graceful shutdown", Level: 4, Concepts: []Concept{
				c("per-job-timeouts", "Per-job timeouts", "context-cancel"),
				c("context-propagation", "Passing context through every layer", "context-basics"),
				c("cancel-vs-deadline", "Cancellation versus deadline"),
				c("signal-notify-context", "signal.NotifyContext for SIGINT and SIGTERM", "context-cancel"),
				c("graceful-drain", "Draining in-flight jobs before exit"),
				c("http-server-shutdown", "Shutting down an http.Server", "server-timeouts"),
				c("shutdown-tests", "Testing shutdown without leaking goroutines", "goroutine-leaks"),
				c("shared-state-shutdown", "Closing shared state safely", "mutex"),
			}},
			{Title: "Persistence with pgx and sqlc", Level: 5, Concepts: []Concept{
				c("database-sql", "database/sql: connections, pools and rows"),
				c("pgx-driver", "Connecting to Postgres with pgx", "database-sql"),
				c("migrations", "Schema migrations with golang-migrate"),
				c("sql-params", "Placeholders instead of string concatenation"),
				c("sqlc-generated", "Typed queries generated by sqlc", "sql-params"),
				c("transactions", "Transactions and rollback", "database-sql"),
				c("skip-locked-claims", "Claiming rows with FOR UPDATE SKIP LOCKED", "transactions"),
				c("rows-close", "Scanning rows and always closing them"),
				c("db-context", "Context-aware database calls", "context-cancel"),
				c("sql-nullable", "Nullable columns with sql.Null types"),
				c("repository-layer", "A repository layer between handlers and SQL"),
				c("import-rules", "Import cycles and package boundaries"),
				c("internal-packages", "internal packages"),
				c("module-deps", "Adding and upgrading dependencies"),
			}},
			{Title: "Retries and failure handling", Level: 6, Concepts: []Concept{
				c("exponential-backoff", "Exponential backoff"),
				c("jitter", "Jitter against thundering herds"),
				c("retry-classification", "Retryable versus permanent errors", "errors-is-as"),
				c("dead-letter-queue", "Dead-letter queues", "retry-classification"),
				c("idempotency-keys", "Idempotency keys"),
				c("panic-in-workers", "Recovering panics inside goroutines", "panic-recover"),
				c("circuit-breaker", "The circuit breaker pattern"),
			}},
			{Title: "Observability", Level: 6, Concepts: []Concept{
				c("slog-logger", "Structured logging with log/slog"),
				c("log-levels-attrs", "Log levels and attributes"),
				c("log-redaction", "Keeping secrets out of logs"),
				c("expvar-metrics", "Counters and gauges with expvar"),
				c("pprof-endpoint", "A pprof endpoint in a running service", "pprof-cpu-heap"),
				c("worker-trace", "Execution traces for a running worker", "execution-trace"),
				c("health-readiness", "Health and readiness endpoints"),
				c("error-log-context", "Logging errors with context"),
			}},
			{Title: "Realtime progress over SSE and WebSockets", Level: 7, Concepts: []Concept{
				c("sse", "Server-sent events for job progress", "http-handler"),
				c("sse-flushing", "Flushing responses and detecting disconnects", "sse"),
				c("websocket", "WebSockets with coder/websocket", "http-handler"),
				c("websocket-lifecycle", "WebSocket ping, close frames and read loops", "websocket"),
			}},
			{Title: "gRPC in depth", Level: 7, Concepts: []Concept{
				c("protobuf-messages", "Protocol buffer messages, fields and field numbers"),
				c("proto3-presence", "Proto3 defaults and field presence", "protobuf-messages"),
				c("protoc-codegen", "Generating Go code with protoc and buf", "protobuf-messages"),
				c("grpc-service-def", "Service definitions and the four RPC method types", "protobuf-messages"),
				c("grpc-unary", "Unary RPCs with google.golang.org/grpc", "grpc-service-def", "protoc-codegen"),
				c("grpc-streaming", "Server, client and bidirectional streaming", "grpc-unary"),
				c("grpc-status-codes", "gRPC status codes and error details", "grpc-unary"),
				c("grpc-metadata", "Metadata as headers and trailers", "grpc-unary"),
				c("grpc-interceptors", "Unary and stream interceptors", "grpc-unary"),
				c("grpc-deadlines", "Deadlines and cancellation over the wire", "context-cancel", "grpc-unary"),
				c("grpc-tls", "Transport credentials and TLS"),
				c("grpc-health-reflection", "gRPC health checking and server reflection", "grpc-unary"),
				c("grpc-load-balancing", "Name resolution and client-side load balancing", "grpc-unary"),
			}},
			{Title: "Connect RPC in depth", Level: 7, Concepts: []Concept{
				c("connect-protocol", "The Connect protocol over HTTP with JSON or protobuf", "protobuf-messages"),
				c("connect-handlers", "Serving Connect handlers on net/http", "connect-protocol", "http-handler"),
				c("connect-clients", "Connect clients and generated stubs", "connect-handlers", "protoc-codegen"),
				c("connect-errors", "connect.Error and Connect error codes", "connect-handlers"),
				c("connect-interceptors", "Connect interceptors", "connect-handlers"),
				c("connect-grpc-compat", "Talking to gRPC clients and calling with curl", "connect-protocol", "grpc-unary"),
				c("streaming-rpc", "Streaming over Connect", "connect-clients", "grpc-streaming"),
				c("rpc-deadlines", "Deadlines on Connect calls", "context-cancel", "connect-clients"),
				c("transport-choice", "Choosing SSE, WebSocket, gRPC or Connect", "sse", "websocket", "grpc-streaming"),
			}},
			{Title: "Security essentials", Level: 8, Concepts: []Concept{
				c("tls-basics", "TLS handshakes, certificates and verification"),
				c("crypto-rand", "crypto/rand versus math/rand"),
				c("password-hashing", "Hashing passwords with bcrypt or argon2"),
				c("constant-time-compare", "Constant-time comparison with crypto/subtle"),
				c("input-validation", "Validating input at trust boundaries"),
				c("html-template-xss", "html/template and preventing XSS", "input-validation"),
				c("hmac-hashing", "HMAC and SHA-256 with crypto/hmac and crypto/sha256"),
				c("authn-tokens", "Bearer tokens and common JWT pitfalls"),
				c("authz-middleware", "Authorization checks in middleware", "middleware-chain"),
				c("secrets-management", "Keeping secrets out of code and images"),
				c("gosec-scan", "Security linting with gosec"),
				c("govulncheck", "Vulnerability scanning with govulncheck"),
			}},
			{Title: "CLI, TUI and release", Level: 8, Concepts: []Concept{
				c("kong-cli", "Command-line parsing with kong", "flag-config"),
				c("subcommands", "Subcommands and help text", "kong-cli"),
				c("os-exec", "Running subprocesses with os/exec"),
				c("text-template", "Report output with text/template"),
				c("tview-tui", "A terminal dashboard with tview"),
				c("doc-comments", "Doc comments and go doc"),
				c("go-install", "Installing binaries with go install"),
				c("golangci-lint", "golangci-lint configuration"),
				c("per-os-builds", "Per-OS files with build constraints", "build-tags"),
				c("os-exit-defer", "os.Exit skips deferred calls"),
				c("release-checklist", "A release checklist for a Go binary"),
			}},
		},
	},
	k8sProject,
}

// Path returns every stage in the single learning order: by level, then by
// project order, then by stage order within the project.
func Path() []PathStep {
	var steps []PathStep
	for _, lv := range Levels {
		for _, p := range Projects {
			for i, s := range p.Stages {
				if s.Level == lv.Number {
					steps = append(steps, PathStep{Level: lv, Project: p, StageNumber: i + 1, Stage: s})
				}
			}
		}
	}
	return steps
}

// TopicCount returns the number of concepts across all stages of the project.
func (p Project) TopicCount() int {
	n := 0
	for _, s := range p.Stages {
		n += len(s.Concepts)
	}
	return n
}
