package curriculum

// Theme is the story a project is told in.
type Theme struct {
	Name  string
	Story string
}

// Bug is a deliberately broken behaviour left in the learner's project. The
// learner diagnoses it, answers the concept question perfectly, then fixes it.
//
// Only learner-visible fields live here. The injection, root cause, rubric and
// reference answer are in the separate keys repository and are used by the
// injector and by `anvil grade`.
type Bug struct {
	Concept     string   // ID of the concept under test; taught in this stage or earlier
	Symptom     string   // what the learner observes
	Reproduce   string   // command that shows the symptom
	Hints       []string // exactly three, revealed one at a time on request
	Question    string   // concept question the learner must answer perfectly
	FixCriteria string   // how the learner's fix is verified
}

// StageContent is the story and bug for one stage.
type StageContent struct {
	Scene string
	Bug   Bug
}

var themes = map[string]Theme{
	"redis": {
		Name:  "The Lighthouse Ledger",
		Story: "You keep the logbook of a lighthouse on a dangerous coast. Every message from a ship must be stored whole, kept safe and answered truthfully. Each stage repairs one part of the ledger.",
	},
	"jobq": {
		Name:  "The Dispatch Office",
		Story: "A courier office takes delivery jobs from customers, sends couriers to carry them and must never lose a parcel. Each stage gives the office a new process.",
	},
	"k8s": {
		Name:  "The Fleet Harbour",
		Story: "A harbour keeps a fleet of ships that must always match the plan in the register. Your controller is the harbour master: it compares the plan with reality and corrects the difference, again and again.",
	},
}

// stageContents is keyed by "<project slug>/<stage title>".
var stageContents = map[string]StageContent{
	// ---------------------------------------------------------------- redis
	"redis/RESP reader and TCP server": {
		Scene: "The first signal from a ship must arrive whole, even when it arrives in pieces.",
		Bug: Bug{
			Concept:   "bufio-scanner",
			Symptom:   "A SET whose value contains a newline stores only the text before the newline. The client still gets +OK.",
			Reproduce: "go test ./... -run TestBulkStringWithNewline",
			Hints: []string{
				"Look at how the RESP bulk-string header states the payload length.",
				"What delimiter does a line scanner use, and does RESP use the same rule to find the end of the payload?",
				"Which reader reads exactly a given number of bytes?",
			},
			Question:    "Why can a line-based scanner not read a RESP bulk string safely, and what framing rule does RESP use instead?",
			FixCriteria: "TestBulkStringWithNewline and TestBulkStringBinary (payload with \\x00 and \\r\\n) both pass.",
		},
	},
	"redis/Key-value store and commands": {
		Scene: "Ships report their cargo to the ledger. Each record must stay true after the next message arrives.",
		Bug: Bug{
			Concept:   "slice-aliasing",
			Symptom:   "GET returns a value that later commands have overwritten.",
			Reproduce: "go test ./... -run TestStoredValueSurvivesNextCommand",
			Hints: []string{
				"Where does args[2] point in memory?",
				"Who reuses that memory after the command finishes?",
				"What does copy do that slicing does not?",
			},
			Question:    "Why does storing a subslice of a reused buffer corrupt stored values, and what must the store do to own its data?",
			FixCriteria: "TestStoredValueSurvivesNextCommand passes, and the existing SET and GET tests still pass.",
		},
	},
	"redis/Design idioms for Go APIs": {
		Scene: "The keeper's rule: anything shown to other crews must be easy to replace and easy to test.",
		Bug: Bug{
			Concept:   "accept-interfaces-return-structs",
			Symptom:   "The stats handler panics with a failed type assertion when a test passes in its own fake server.",
			Reproduce: "go test ./... -run TestStatsWithFakeServer",
			Hints: []string{
				"What does the stats handler need that the Server interface does not offer?",
				"Which types does the failing test supply?",
				"Could the consumer declare the one method it needs as its own small interface?",
			},
			Question:    "Why should a constructor usually return a concrete type, and where should the small interface a consumer needs be declared?",
			FixCriteria: "TestStatsWithFakeServer passes without any type assertion on the server.",
		},
	},
	"redis/Shared state with goroutines": {
		Scene: "Two ships reach the harbour book at the same time. Only one may write at once, or the entries tear.",
		Bug: Bug{
			Concept:   "mutex",
			Symptom:   "Under concurrent SET load some keys go missing. go test -race reports a data race on the store map.",
			Reproduce: "go test -race ./... -run TestConcurrentSet",
			Hints: []string{
				"Which lines does the lock have to cover?",
				"What does the race detector name as shared?",
				"Where does Unlock belong relative to the write?",
			},
			Question:    "What exactly does a mutex protect, and why is unlocking before the shared write wrong?",
			FixCriteria: "TestConcurrentSet passes with -race and -count=20.",
		},
	},
	"redis/Expiry, timers and cancellation": {
		Scene: "Each key carries a countdown. When the key is deleted, its countdown must stop too.",
		Bug: Bug{
			Concept:   "timers-tickers",
			Symptom:   "After many SET, EXPIRE and DEL cycles with long TTLs, runtime.NumGoroutine keeps growing.",
			Reproduce: "go test ./... -run TestDeleteDoesNotLeakGoroutines",
			Hints: []string{
				"What is the expiry goroutine doing between EXPIRE and the timer firing?",
				"What happens to that goroutine when DEL runs?",
				"Which timer can be stopped, and which channel can the goroutine also watch?",
			},
			Question:    "How can a key's expiry be cancelled when the key is deleted, and what is wrong with a goroutine blocked on time.After?",
			FixCriteria: "TestDeleteDoesNotLeakGoroutines passes, and expired keys still disappear from GET.",
		},
	},
	"redis/Append-only persistence": {
		Scene: "Every change is written to the logbook before the ship is answered. A failed write must never be reported as saved.",
		Bug: Bug{
			Concept:   "bufio-writer",
			Symptom:   "The client gets +OK after SET. After a restart the key is gone. The test uses a writer whose flush fails.",
			Reproduce: "go test ./... -run TestWriteFailureIsReported",
			Hints: []string{
				"Where do bytes actually reach the file?",
				"Which call reports that the write failed?",
				"When should the client receive its reply?",
			},
			Question:    "What does bufio.Writer buffering mean for when a write error appears, and when must a durable write report success?",
			FixCriteria: "TestWriteFailureIsReported passes, and the client gets an error when the write fails.",
		},
	},
	"redis/Performance and the runtime": {
		Scene: "The log has to keep up during a storm. Each allocation on the hot path costs time.",
		Bug: Bug{
			Concept:   "string-builder",
			Symptom:   "The array-reply benchmark slows down quadratically: ten times the entries takes about a hundred times as long.",
			Reproduce: "go test ./... -run TestEncodeArrayScalesLinearly",
			Hints: []string{
				"How many times is the growing reply copied?",
				"What does -benchmem report about allocations?",
				"Which type appends to one buffer without copying the whole string each time?",
			},
			Question:    "Why is building a string with += in a loop expensive, and what does strings.Builder do differently?",
			FixCriteria: "Time per entry stays roughly constant as the entry count grows, and allocs/op falls.",
		},
	},
	"redis/Tests, quality and tooling": {
		Scene: "The test crew shares one logbook. Their checks must not trample each other.",
		Bug: Bug{
			Concept:   "t-parallel",
			Symptom:   "TestGetMissing and TestSetThenGet fail now and then. go test -race reports races on the store.",
			Reproduce: "go test -race -count=20 ./... -run 'TestGetMissing|TestSetThenGet'",
			Hints: []string{
				"What do the tests share?",
				"What does t.Parallel change about when tests run?",
				"How can each test get its own state?",
			},
			Question:    "What does t.Parallel guarantee and not guarantee, and how should parallel tests avoid sharing state?",
			FixCriteria: "go test -race -count=50 passes.",
		},
	},
	"redis/Advanced: reflection, unsafe and cgo": {
		Scene: "A fast path lets the ledger point straight at the bytes of an incoming message. A pointer like that must never outlive its owner.",
		Bug: Bug{
			Concept:   "unsafe-pointer",
			Symptom:   "Keys read from the store change after later commands. GET returns garbage for keys that were correct a moment earlier.",
			Reproduce: "go test ./... -run TestKeysSurviveBufferReuse",
			Hints: []string{
				"Who owns the memory behind that string?",
				"How long does the connection's buffer live, and when is it reused?",
				"Which rule about unsafe pointer use does this break?",
			},
			Question:    "When is it safe to use unsafe to view bytes as a string, and what must hold about the memory's lifetime and mutation?",
			FixCriteria: "TestKeysSurviveBufferReuse passes. The fast path either copies the key or is removed.",
		},
	},
	"redis/Shipping and supply chain": {
		Scene: "The release crew stamps each ship with its build number before it sails.",
		Bug: Bug{
			Concept:   "ldflags-version",
			Symptom:   "A release built with -ldflags '-X main.version=1.2.0' still prints version dev.",
			Reproduce: "go build -ldflags '-X main.version=1.2.0' -o \"$TMPDIR/redis-release\" ./cmd/redis && \"$TMPDIR/redis-release\" --version",
			Hints: []string{
				"Which kind of declaration can the linker replace?",
				"What does the -X flag require from its target?",
				"Is the version declared with var or with const?",
			},
			Question:    "How does -ldflags -X inject a value at build time, and what must the target declaration look like?",
			FixCriteria: "The release build prints 1.2.0, with the target declared as a var.",
		},
	},

	// ---------------------------------------------------------------- jobq
	"jobq/Generics and pluggable jobs": {
		Scene: "The office keeps a rack of job forms. Each form must match the courier who will carry it.",
		Bug: Bug{
			Concept:   "typed-payloads",
			Symptom:   "Submitting a job of a type nobody registered crashes the server instead of returning an error.",
			Reproduce: "go test ./... -run TestDecodeUnknownPayloadReturnsError",
			Hints: []string{
				"What happens when a type assertion without the ok form fails?",
				"What static type does JSON decoding produce?",
				"How can a generic function report a mismatch without panicking?",
			},
			Question:    "Why don't generics remove runtime type checks at a JSON boundary, and how should a typed decode report a mismatch?",
			FixCriteria: "TestDecodeUnknownPayloadReturnsError passes with an error and no panic.",
		},
	},
	"jobq/HTTP API with the standard library": {
		Scene: "Customers drop parcels at the front desk. The desk takes what fits and turns away what does not.",
		Bug: Bug{
			Concept:   "body-limit",
			Symptom:   "A 2 MB POST /jobs body is accepted with 201 Created, when it should be rejected with 413.",
			Reproduce: "go test ./... -run TestSubmitRejectsLargeBody",
			Hints: []string{
				"Where in the handler is the body read?",
				"Which wrapper caps how many bytes a reader returns?",
				"What status code does the test expect for an oversized body?",
			},
			Question:    "Why must a handler limit how much of a request body it reads, and how does MaxBytesReader report that the limit was hit?",
			FixCriteria: "TestSubmitRejectsLargeBody passes with 413.",
		},
	},
	"jobq/Routing and middleware with chi": {
		Scene: "Only signed-in couriers may read the dispatch board. The door check must cover every route on the board.",
		Bug: Bug{
			Concept:   "middleware-chain",
			Symptom:   "GET /jobs without a token returns 200 with the job list.",
			Reproduce: "go test ./... -run TestJobsRequireToken",
			Hints: []string{
				"Which router or group does the auth middleware belong to?",
				"Which group does GET /jobs belong to now?",
				"What does chi attach a middleware to?",
			},
			Question:    "How does chi decide which routes a middleware applies to, and how can a route escape authentication?",
			FixCriteria: "TestJobsRequireToken passes for every protected route.",
		},
	},
	"jobq/Worker pool and dispatch": {
		Scene: "The office has a fixed crew of couriers. Work waits at the counter when the crew is busy, and the counter has a limit.",
		Bug: Bug{
			Concept:   "backpressure",
			Symptom:   "Enqueuing 100000 jobs quickly grows the process to hundreds of MB, and runtime.NumGoroutine reaches about 100000.",
			Reproduce: "go test ./... -run TestDispatchBoundsGoroutines",
			Hints: []string{
				"What limits how much work is in flight now?",
				"What happens to a sender when a bounded channel is full?",
				"Who should absorb overload: the producer, or the memory?",
			},
			Question:    "What is backpressure, and why does starting one goroutine per job defeat it?",
			FixCriteria: "TestDispatchBoundsGoroutines passes. Producers block or get an error when the queue is full.",
		},
	},
	"jobq/Cancellation and graceful shutdown": {
		Scene: "When the office closes, couriers on the road finish their route or are recalled within their time limit.",
		Bug: Bug{
			Concept:   "per-job-timeouts",
			Symptom:   "A job with a 1 s timeout runs for 30 s and is never cancelled.",
			Reproduce: "go test ./... -run TestJobTimeoutCancelsHandler",
			Hints: []string{
				"Which context does the handler actually receive?",
				"Which context should a per-job timeout derive from?",
				"Which code has to observe ctx.Done() for the timeout to matter?",
			},
			Question:    "How does a per-job timeout actually stop work, and why does passing context.Background() defeat it?",
			FixCriteria: "TestJobTimeoutCancelsHandler passes. The handler returns context.DeadlineExceeded.",
		},
	},
	"jobq/Persistence with pgx and sqlc": {
		Scene: "Two couriers reach for the same parcel at once. Only one may carry it.",
		Bug: Bug{
			Concept:   "skip-locked-claims",
			Symptom:   "With 10 workers and 1000 queued jobs, some jobs run twice.",
			Reproduce: "ANVIL_TEST_PG_DSN=<postgres dsn> go test ./... -run TestClaimsAreExclusive",
			Hints: []string{
				"What can happen between the SELECT and the UPDATE?",
				"Which single statement can claim and update at once?",
				"What does FOR UPDATE lock, and what does SKIP LOCKED change about waiting?",
			},
			Question:    "How does a worker claim a job exclusively in Postgres, and what does SKIP LOCKED add to FOR UPDATE?",
			FixCriteria: "TestClaimsAreExclusive passes with 10 workers over 1000 jobs.",
		},
	},
	"jobq/Retries and failure handling": {
		Scene: "Some parcels can be delivered later. Others never will. The office must tell them apart.",
		Bug: Bug{
			Concept:   "retry-classification",
			Symptom:   "A validation failure is retried five times before it reaches the dead-letter queue.",
			Reproduce: "go test ./... -run TestPermanentErrorGoesStraightToDeadLetter",
			Hints: []string{
				"What does errors.Is search for?",
				"How does %w differ from %v?",
				"Where is the sentinel created, and where is it checked?",
			},
			Question:    "How does errors.Is decide that an error is permanent, and what breaks that check when an error is wrapped with %v?",
			FixCriteria: "TestPermanentErrorGoesStraightToDeadLetter passes with an attempt count of 1.",
		},
	},
	"jobq/Observability": {
		Scene: "The office keeps a logbook for its auditors. Customers' secrets must never be written in it.",
		Bug: Bug{
			Concept:   "log-redaction",
			Symptom:   "The submit log line contains the caller's API token.",
			Reproduce: "go test ./... -run TestLogsDoNotContainToken",
			Hints: []string{
				"Which value ends up in the log line?",
				"Which slog interface controls how a type is rendered?",
				"Is the right fix per call site or per type?",
			},
			Question:    "How can a type control what slog writes for it, and why is redaction better placed on the type than at each log call?",
			FixCriteria: "TestLogsDoNotContainToken passes. The token is redacted through LogValue.",
		},
	},
	"jobq/Realtime progress over SSE and WebSockets": {
		Scene: "Customers watch their parcel move across the board, one update at a time.",
		Bug: Bug{
			Concept:   "sse-flushing",
			Symptom:   "The SSE client receives every progress event only when the job finishes.",
			Reproduce: "go test ./... -run TestSSEDeliversEventsBeforeCompletion",
			Hints: []string{
				"Which interface does the handler check for?",
				"Does the wrapper type implement that interface?",
				"Which Go 1.20 helper gives access to flushing through a wrapper?",
			},
			Question:    "Why does wrapping http.ResponseWriter break streaming, and how can middleware keep Flush available?",
			FixCriteria: "TestSSEDeliversEventsBeforeCompletion passes. Events arrive before the job ends.",
		},
	},
	"jobq/gRPC in depth": {
		Scene: "The office phones the dispatch centre. Every call has a time limit, and the limit must travel with the call.",
		Bug: Bug{
			Concept:   "grpc-deadlines",
			Symptom:   "A handler with a 2 s deadline waits five minutes when the worker service stalls.",
			Reproduce: "go test ./... -run TestWorkerCallHonoursDeadline",
			Hints: []string{
				"Which context does the RPC call receive?",
				"Where does the deadline set for the handler live?",
				"What status should the call return when its deadline passes?",
			},
			Question:    "How does a gRPC deadline travel from caller to server, and what happens to a call made with context.Background()?",
			FixCriteria: "TestWorkerCallHonoursDeadline passes, and the call returns DeadlineExceeded.",
		},
	},
	"jobq/Connect RPC in depth": {
		Scene: "Customers ask the desk on a standard form. The desk answers with a standard code, so every client understands what went wrong.",
		Bug: Bug{
			Concept:   "connect-errors",
			Symptom:   "Asking for an unknown job returns HTTP 500 with code unknown, instead of HTTP 404 with not_found.",
			Reproduce: "go test ./... -run TestGetUnknownJobIsNotFound",
			Hints: []string{
				"Which code does the client see?",
				"How does Connect decide which code to send?",
				"Which constructor attaches a Connect code to an error?",
			},
			Question:    "How does Connect choose the error code and HTTP status for a handler's error, and how should a not-found case be reported?",
			FixCriteria: "TestGetUnknownJobIsNotFound passes with code not_found and HTTP 404.",
		},
	},
	"jobq/Security essentials": {
		Scene: "Partners seal their webhook messages. The office opens a seal only when it matches the exact parcel that arrived.",
		Bug: Bug{
			Concept:   "hmac-hashing",
			Symptom:   "A webhook signed correctly by the partner is rejected with 401 when the JSON arrives with different spacing.",
			Reproduce: "go test ./... -run TestWebhookSignatureOverRawBody",
			Hints: []string{
				"What exactly did the sender sign?",
				"Does re-encoding JSON always produce the same bytes?",
				"Which bytes does the handler have before it parses anything?",
			},
			Question:    "What must an HMAC webhook signature cover, and why is verifying a re-encoded body wrong?",
			FixCriteria: "A valid signature over the raw body is accepted, and the same signature over altered bytes is rejected.",
		},
	},
	"jobq/CLI, TUI and release": {
		Scene: "The terminal desk takes orders. Each order form must pick up its own options, not the options on the form beside it.",
		Bug: Bug{
			Concept:   "subcommands",
			Symptom:   "jobq submit --priority 5 stores priority 0.",
			Reproduce: "go test ./... -run TestSubmitStoresPriority",
			Hints: []string{
				"Which struct holds the value submit reads?",
				"Which command's parser fills that struct?",
				"Where should each command's flags live?",
			},
			Question:    "How should a CLI parser scope flags to a subcommand, and why does sharing a flag variable between commands cause silent bugs?",
			FixCriteria: "TestSubmitStoresPriority passes with priority 5 stored.",
		},
	},

	// ---------------------------------------------------------------- k8s
	"k8s/The Kubernetes API model": {
		Scene: "The harbour keeps one official register. Every ship's record changes only through the register office.",
		Bug: Bug{
			Concept:   "resource-versions",
			Symptom:   "Two status updates race. One update disappears and the controller reports success.",
			Reproduce: "go test ./... -run TestStatusUpdateConflictIsRetried",
			Hints: []string{
				"What does the API server compare when you update an object?",
				"Which HTTP status signals a stale write?",
				"What should the code do with that error?",
			},
			Question:    "How does Kubernetes prevent lost updates, and what should a controller do when an update returns a Conflict?",
			FixCriteria: "TestStatusUpdateConflictIsRetried passes, and the final status matches the latest write.",
		},
	},
	"k8s/client-go in depth": {
		Scene: "The harbour clerk must read the right register page, and must handle a missing ship without losing the thread.",
		Bug: Bug{
			Concept:   "apierrors",
			Symptom:   "After a worker Job is deleted, the controller returns an error on every loop instead of recreating the Job.",
			Reproduce: "go test ./... -run TestMissingJobIsRecreated",
			Hints: []string{
				"What does apierrors.IsNotFound inspect?",
				"What does %v keep from the original error?",
				"Which verb keeps the chain intact?",
			},
			Question:    "How do the apierrors helpers recognise a NotFound, and why does wrapping an error with %v break that check?",
			FixCriteria: "TestMissingJobIsRecreated passes. The controller creates the Job after it is deleted.",
		},
	},
	"k8s/Informers, caches and workqueues": {
		Scene: "The harbour's lookup board is copied from the register. The clerk must not read the board before the copy is complete.",
		Bug: Bug{
			Concept:   "cache-sync-waits",
			Symptom:   "After a controller restart, duplicate worker Jobs appear for queues that already have one.",
			Reproduce: "go test ./... -run TestRestartDoesNotDuplicateJobs",
			Hints: []string{
				"What does a lister return before its initial list has loaded?",
				"Which call blocks until the cache holds its initial state?",
				"Where in startup must that call happen?",
			},
			Question:    "Why must a controller wait for its informer caches to sync before reading listers, and what goes wrong without it?",
			FixCriteria: "TestRestartDoesNotDuplicateJobs passes. A restart creates no new Jobs.",
		},
	},
	"k8s/Workloads: pods, jobs and scheduling": {
		Scene: "Each worker is a short-lived ship. It must finish or fail within its allowance, and it must never sail forever.",
		Bug: Bug{
			Concept:   "job-semantics",
			Symptom:   "A worker Job whose container hangs stays active for days, and the queue never drains.",
			Reproduce: "go test ./... -run TestStuckWorkerIsTerminated",
			Hints: []string{
				"Which Job fields limit how long a Job may run?",
				"Does backoffLimit apply to a pod that is running but stuck?",
				"Which field sets a hard limit on the Job's active time?",
			},
			Question:    "What does backoffLimit limit, what does activeDeadlineSeconds limit, and why does backoffLimit never fire for a stuck pod?",
			FixCriteria: "TestStuckWorkerIsTerminated passes. The Job fails with DeadlineExceeded.",
		},
	},
	"k8s/Networking and configuration": {
		Scene: "The harbour's signal board tells visitors which ship to find. If the board lists nobody, visitors get lost.",
		Bug: Bug{
			Concept:   "endpointslices",
			Symptom:   "The jobq-status Service has no endpoints. curl to the Service times out, although the worker pods are Ready.",
			Reproduce: "go test ./... -run TestStatusServiceHasEndpoints",
			Hints: []string{
				"What decides which pods a Service routes to?",
				"Compare the Service selector with the pod labels.",
				"Where do you look to see the endpoints a Service has?",
			},
			Question:    "How does a Service find its backends, and how would you tell from kubectl that a Service has no endpoints?",
			FixCriteria: "TestStatusServiceHasEndpoints passes. The labels and the selector match.",
		},
	},
	"k8s/Operators with controller-runtime": {
		Scene: "A deleted job record means the ship has already been dismissed. The clerk must stop calling about it.",
		Bug: Bug{
			Concept:   "reconcile-results",
			Symptom:   "After a JobQueue is deleted, the controller logs an error every few milliseconds, and its CPU stays high.",
			Reproduce: "go test ./... -run TestDeletedJobQueueStopsReconciling",
			Hints: []string{
				"What does NotFound mean for a reconcile of a deleted object?",
				"What does returning an error make the manager do?",
				"What does a nil error with an empty Result mean?",
			},
			Question:    "What should a reconcile return for a deleted object, and when should it return an error rather than a requeue?",
			FixCriteria: "TestDeletedJobQueueStopsReconciling passes. No error requeues after deletion.",
		},
	},
	"k8s/Testing and admission": {
		Scene: "Before a ship enters the harbour, an inspector checks its papers in a safe room that holds a copy of the register.",
		Bug: Bug{
			Concept:   "fake-client",
			Symptom:   "The reconciler test fails when it builds the fake client: no kind is registered for the JobQueue type.",
			Reproduce: "go test ./... -run TestReconcileCreatesJobs",
			Hints: []string{
				"What does the fake client need to know about each object type?",
				"Where are types added to a runtime.Scheme?",
				"Which builder method sets the scheme the fake client uses?",
			},
			Question:    "How does the controller-runtime fake client know which types it handles, and what must a test register?",
			FixCriteria: "TestReconcileCreatesJobs passes with the scheme registered.",
		},
	},
	"k8s/Deploying and operating": {
		Scene: "The harbour master runs two lookout towers. Only one may give orders at a time, or the fleet receives two plans.",
		Bug: Bug{
			Concept:   "operator-upgrades",
			Symptom:   "When the operator Deployment is scaled to two replicas, every worker Job is created twice.",
			Reproduce: "go test ./... -run TestTwoReplicasDoNotDuplicateJobs",
			Hints: []string{
				"How many replicas are reconciling right now?",
				"Which manager option lets only one replica act?",
				"What does the Lease object record?",
			},
			Question:    "Why do multiple operator replicas need leader election, and how does a Lease decide which replica acts?",
			FixCriteria: "TestTwoReplicasDoNotDuplicateJobs passes with leader election on.",
		},
	},
}

// ThemeFor returns the theme of a project.
func ThemeFor(projectSlug string) (Theme, bool) {
	t, ok := themes[projectSlug]
	return t, ok
}

// ContentFor returns the scene and bug of a stage.
func ContentFor(projectSlug, stageTitle string) (StageContent, bool) {
	sc, ok := stageContents[projectSlug+"/"+stageTitle]
	return sc, ok
}

// stageContentKeys lists every key in stageContents, for consistency checks.
func stageContentKeys() []string {
	keys := make([]string, 0, len(stageContents))
	for k := range stageContents {
		keys = append(keys, k)
	}
	return keys
}
