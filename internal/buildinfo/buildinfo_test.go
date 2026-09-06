package buildinfo

import (
	"bytes"
	"log"
	"os"
	"testing"
)

func stamp(t *testing.T, v string) {
	t.Helper()
	prev := version
	version = v
	t.Cleanup(func() { version = prev })
}

func TestVersionPrecedence(t *testing.T) {
	tests := []struct {
		name  string
		stamp string
		env   string
		envOK bool
		want  string
	}{
		{name: "unstamped and no env reports dev", want: "dev"},
		{name: "unstamped reads the env", env: "9.9.9", envOK: true, want: "9.9.9"},
		{name: "the stamp wins with no env", stamp: "1.2.3", want: "1.2.3"},
		{name: "the stamp beats the env", stamp: "1.2.3", env: "9.9.9", envOK: true, want: "1.2.3"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stamp(t, tc.stamp)
			t.Setenv("VERGE_VERSION", tc.env)
			if !tc.envOK {
				os.Unsetenv("VERGE_VERSION")
			}
			if got := Version(); got != tc.want {
				t.Errorf("Version() = %q, want %q", got, tc.want)
			}
			if got := Stamped(); got != (tc.stamp != "") {
				t.Errorf("Stamped() = %v, want %v", got, tc.stamp != "")
			}
		})
	}
}

func TestVersionLogsNothing(t *testing.T) {
	var buf bytes.Buffer
	prevOut, prevFlags, prevPrefix := log.Writer(), log.Flags(), log.Prefix()
	log.SetOutput(&buf)
	t.Cleanup(func() {
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
		log.SetPrefix(prevPrefix)
	})

	t.Setenv("VERGE_VERSION", "9.9.9")
	for _, s := range []string{"", "1.2.3"} {
		stamp(t, s)
		_ = Version()
		_ = Stamped()
	}

	if buf.Len() != 0 {
		t.Errorf("Version() wrote %q, want nothing", buf.String())
	}
}
