package stackpin

import "testing"

func TestExactToolchainPins(t *testing.T) {
	cases := []struct{ name, got, want string }{
		{"GoVersion", GoVersion, "1.27.1"},
		{"UnityEditor", UnityEditor, "6000.6.1f1"},
		{"PostgreSQL", PostgreSQL, "18.6"},
		{"Protoc", Protoc, "36.2"},
		{"ProtocGenGo", ProtocGenGo, "v1.36.12"},
		{"ProtobufCSharp", ProtobufCSharp, "3.36.2"},
		{"ProtobufGo", ProtobufGo, "v1.36.12"},
		{"Pwsh", Pwsh, "7.6.6"},
		{"Jq", Jq, "1.8.2"},
		{"GhCli", GhCli, "2.101.0"},
		{"GitLfs", GitLfs, "3.8.0"},
		{"Staticcheck", Staticcheck, "2026.2.1"},
		{"StaticcheckModule", StaticcheckModule, "v0.8.1"},
		{"PGX", PGX, "v5.11.0"},
		{"Websocket", Websocket, "v1.8.15"},
		{"Migrate", Migrate, "v4.20.1"},
		{"OTel", OTel, "v1.46.0"},
		{"OTelHTTP", OTelHTTP, "v0.71.0"},
		{"XText", XText, "v0.42.0"},
		{"XCrypto", XCrypto, "v0.57.0"},
		{"Uax29", Uax29, "v2.7.0"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want canonical %q", c.name, c.got, c.want)
		}
	}
}

func TestImageDigests(t *testing.T) {
	if PostgresImage != "postgres:18.6" {
		t.Errorf("PostgresImage %q", PostgresImage)
	}
	if PostgresDigest != "sha256:5a5a84b19854a9ffaa54082c166ff4ec27473a361e496e5ea167f298f2da9722" {
		t.Errorf("PostgresDigest %q", PostgresDigest)
	}
	want := map[string]string{
		"linux":          "unityci/editor:ubuntu-6000.6.1f1-base-3.2.2@sha256:2197a718c75ba71d6d9a05cfdfbce31cc401113f530963ac789160dffc96763d",
		"windows":        "unityci/editor:windows-6000.6.1f1-base-3.2.2@sha256:a995b9d1d03dc08c1702f91acc05c64297217522aebb9387af7ce912331fb534",
		"android":        "unityci/editor:ubuntu-6000.6.1f1-android-3.2.2@sha256:33f6f1056b02dcabd46ed9bfb8ff26aae241e0af412f9628bc06fc760df248ab",
		"windows-il2cpp": "unityci/editor:windows-6000.6.1f1-windows-il2cpp-3.2.2@sha256:5bd80a61ac442b81745f653dd39395f6e93167ebc51c4b494bdd42c2b656195b",
	}
	for k, w := range want {
		if UnityImages[k] != w {
			t.Errorf("UnityImages[%q] = %q, want %q", k, UnityImages[k], w)
		}
	}
}

func TestRunnerLabelsClosed(t *testing.T) {
	if len(RunnerLabels) != 2 || !RunnerLabels["ubuntu-24.04"] || !RunnerLabels["windows-2022"] {
		t.Fatalf("RunnerLabels = %v (ADR-0058: ubuntu-24.04 + windows-2022 only)", RunnerLabels)
	}
	for label := range RunnerLabels {
		if label == "ubuntu-latest" || label == "windows-latest" {
			t.Fatalf("*-latest runner label must never be allowed: %q", label)
		}
	}
}

func TestActionPinsExact(t *testing.T) {
	want := map[string]ActionPin{
		"actions/checkout":                {SHA: "11bd71901bbe5b1630ceea73d27597364c9af683", Tag: "v4.2.2"},
		"actions/setup-go":                {SHA: "f111f3307d8850f501ac008e886eec1fd1932a34", Tag: "v5.3.0"},
		"actions/upload-artifact":         {SHA: "ea165f8d65b6e75b540449e92b4886f43607fa02", Tag: "v4.6.2"},
		"actions/download-artifact":       {SHA: "d3f86a106a0bac45b974a628896c90dbdf5c8093", Tag: "v4.3.0"},
		"actions/cache":                   {SHA: "55cc8345863c7cc4c66a329aec7e433d2d1c52a9", Tag: "v6.1.0"},
		"game-ci/unity-test-runner":       {SHA: "fa6ced25861c16ef56187828c43f76d00df43a23", Tag: "v4.3.2"},
		"game-ci/unity-builder":           {SHA: "eb1b9fba120c6e62c9fb7a7a81d6c107ce004c45", Tag: "v6.0.0"},
		"actions/create-github-app-token": {SHA: "bcd2ba49218906704ab6c1aa796996da409d3eb1", Tag: "v3.2.0"},
	}
	for action, w := range want {
		got, ok := GitHubActions[action]
		if !ok {
			t.Errorf("missing action pin %s", action)
			continue
		}
		if got != w {
			t.Errorf("%s pin = %+v, want %+v", action, got, w)
		}
	}
	if len(GitHubActions) != len(want) {
		t.Errorf("GitHubActions has %d entries, want %d", len(GitHubActions), len(want))
	}
}

func TestForbiddenDependencyList(t *testing.T) {
	forbidden := []string{
		"github.com/gorilla/websocket", "github.com/gorilla/mux",
		"google.golang.org/grpc", "gorm.io/gorm", "github.com/jmoiron/sqlx",
		"go.uber.org/zap", "github.com/sirupsen/logrus", "github.com/rs/zerolog",
		"github.com/gin-gonic/gin", "github.com/go-chi/chi/v5",
		"github.com/labstack/echo/v4", "github.com/gofiber/fiber/v2",
	}
	seen := map[string]bool{}
	for _, m := range ForbiddenModules {
		seen[m] = true
	}
	for _, f := range forbidden {
		if !seen[f] {
			t.Errorf("forbidden module %s missing from ForbiddenModules", f)
		}
	}
}

func TestGoogleProtobufNupkgSha256(t *testing.T) {
	const wantURL = "https://api.nuget.org/v3-flatcontainer/google.protobuf/3.36.2/google.protobuf.3.36.2.nupkg"
	const wantSHA = "1182590db175f9057707857a1df48b217226d0732716cd353fa4aa4683d38dcb"
	if GoogleProtobufNupkgURL != wantURL {
		t.Errorf("nupkg URL %q", GoogleProtobufNupkgURL)
	}
	if GoogleProtobufNupkgSHA256 != wantSHA {
		t.Errorf("nupkg SHA-256 %q != %q", GoogleProtobufNupkgSHA256, wantSHA)
	}
}

func TestEdbZipSha256(t *testing.T) {
	const wantURL = "https://get.enterprisedb.com/postgresql/postgresql-18.6-1-windows-x64-binaries.zip"
	const wantSHA = "fbe23da234ee31547bf8a36d29dfd81e82b849df2d2b78d2eecb43d360252f8c"
	if EdbZipURL != wantURL {
		t.Errorf("EDB zip URL %q", EdbZipURL)
	}
	if EdbZipSHA256 != wantSHA {
		t.Errorf("EDB zip SHA-256 %q != %q", EdbZipSHA256, wantSHA)
	}
	if PostgresTestAssets["windows-edb-zip"] != wantSHA {
		t.Errorf("PostgresTestAssets[windows-edb-zip] %q", PostgresTestAssets["windows-edb-zip"])
	}
}

func TestDownloadArtifactAndGitLfsPins(t *testing.T) {
	da, ok := GitHubActions["actions/download-artifact"]
	if !ok || da.SHA != "d3f86a106a0bac45b974a628896c90dbdf5c8093" || da.Tag != "v4.3.0" {
		t.Errorf("download-artifact pin %+v", da)
	}
	lfs, ok := CliAssets["git-lfs"]
	if !ok || lfs.SHA256 != "e455e00f15d9b95661b8d53498ffb0c3367962cf1ec73c31ab7369516cd6ab8d" {
		t.Errorf("git-lfs linux pin %+v", lfs)
	}
	lfsW, ok := CliAssets["git-lfs-windows"]
	if !ok || lfsW.SHA256 != "b62e7b8ceddee635f691233d77de8eaa4b213e9209e0173811d8cfa77f7882c1" {
		t.Errorf("git-lfs windows pin %+v", lfsW)
	}
}

func TestCliAssetSha256(t *testing.T) {
	want := map[string]string{
		"pwsh":            "ddbc4a2d113bbd46d283cfedcbcd117a70caefd7673f41f2b4e0000badf103bc",
		"pwsh-windows":    "02fe458be20493fbdf43f61ea20610b811ee6c738ab1676c61b9cfcd1a33c860",
		"jq":              "b1c22172dd303f3be49e935aa56aa48a8b7a46e0bc838b4997d3bb451495870f",
		"jq-windows":      "a6fc67fedaf9128a3309a1e2ebb8b986aeccf70122ee46d2cb4849e423f0c627",
		"gh":              "9bca2d1c16825f109907a23307628a2f0698fbf99662b73a5cf0b020293072b8",
		"gh-windows":      "bc6c814367b193cd8e713611d61e36013c0ef843b8f516458fe3eda039192794",
		"git-lfs":         "e455e00f15d9b95661b8d53498ffb0c3367962cf1ec73c31ab7369516cd6ab8d",
		"git-lfs-windows": "b62e7b8ceddee635f691233d77de8eaa4b213e9209e0173811d8cfa77f7882c1",
	}
	for name, sha := range want {
		asset, ok := CliAssets[name]
		if !ok {
			t.Errorf("missing CLI asset %s", name)
			continue
		}
		if asset.SHA256 != sha {
			t.Errorf("%s SHA-256 %q != %q", name, asset.SHA256, sha)
		}
		if asset.URL == "" {
			t.Errorf("%s missing download URL", name)
		}
	}
}
