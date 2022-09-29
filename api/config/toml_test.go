package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMarshal(t *testing.T) {
	mp := map[string]interface{}{
		"int":    int64(1),
		"float":  1.1,
		"string": "string",
		"object": map[string]interface{}{
			"int":    int64(1),
			"float":  1.1,
			"string": "string",
		},
	}

	data, err := Marshal(&mp)
	require.NoError(t, err)

	var mpback map[string]interface{}
	err = Unmarshal(data, &mpback)
	require.NoError(t, err)
	require.Equal(t, mp, mpback)
}

func TestEqual(t *testing.T) {
	type testcase struct {
		d1    string
		d2    string
		equal bool
	}

	tests := []*testcase{
		{
			d1:    "a = 1",
			d2:    "a = 1",
			equal: true,
		},
		{
			d1:    "a = 1",
			d2:    "a = 2",
			equal: false,
		},
		{
			d1:    "a =  1",
			d2:    "a = 1",
			equal: true,
		},
		{
			d1:    "a =  1",
			d2:    "a = 2",
			equal: false,
		},
		{
			d1:    "[user]\n[user.default]\np = 'ok'",
			d2:    "[user.default]\np = 'ok'",
			equal: true,
		},
	}

	for _, test := range tests {
		equal, err := Equal([]byte(test.d1), []byte(test.d2))
		require.NoError(t, err)
		t.Logf("check '%s' and '%s'", test.d1, test.d2)
		require.Equal(t, test.equal, equal)
	}
}
