package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type Simple struct {
	A string
	B int
}

func TestDeepCopyJsonObject(t *testing.T) {
	objects := []*GenericConfig{
		New(nil),
		New(map[string]interface{}{
			"k1": true,
			"k2": "v2",
			"k3": 1,
			"k4": 'v',
			"k5": []byte("v5"),
			"k6": nil,
		}),
		New(map[string]interface{}{
			"k1": map[string]interface{}{
				"nest-1": map[string]interface{}{
					"nest-2": map[string]interface{}{
						"nest-3": "internal",
					},
				},
			},
		}),
		New(map[string]interface{}{
			"k1": map[string]interface{}{
				"nest-1": &Simple{
					A: "xx",
					B: 1,
				},
				"nest-2": Simple{
					A: "xx",
					B: 2,
				},
			},
		}),
	}

	for _, obj := range objects {
		copied := obj.DeepCopy()
		require.Equal(t, obj, copied)

		out := New(nil)
		obj.DeepCopyInto(out)
		require.Equal(t, obj, out)
	}
	copied := objects[1].DeepCopy()
	copied.Inner()["k1"] = false
	require.Equal(t, true, objects[1].Inner()["k1"], "Mutation copy should net affect origin")
}

func TestMarshalTOML(t *testing.T) {
	c := New(map[string]interface{}{
		"int":       int64(1),
		"float":     1.0,
		"str":       "str",
		"str_slice": []interface{}{"s1", "s2"},
	})

	data, err := c.MarshalTOML()
	require.NoError(t, err)
	t.Log("toml: ", string(data))

	cback := New(nil)
	err = cback.UnmarshalTOML(data)
	require.NoError(t, err)
	require.Equal(t, c, cback)
}

func TestMarshalJSON(t *testing.T) {
	type S struct {
		Config *GenericConfig `json:"config,omitempty"`
	}

	s := &S{
		Config: New(map[string]interface{}{}),
	}
	s.Config.MP["sk"] = "v"
	s.Config.MP["ik"] = int64(1)

	// test string type
	data, err := json.Marshal(s)
	require.NoError(t, err)

	sback := new(S)
	err = json.Unmarshal(data, sback)
	require.NoError(t, err)
	require.Equal(t, s, sback)

	// test object type
	data, err = json.Marshal(map[string]interface{}{
		"config": s.Config.MP,
	})
	require.Nil(t, err)

	sback = new(S)
	err = json.Unmarshal(data, sback)
	require.NoError(t, err)
	require.Equal(t, s, sback)
}

func TestJsonOmitempty(t *testing.T) {
	type S struct {
		Config *GenericConfig `json:"config,omitempty"`
	}

	// test Config should be nil
	s := new(S)
	err := json.Unmarshal([]byte("{}"), s)
	require.NoError(t, err)
	require.Nil(t, s.Config)
	data, err := json.Marshal(s)
	require.NoError(t, err)
	s = new(S)
	err = json.Unmarshal(data, s)
	require.NoError(t, err)
	require.Nil(t, s.Config)

	// test Config should not be nil
	s = new(S)
	err = json.Unmarshal([]byte("{\"config\":\"a = 1\"}"), s)
	require.NoError(t, err)
	require.NotNil(t, s.Config)
	data, err = json.Marshal(s)
	require.NoError(t, err)
	s = new(S)
	err = json.Unmarshal(data, s)
	require.NoError(t, err)
	require.NotNil(t, s.Config)
}
