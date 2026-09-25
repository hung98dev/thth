// Package stackpin holds the canonical version pins from
// docs/00_context/technology_versions.md and the Q1 gate checks that verify
// materialized files match them exactly. The matrix document is the source of
// truth; this package is its executable mirror.
package stackpin

const (
	GoVersion   = "1.27.1"
	UnityEditor = "6000.6.1f1"

	PostgreSQL     = "18.6"
	PostgresImage  = "postgres:18.6"
	PostgresDigest = "sha256:5a5a84b19854a9ffaa54082c166ff4ec27473a361e496e5ea167f298f2da9722"

	Protoc         = "36.2"
	ProtocGenGo    = "v1.36.12"
	ProtobufCSharp = "3.36.2"
	ProtobufGo     = "v1.36.12"

	Pwsh              = "7.6.6"
	Jq                = "1.8.2"
	GhCli             = "2.101.0"
	GitLfs            = "3.8.0"
	Staticcheck       = "2026.2.1" // module honnef.co/go/tools v0.8.1
	StaticcheckModule = "v0.8.1"

	PGX       = "v5.11.0"
	Websocket = "v1.8.15"
	Migrate   = "v4.20.1"
	OTel      = "v1.46.0"
	OTelHTTP  = "v0.71.0"
	XText     = "v0.42.0"
	XCrypto   = "v0.57.0"
	Uax29     = "v2.7.0"
)

// CLI release assets installed by verify.yml bootstrap steps. Preinstalled
// tools on hosted runners are never used (technology_versions.md).
var CliAssets = map[string]CliAsset{
	"pwsh": {
		URL:    "https://github.com/PowerShell/PowerShell/releases/download/v" + Pwsh + "/powershell-" + Pwsh + "-linux-x64.tar.gz",
		SHA256: "ddbc4a2d113bbd46d283cfedcbcd117a70caefd7673f41f2b4e0000badf103bc",
	},
	"pwsh-windows": {
		URL:    "https://github.com/PowerShell/PowerShell/releases/download/v" + Pwsh + "/PowerShell-" + Pwsh + "-win-x64.zip",
		SHA256: "02fe458be20493fbdf43f61ea20610b811ee6c738ab1676c61b9cfcd1a33c860",
	},
	"jq": {
		URL:    "https://github.com/jqlang/jq/releases/download/jq-" + Jq + "/jq-linux-amd64",
		SHA256: "b1c22172dd303f3be49e935aa56aa48a8b7a46e0bc838b4997d3bb451495870f",
	},
	"jq-windows": {
		URL:    "https://github.com/jqlang/jq/releases/download/jq-" + Jq + "/jq-windows-amd64.exe",
		SHA256: "a6fc67fedaf9128a3309a1e2ebb8b986aeccf70122ee46d2cb4849e423f0c627",
	},
	"gh": {
		URL:    "https://github.com/cli/cli/releases/download/v" + GhCli + "/gh_" + GhCli + "_linux_amd64.tar.gz",
		SHA256: "9bca2d1c16825f109907a23307628a2f0698fbf99662b73a5cf0b020293072b8",
	},
	"gh-windows": {
		URL:    "https://github.com/cli/cli/releases/download/v" + GhCli + "/gh_" + GhCli + "_windows_amd64.zip",
		SHA256: "bc6c814367b193cd8e713611d61e36013c0ef843b8f516458fe3eda039192794",
	},
	"git-lfs": {
		URL:    "https://github.com/git-lfs/git-lfs/releases/download/v" + GitLfs + "/git-lfs-linux-amd64-v" + GitLfs + ".tar.gz",
		SHA256: "e455e00f15d9b95661b8d53498ffb0c3367962cf1ec73c31ab7369516cd6ab8d",
	},
	"git-lfs-windows": {
		URL:    "https://github.com/git-lfs/git-lfs/releases/download/v" + GitLfs + "/git-lfs-windows-amd64-v" + GitLfs + ".zip",
		SHA256: "b62e7b8ceddee635f691233d77de8eaa4b213e9209e0173811d8cfa77f7882c1",
	},
}

// CliAsset is one pinned release asset: download URL + SHA-256 the CI
// bootstrap verifies before use.
type CliAsset struct {
	URL    string
	SHA256 string
}

// PostgresTestAssets pins the PostgreSQL test servers: the linux docker
// service image digest and the Windows EDB binaries zip.
var PostgresTestAssets = map[string]string{
	"linux-image-digest": PostgresDigest,
	"windows-edb-zip":    "fbe23da234ee31547bf8a36d29dfd81e82b849df2d2b78d2eecb43d360252f8c",
}

// EdbZipURL is the EDB Windows binaries download (postgresql-18.6-1).
const EdbZipURL = "https://get.enterprisedb.com/postgresql/postgresql-18.6-1-windows-x64-binaries.zip"
const EdbZipSHA256 = "fbe23da234ee31547bf8a36d29dfd81e82b849df2d2b78d2eecb43d360252f8c"

// GoogleProtobufNupkgURL/SHA256 pin the Google.Protobuf 3.36.2 NuGet package;
// lib/netstandard2.0/Google.Protobuf.dll is extracted and committed to
// client/Assets/Plugins/Google.Protobuf/ (technology_versions.md).
const GoogleProtobufNupkgURL = "https://api.nuget.org/v3-flatcontainer/google.protobuf/3.36.2/google.protobuf.3.36.2.nupkg"
const GoogleProtobufNupkgSHA256 = "1182590db175f9057707857a1df48b217226d0732716cd353fa4aa4683d38dcb"

// UnityImages are the pinned unityci editor images used by verify.yml.
var UnityImages = map[string]string{
	"linux":          "unityci/editor:ubuntu-6000.6.1f1-base-3.2.2@sha256:2197a718c75ba71d6d9a05cfdfbce31cc401113f530963ac789160dffc96763d",
	"windows":        "unityci/editor:windows-6000.6.1f1-base-3.2.2@sha256:a995b9d1d03dc08c1702f91acc05c64297217522aebb9387af7ce912331fb534",
	"android":        "unityci/editor:ubuntu-6000.6.1f1-android-3.2.2@sha256:33f6f1056b02dcabd46ed9bfb8ff26aae241e0af412f9628bc06fc760df248ab",
	"windows-il2cpp": "unityci/editor:windows-6000.6.1f1-windows-il2cpp-3.2.2@sha256:5bd80a61ac442b81745f653dd39395f6e93167ebc51c4b494bdd42c2b656195b",
}

// RunnerLabels is the closed set of allowed runs-on labels (ADR-0058:
// GitHub-hosted ubuntu-24.04 + windows-2022 only; never *-latest, never
// self-hosted/GPU/larger).
var RunnerLabels = map[string]bool{
	"ubuntu-24.04": true,
	"windows-2022": true,
}

// ActionPin pairs an approved GitHub Action with its required commit SHA and
// release tag (ADR-0010: pin tag and commit SHA).
type ActionPin struct {
	SHA string
	Tag string
}

// GitHubActions is the approved set of GitHub Actions usable in workflows.
var GitHubActions = map[string]ActionPin{
	"actions/checkout":                {SHA: "11bd71901bbe5b1630ceea73d27597364c9af683", Tag: "v4.2.2"},
	"actions/setup-go":                {SHA: "f111f3307d8850f501ac008e886eec1fd1932a34", Tag: "v5.3.0"},
	"actions/upload-artifact":         {SHA: "ea165f8d65b6e75b540449e92b4886f43607fa02", Tag: "v4.6.2"},
	"actions/download-artifact":       {SHA: "d3f86a106a0bac45b974a628896c90dbdf5c8093", Tag: "v4.3.0"},
	"actions/cache":                   {SHA: "55cc8345863c7cc4c66a329aec7e433d2d1c52a9", Tag: "v6.1.0"},
	"game-ci/unity-test-runner":       {SHA: "fa6ced25861c16ef56187828c43f76d00df43a23", Tag: "v4.3.2"},
	"game-ci/unity-builder":           {SHA: "eb1b9fba120c6e62c9fb7a7a81d6c107ce004c45", Tag: "v6.0.0"},
	"actions/create-github-app-token": {SHA: "bcd2ba49218906704ab6c1aa796996da409d3eb1", Tag: "v3.2.0"},
}

// GoModulePins is the canonical go.mod direct-dependency allowlist. Any
// module required by server/go.mod must appear here at this exact version.
var GoModulePins = map[string]string{
	"github.com/jackc/pgx/v5":                                           PGX,
	"github.com/coder/websocket":                                        Websocket,
	"github.com/golang-migrate/migrate/v4":                              Migrate,
	"google.golang.org/protobuf":                                        ProtobufGo,
	"go.opentelemetry.io/otel":                                          OTel,
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp":     OTelHTTP,
	"go.opentelemetry.io/otel/sdk":                                      OTel,
	"go.opentelemetry.io/otel/sdk/metric":                               OTel,
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp":   OTel,
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp": OTel,
	"golang.org/x/text":                                                 XText,
	"golang.org/x/crypto":                                               XCrypto,
	"github.com/clipperhouse/uax29/v2":                                  Uax29,
}

// ForbiddenModules must never appear in server/go.mod (AGENTS.md forbidden
// dependency list).
var ForbiddenModules = []string{
	"github.com/gorilla/websocket",
	"github.com/gorilla/mux",
	"google.golang.org/grpc",
	"gorm.io/gorm",
	"github.com/jmoiron/sqlx",
	"go.uber.org/zap",
	"github.com/sirupsen/logrus",
	"github.com/rs/zerolog",
	"github.com/redis/go-redis/v9",
	"github.com/go-redis/redis/v8",
	"github.com/segmentio/kafka-go",
	"github.com/nats-io/nats.go",
	"github.com/gin-gonic/gin",
	"github.com/go-chi/chi/v5",
	"github.com/labstack/echo/v4",
	"github.com/gofiber/fiber/v2",
}

// UnityPackages is the canonical com.unity.* package set for
// client/Packages/manifest.json (Unity 6000.6.1f1-aligned versions).
var UnityPackages = map[string]string{
	"com.unity.render-pipelines.universal":   "17.6.0",
	"com.unity.render-pipelines.core":        "17.6.0",
	"com.unity.shadergraph":                  "17.6.0",
	"com.unity.inputsystem":                  "1.20.0",
	"com.unity.2d.animation":                 "16.0.0",
	"com.unity.2d.psdimporter":               "15.0.0",
	"com.unity.addressables":                 "2.11.2",
	"com.unity.localization":                 "1.5.12",
	"com.unity.test-framework.performance":   "6.6.0",
	"com.unity.memoryprofiler":               "1.1.12",
	"com.unity.performance.profile-analyzer": "1.4.0",
}
