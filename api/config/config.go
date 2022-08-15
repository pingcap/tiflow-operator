package config

import (
	"bytes"
	stdjson "encoding/json"
	"reflect"

	"github.com/BurntSushi/toml"
	"github.com/mohae/deepcopy"
	"github.com/pingcap/errors"
	"k8s.io/apimachinery/pkg/util/json"
)

type GenericConfig struct {
	// Export this field to make "apiequality.Semantic.DeepEqual" happy now.
	// User of GenericConfig should not directly access this field.
	MP map[string]interface{}
}

var _ stdjson.Marshaler = &GenericConfig{}
var _ stdjson.Unmarshaler = &GenericConfig{}

func (c *GenericConfig) MarshalTOML() ([]byte, error) {
	if c == nil {
		return nil, nil
	}

	buff := new(bytes.Buffer)
	encoder := toml.NewEncoder(buff)
	err := encoder.Encode(c.MP)
	if err != nil {
		return nil, errors.AddStack(err)
	}

	return buff.Bytes(), nil
}

func (c *GenericConfig) UnmarshalTOML(data []byte) error {
	return toml.Unmarshal(data, &c.MP)
}

func (c *GenericConfig) MarshalJSON() ([]byte, error) {
	toml, err := c.MarshalTOML()
	if err != nil {
		return nil, err
	}

	return json.Marshal(string(toml))
}

func (c *GenericConfig) UnmarshalJSON(data []byte) error {
	var value interface{}
	err := json.Unmarshal(data, &value)
	if err != nil {
		return errors.AddStack(err)
	}

	switch s := value.(type) {
	case string:
		err = toml.Unmarshal([]byte(s), &c.MP)
		if err != nil {
			return errors.AddStack(err)
		}
		return nil
	case map[string]interface{}:
		// If v is a *map[string]interface{}, numbers are converted to int64 or float64
		// using s directly all numbers are float type of go
		// Keep the behavior Unmarshal *map[string]interface{} directly we unmarshal again here.
		err = json.Unmarshal(data, &c.MP)
		if err != nil {
			return errors.AddStack(err)
		}
		return nil
	default:
		return errors.Errorf("unknown type: %v", reflect.TypeOf(value))
	}
}

func New(o map[string]interface{}) *GenericConfig {
	return &GenericConfig{o}
}

func (c *GenericConfig) Inner() map[string]interface{} {
	return c.MP
}

func (c *GenericConfig) DeepCopyJsonObject() *GenericConfig {
	if c == nil {
		return nil
	}
	if c.MP == nil {
		return New(nil)
	}

	MP := deepcopy.Copy(c.MP).(map[string]interface{})
	return New(MP)
}

func (c *GenericConfig) DeepCopy() *GenericConfig {
	return c.DeepCopyJsonObject()
}

func (c *GenericConfig) DeepCopyInto(out *GenericConfig) {
	*out = *c
	out.MP = c.DeepCopyJsonObject().MP
}
