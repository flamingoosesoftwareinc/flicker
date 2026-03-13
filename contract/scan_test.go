package contract

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

func TestBuildContracts(t *testing.T) {
	tests := map[string]struct {
		dir     string
		wantErr bool
	}{
		"simple":               {dir: "testdata/simple"},
		"parallel":             {dir: "testdata/parallel"},
		"providers":            {dir: "testdata/providers"},
		"wait_for_event":       {dir: "testdata/wait_for_event"},
		"sleep_until":          {dir: "testdata/sleep_until"},
		"dynamic_step_name":    {dir: "testdata/dynamic_step_name"},
		"unresolvable_factory": {dir: "testdata/unresolvable_factory"},
		"multiple_workflows":   {dir: "testdata/multiple_workflows"},
		"const_step_name":      {dir: "testdata/const_step_name"},
		"complex_types":        {dir: "testdata/complex_types"},
	}

	g := goldie.New(t,
		goldie.WithFixtureDir("testdata"),
		goldie.WithNameSuffix(".golden"),
	)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			contracts, err := BuildContracts(context.Background(), tc.dir)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			data, err := json.MarshalIndent(contracts, "", "  ")
			require.NoError(t, err)

			if os.Getenv("UPDATE_GOLDEN") != "" {
				require.NoError(t, g.Update(t, name, data))
			}

			g.Assert(t, name, data)
		})
	}
}
