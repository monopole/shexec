package shexec_test

import (
	"fmt"

	. "github.com/monopole/shexec"
	"github.com/monopole/shexec/channeler"
)

const unlikelyWord = "supercalifragilisticexpialidocious"

// An example using /bin/sh, a shell that's available on most platforms.
func Example_binSh() {
	sh := NewShell(Parameters{
		Params: channeler.Params{
			Path: "/bin/sh",
		},
		SentinelOut: Sentinel{
			C: "echo " + unlikelyWord,
			V: unlikelyWord,
		},
	})
	err := sh.Start(timeOutShort)
	assertNoErr(err)
	assertNoErr(sh.Run(timeOutShort,
		NewLabellingCommander(`
echo alpha
which cat
`)))
	assertNoErr(sh.Run(timeOutShort,
		NewLabellingCommander(`
echo beta
which find
`,
		)))
	assertNoErr(sh.Stop(timeOutShort, ""))

	// Output:
	// out: alpha
	// out: /usr/bin/cat
	// out: beta
	// out: /usr/bin/find
}

// The tests below require the "conch" shell.
// As written, they require that
// * The `go` program is installed.
// * Tests are run from the top of the repo, such that ./conch is below you.

var (
	// The version command is a good stdOut sentinel for conch.
	sentinelVersion = Sentinel{
		C: "version",
		V: "v1.2.3",
	}

	// An unknown command is a good stdErr sentinel for conch.
	sentinelUnknownCommand = Sentinel{
		C: unlikelyWord,
		V: `unrecognized command: "` + unlikelyWord + `"`,
	}
)

// An error free run using the (locally defined) conch shell.
func Example_basicRun() {
	sh := NewShell(Parameters{
		Params: channeler.Params{
			WorkingDir: "./conch",
			Path:       "go",
			Args: []string{
				"run", ".",
				// the prompt goes to stdout, so get rid of it in tests.
				"--disable-prompt",
			}},
		SentinelOut: sentinelVersion,
	})
	assertNoErr(sh.Start(timeOutShort))
	assertNoErr(sh.Run(timeOutShort, NewLabellingCommander("query limit 3")))
	assertNoErr(sh.Stop(timeOutShort, ""))

	// Output:
	// out: Cempedak_|_Bamberga_|_4_|_00000000000000000000000000000001
	// out: Buddha's hand_|_Hermione_|_6_|_00000000000000000000000000000002
	// out: African cucumber_|_Ursula_|_6_|_00000000000000000000000000000003
}

// A shell that crashes on startup.
func Example_subprocessFailOnStartup() {
	sh := NewShell(Parameters{
		Params: channeler.Params{
			WorkingDir: "./conch",
			Path:       "go",
			Args: []string{
				"run", ".",
				"--fail-on-startup",
			},
		},
		SentinelOut: sentinelVersion,
	})
	err := sh.Start(timeOutShort)
	fmt.Println(err.Error())

	// Output:
	// shexec infra; stdOut closed before sentinel "v1.2.3" found
}

// A command takes too long and fails as a result.
func Example_subprocessTakesTooLong() {
	sh := NewShell(Parameters{
		Params: channeler.Params{
			WorkingDir: "./conch",
			Path:       "go",
			Args: []string{
				"run", ".",
				"--disable-prompt",
			}},
		SentinelOut: sentinelVersion,
	})

	assertNoErr(sh.Start(timeOutShort))

	// Send in a sleep command that consumes twice the timeOut.
	err := sh.Run(
		timeOutShort,
		NewLabellingCommander("sleep "+(2*timeOutShort).String()))
	fmt.Println(err.Error())

	// Output:
	// shexec infra; running "sleep 1.6s", no sentinels found after 800ms
}

// A shell spits output to stderr.
func Example_subprocessSurvivableError() {
	sh := NewShell(Parameters{
		Params: channeler.Params{
			WorkingDir: "./conch",
			Path:       "go",
			Args: []string{
				"run", ".",
				"--disable-prompt",
				"--row-to-error-on", "4",
			}},
		SentinelOut: sentinelVersion,
		SentinelErr: sentinelUnknownCommand,
	})
	assertNoErr(sh.Start(timeOutShort))

	cmdr := NewLabellingCommander("query limit 3")

	// The following yields three lines.
	assertNoErr(sh.Run(timeOutShort, cmdr))

	// Query again, but ask for a row beyond the row that
	// triggers a DB error.
	// Because of the nature of output streams,
	// there's no way to know
	// when the error will show up in the combined output.
	// It might come out first, last, or anywhere in the
	// middle relative to lines from stdOut,
	// so this test must not be fragile to the order.
	// This will yield three "good lines", and one error line.
	cmdr.C = "query limit 7"
	assertNoErr(sh.Run(timeOutShort, cmdr))

	// Yields two lines.
	cmdr.C = "query limit 2"
	assertNoErr(sh.Run(timeOutShort, cmdr))

	assertNoErr(sh.Stop(timeOutShort, ""))

	// There should be nine (3 + 3 + 1 + 2) lines in the output.

	// Unordered output:
	// out: Cempedak_|_Bamberga_|_4_|_00000000000000000000000000000001
	// out: Buddha's hand_|_Hermione_|_6_|_00000000000000000000000000000002
	// out: African cucumber_|_Ursula_|_6_|_00000000000000000000000000000003
	// out: Currant_|_Alauda_|_5_|_00000000000000000000000000000001
	// out: Banana_|_Egeria_|_5_|_00000000000000000000000000000002
	// out: Bilberry_|_Interamnia_|_2_|_00000000000000000000000000000003
	// err: error! touching row 4 triggers this error
	// out: Cherimoya_|_Palma_|_6_|_00000000000000000000000000000001
	// out: Abiu_|_Metis_|_3_|_00000000000000000000000000000002
}

// A shell that crashes, and is then restarted.
func Example_subprocessNonSurvivableError() {
	sh := NewShell(Parameters{
		Params: channeler.Params{
			WorkingDir: "./conch",
			Path:       "go",
			Args: []string{
				"run", ".",
				"--disable-prompt",
				// Using this means any error will cause process exit.
				// So we cannot use an errSentinel, as it by definition causes an error.
				"--exit-on-error",
				"--row-to-error-on", "4",
			}},
		SentinelOut: sentinelVersion,
	})
	assertNoErr(sh.Start(timeOutShort))

	cmdr := NewLabellingCommander("query limit 3")

	// The following yields three lines.
	assertNoErr(sh.Run(timeOutShort, cmdr))

	// Query again, but ask for a row beyond the row that
	// triggers a DB error.
	// Since flag "exit-on-error" is enabled, this causes the CLI to die.
	cmdr.C = "query limit 5"
	err := sh.Run(timeOutShort, cmdr)
	assertErr(err)
	fmt.Println(err.Error())

	err = sh.Stop(timeOutShort, "")
	assertErr(err)
	fmt.Println(err.Error())

	// Start it up again and issue a command to show that it works.
	assertNoErr(sh.Start(timeOutShort))
	cmdr.C = "query limit 2"
	assertNoErr(sh.Run(timeOutShort, cmdr))
	assertNoErr(sh.Stop(timeOutShort, ""))

	// Output:
	// out: Cempedak_|_Bamberga_|_4_|_00000000000000000000000000000001
	// out: Buddha's hand_|_Hermione_|_6_|_00000000000000000000000000000002
	// out: African cucumber_|_Ursula_|_6_|_00000000000000000000000000000003
	// out: Currant_|_Alauda_|_5_|_00000000000000000000000000000001
	// out: Banana_|_Egeria_|_5_|_00000000000000000000000000000002
	// out: Bilberry_|_Interamnia_|_2_|_00000000000000000000000000000003
	// shexec infra; stdOut closed before sentinel "v1.2.3" found
	// shexec infra; stop called, but shell not started yet
	// out: Cempedak_|_Bamberga_|_4_|_00000000000000000000000000000001
	// out: Buddha's hand_|_Hermione_|_6_|_00000000000000000000000000000002
}
