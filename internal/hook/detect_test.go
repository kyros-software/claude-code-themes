package hook

import "testing"

// The detection heuristics are the highest-churn, highest-risk code here. Three
// of the last five commits before the rewrite touched this logic, always for
// the same reason - a pattern matching somewhere it had no business matching.
// The tables are the regression net.

var runnerYes = []string{
	"pytest -q",
	"npm test",
	"npm run test -- --watch=false",
	"yarn test",
	"pnpm run test",
	"cargo test",
	"cargo nextest run",
	"go test ./...",
	"FOO=1 sudo cargo nextest run",
	"cd /tmp && pytest",
	"make test",
	"just test-all",
	"mvn verify",
	"gradle test",
	"dotnet test",
	"./run-tests.sh",
	"bin/spec",
	"./testear.sh",
	"nix flake check",
	"ls && pytest",
	// reached by peeling launchers off the front
	"python -m pytest tests/",
	"python3 -m pytest",
	"uv run pytest",
	"poetry run pytest -q",
	"npx vitest run",
	"npx -y jest",
	"bundle exec rspec",
	"env RUST_LOG=debug cargo test",
	"sudo env FOO=1 cargo test",
	// reached by matching argv[0] on its basename too
	"./gradlew test",
	"/usr/local/bin/pytest -q",
}

var runnerNo = []string{
	// A runner named inside a string is not a run.
	`echo '{"cmd":"pytest"}'`,
	"grep -rn pytest .",
	"cat pytest.ini",
	"git commit -m 'add pytest'",
	// Shell's `test` compares files, it is not a suite.
	"test -f x && echo yes",
	"[ -f x ]",
	"rm -rf node_modules",
	"python manage.py runserver",
	"npx create-react-app mine",
	"",
	"   ",
}

// Still missed on purpose: documentation, not a wish list. The rule is to
// rather miss a meal than invent one, and widening a pattern to catch these is
// a deliberate change, not a bug fix.
var runnerKnownMisses = []string{
	`gradlew testDebug`, // the pattern ends at \w*test\b
	"make check",        // "check" is not "test"
}

var commitYes = []string{
	"git commit -m 'x'",
	"git -C /tmp commit -m x",
	"cd /repo && git commit -am wip",
	"GIT_AUTHOR_NAME=x git commit -m y",
	"sudo git commit -m y",
	"ls; git commit -m y",
}

var commitNo = []string{
	// The word in a search is not a commit.
	`grep -rn 'git commit' .`,
	"echo git commit",
	"git commit --dry-run -m x",
	"git log --oneline",
	"man git-commit",
}

func TestRunnerDetected(t *testing.T) {
	for _, cmd := range runnerYes {
		if !IsTestRunner(cmd, nil) {
			t.Errorf("IsTestRunner(%q) = false, want true", cmd)
		}
	}
}

func TestRunnerNotDetected(t *testing.T) {
	for _, cmd := range runnerNo {
		if IsTestRunner(cmd, nil) {
			t.Errorf("IsTestRunner(%q) = true, want false", cmd)
		}
	}
}

func TestKnownMissesStayMissed(t *testing.T) {
	for _, cmd := range runnerKnownMisses {
		if IsTestRunner(cmd, nil) {
			t.Errorf("IsTestRunner(%q) = true; widening a pattern is a conscious act", cmd)
		}
	}
}

func TestCommitDetected(t *testing.T) {
	for _, cmd := range commitYes {
		if !IsCommit(cmd) {
			t.Errorf("IsCommit(%q) = false, want true", cmd)
		}
	}
}

func TestCommitNotDetected(t *testing.T) {
	for _, cmd := range commitNo {
		if IsCommit(cmd) {
			t.Errorf("IsCommit(%q) = true, want false", cmd)
		}
	}
}

func TestPrefixPeelingTerminatesOnPathologicalInput(t *testing.T) {
	long := ""
	for i := 0; i < 200; i++ {
		long += "env "
	}
	IsTestRunner(long+"pytest", nil)
	assignments := ""
	for i := 0; i < 500; i++ {
		assignments += "A=1 "
	}
	IsTestRunner(assignments, nil)
}

func TestRedOnlyReadFromTheTail(t *testing.T) {
	// A test named test_login_failed must not paint a green suite red.
	green := ""
	for i := 0; i < 30; i++ {
		green += "test_login_failed PASSED\n"
	}
	green += "5 passed in 1.2s"
	if TestsAreRed(green, false) {
		t.Error("a green suite was painted red by a test's name")
	}
}

func TestRedWhenTheSummarySaysSo(t *testing.T) {
	for _, output := range []string{
		"3 passed\n1 failed in 0.3s",
		"FAILED tests/test_x.py::test_y",
		"panic: runtime error",
	} {
		if !TestsAreRed(output, false) {
			t.Errorf("TestsAreRed(%q) = false", output)
		}
	}
}

func TestIsErrorBeatsAnyPattern(t *testing.T) {
	// The CLI's own flag is the only hard datum: it stands in for the exit code.
	if !TestsAreRed("everything is fine, 9 passed", true) {
		t.Error("is_error did not win")
	}
}

func TestExtraRunnersFromTheEnvironment(t *testing.T) {
	t.Setenv("PET_TEST_RUNNERS", `\bmytestrunner\b`)
	ResetRunnersPattern()
	defer ResetRunnersPattern()
	if !IsTestRunner("mytestrunner --all", RunnersPattern()) {
		t.Error("PET_TEST_RUNNERS was ignored")
	}
}

func TestABrokenExtraRegexIsIgnored(t *testing.T) {
	t.Setenv("PET_TEST_RUNNERS", "([unclosed")
	ResetRunnersPattern()
	defer ResetRunnersPattern()
	if !IsTestRunner("pytest", RunnersPattern()) {
		t.Error("a broken PET_TEST_RUNNERS took the built-in list down with it")
	}
}

func TestCommitSummaryIsRead(t *testing.T) {
	output := "[main a1b2c3] a message\n 3 files changed, 4 insertions(+)"
	if n, ok := FilesChangedCount(output); !ok || n != 3 {
		t.Errorf("FilesChangedCount = %d,%v", n, ok)
	}
	if ref := CommitRef(output); ref != "main" {
		t.Errorf("CommitRef = %q", ref)
	}
	// Only git's own line: the first bracket of whatever else the command
	// printed is not a branch, and it used to be booked as one.
	for _, noise := range []string{
		"array [$companyAdmin->id => 3]\n[feature/x 9f8e7d6] msg",
		"\x1b[30;42mOK\x1b[0m\n[main (root-commit) abc1234] first",
	} {
		if ref := CommitRef(noise); ref != "feature/x" && ref != "main" {
			t.Errorf("CommitRef(%q) = %q", noise, ref)
		}
	}
	if ref := CommitRef("['tengo hambre']\nOK"); ref != "" {
		t.Errorf("CommitRef read %q out of output with no commit line", ref)
	}
	if !SaidNothingCommitted("nothing to commit, working tree clean") {
		t.Error("the empty commit was not recognised")
	}
	if !SaidNothingCommitted("nada que confirmar") {
		t.Error("git's Spanish was not recognised")
	}
}

func TestDocsCommitByWhatGitSays(t *testing.T) {
	docs := "3\t1\tREADME.md\n2\t0\tdocs/guide.md\n"
	mixed := "3\t1\tREADME.md\n40\t2\tsrc/app.py\n5\t1\tsrc/lib.py\n"
	deletion := "0\t120\tsrc/dead.py\n"
	if !IsDocsCommit(docs) {
		t.Error("a docs commit was not recognised")
	}
	if IsDocsCommit(mixed) {
		t.Error("a mixed commit counted as docs")
	}
	if !IsDocsCommit(deletion) {
		t.Error("a big net deletion did not count as cleanup")
	}
}
