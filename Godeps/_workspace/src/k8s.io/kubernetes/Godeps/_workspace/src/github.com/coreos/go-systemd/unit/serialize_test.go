package unit

import (
	"io/ioutil"
	"testing"
)

func TestSerialize(t *testing.T) {
	tests := []struct {
		input  []*UnitOption
		output string
	}{
		// no options results in empty file
		{
			[]*UnitOption{},
			``,
		},

		// options with same section share the header
		{
			[]*UnitOption{
				&UnitOption{"Unit", "Description", "Foo"},
				&UnitOption{"Unit", "BindsTo", "bar.service"},
			},
			`[Unit]
Description=Foo
BindsTo=bar.service
`,
		},

		// options with same name are not combined
		{
			[]*UnitOption{
				&UnitOption{"Unit", "Description", "Foo"},
				&UnitOption{"Unit", "Description", "Bar"},
			},
			`[Unit]
Description=Foo
Description=Bar
`,
		},

		// multiple options printed under different section headers
		{
			[]*UnitOption{
				&UnitOption{"Unit", "Description", "Foo"},
				&UnitOption{"Service", "ExecStart", "/usr/bin/sleep infinity"},
			},
			`[Unit]
Description=Foo

[Service]
ExecStart=/usr/bin/sleep infinity
`,
		},

		// no optimization for unsorted options
		{
			[]*UnitOption{
				&UnitOption{"Unit", "Description", "Foo"},
				&UnitOption{"Service", "ExecStart", "/usr/bin/sleep infinity"},
				&UnitOption{"Unit", "BindsTo", "bar.service"},
			},
			`[Unit]
Description=Foo

[Service]
ExecStart=/usr/bin/sleep infinity

[Unit]
BindsTo=bar.service
`,
		},

		// utf8 characters are not a problem
		{
			[]*UnitOption{
				&UnitOption{"©", "µ☃", "ÇôrèÕ$"},
			},
			`[©]
µ☃=ÇôrèÕ$
`,
		},

		// no verification is done on section names
		{
			[]*UnitOption{
				&UnitOption{"Un\nit", "Description", "Foo"},
			},
			`[Un
it]
Description=Foo
`,
		},

		// no verification is done on option names
		{
			[]*UnitOption{
				&UnitOption{"Unit", "Desc\nription", "Foo"},
			},
			`[Unit]
Desc
ription=Foo
`,
		},

		// no verification is done on option values
		{
			[]*UnitOption{
				&UnitOption{"Unit", "Description", "Fo\no"},
			},
			`[Unit]
Description=Fo
o
`,
		},
	}

	for i, tt := range tests {
		outReader := Serialize(tt.input)
		outBytes, err := ioutil.ReadAll(outReader)
		if err != nil {
			t.Errorf("case %d: encountered error while reading output: %v", i, err)
			continue
		}

		output := string(outBytes)
		if tt.output != output {
			t.Errorf("case %d: incorrect output")
			t.Logf("Expected:\n%s", tt.output)
			t.Logf("Actual:\n%s", output)
		}
	}
}
